#!/usr/bin/env python3
"""
Atius Antigravity Daemon (agy-daemon)
Exposes an OpenAI-compatible /v1/chat/completions endpoint backed by official agy CLI.
Runs inside isolated rootless Podman containers with dedicated egress IPs.
"""

import json
import os
import sys
import time
import uuid
import subprocess
from http.server import HTTPServer, BaseHTTPRequestHandler

PORT = int(os.environ.get("AGY_DAEMON_PORT", "8080"))
ACCOUNT_LABEL = os.environ.get("AGY_ACCOUNT_LABEL", "default")
DEFAULT_MODEL = os.environ.get("AGY_DEFAULT_MODEL", "gemini-3.8-flash-high")

SUPPORTED_MODELS = [
    "gemini-3.8-flash-high",
    "gemini-3.8-flash-medium",
    "gemini-3.8-flash-low",
    "gemini-3.7-flash-high",
    "gemini-3.7-flash-medium",
    "gemini-3.7-flash-low",
    "gemini-3.6-flash-high",
    "gemini-3.6-flash-medium",
    "gemini-3.6-flash-low",
    "gemini-3.1-pro-high",
    "gemini-3.1-pro-low",
    "claude-sonnet-4-6",
    "claude-opus-4-6-thinking",
    "gpt-oss-120b-medium",
]


def format_messages_to_prompt(messages):
    """Converts an OpenAI messages list into a coherent prompt string for agy."""
    if not messages:
        return ""
    if len(messages) == 1 and messages[0].get("role") == "user":
        content = messages[0].get("content", "")
        if isinstance(content, list):
            # Extract text parts
            parts = [p.get("text", "") for p in content if isinstance(p, dict) and p.get("type") == "text"]
            return "\n".join(parts)
        return str(content)

    formatted = []
    for msg in messages:
        role = msg.get("role", "user")
        content = msg.get("content", "")
        if isinstance(content, list):
            parts = [p.get("text", "") for p in content if isinstance(p, dict) and p.get("type") == "text"]
            text = "\n".join(parts)
        else:
            text = str(content)

        if role == "system":
            formatted.append(f"[System Instruction]\n{text}")
        elif role == "assistant":
            formatted.append(f"[Assistant]\n{text}")
        else:
            formatted.append(f"[User]\n{text}")

    return "\n\n".join(formatted)


class AgyHandler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        # Concise logging to stdout
        sys.stdout.write(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] [{ACCOUNT_LABEL}] {format % args}\n")
        sys.stdout.flush()

    def do_GET(self):
        if self.path in ("/health", "/v1/health", "/ready"):
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({
                "status": "ready",
                "account": ACCOUNT_LABEL,
                "timestamp": int(time.time()),
            }).encode("utf-8"))
            return

        if self.path == "/v1/models":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            model_entries = [
                {"id": m, "object": "model", "created": 1740000000, "owned_by": "google-antigravity"}
                for m in SUPPORTED_MODELS
            ]
            self.wfile.write(json.dumps({
                "object": "list",
                "data": model_entries
            }).encode("utf-8"))
            return

        self.send_error(404, "Endpoint not found")

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404, "Endpoint not found")
            return

        content_length = int(self.headers.get("Content-Length", 0))
        if content_length == 0:
            self.send_error(400, "Empty request body")
            return

        body = self.rfile.read(content_length)
        try:
            req_data = json.loads(body.decode("utf-8"))
        except Exception as e:
            self.send_error(400, f"Malformed JSON: {e}")
            return

        model = req_data.get("model", DEFAULT_MODEL)
        messages = req_data.get("messages", [])
        stream = req_data.get("stream", False)
        prompt = format_messages_to_prompt(messages)

        if not prompt.strip():
            self.send_error(400, "Prompt cannot be empty")
            return

        req_id = f"chatcmpl-agy-{uuid.uuid4().hex[:12]}"
        created = int(time.time())

        # Execute agy CLI with stream-json output
        cmd = [
            "agy",
            "-p", prompt,
            "--output-format", "stream-json",
            "--dangerously-skip-permissions",
        ]
        if model in SUPPORTED_MODELS:
            cmd.extend(["--model", model])

        try:
            proc = subprocess.Popen(
                cmd,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                bufsize=1,
            )
        except Exception as e:
            self.send_error(500, f"Failed to spawn agy process: {e}")
            return

        if stream:
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.send_header("Cache-Control", "no-cache")
            self.send_header("Connection", "close")
            self.send_header("X-Accel-Buffering", "no")
            self.end_headers()
            self.close_connection = True

            total_input_tokens = 0
            total_output_tokens = 0

            # Initial role chunk
            initial_chunk = {
                "id": req_id,
                "object": "chat.completion.chunk",
                "created": created,
                "model": model,
                "choices": [{
                    "index": 0,
                    "delta": {"role": "assistant"},
                    "finish_reason": None,
                }],
            }
            self.wfile.write(f"data: {json.dumps(initial_chunk)}\n\n".encode("utf-8"))
            self.wfile.flush()

            while True:
                line = proc.stdout.readline()
                if not line:
                    if proc.poll() is not None:
                        break
                    time.sleep(0.01)
                    continue

                line = line.strip()
                if not line:
                    continue
                try:
                    event = json.loads(line)
                except Exception:
                    continue

                event_type = event.get("event")
                if event_type == "step_update":
                    step = event.get("step_update", {})
                    delta = step.get("text_delta")
                    if delta:
                        chunk = {
                            "id": req_id,
                            "object": "chat.completion.chunk",
                            "created": created,
                            "model": model,
                            "choices": [{
                                "index": 0,
                                "delta": {"content": delta},
                                "finish_reason": None,
                            }],
                        }
                        self.wfile.write(f"data: {json.dumps(chunk)}\n\n".encode("utf-8"))
                        self.wfile.flush()

                    usage = step.get("usage")
                    if usage:
                        total_input_tokens = usage.get("input_tokens", total_input_tokens)
                        total_output_tokens = usage.get("output_tokens", total_output_tokens)

                elif event_type == "result":
                    result = event.get("result", {})
                    usage = result.get("usage")
                    if usage:
                        total_input_tokens = usage.get("input_tokens", total_input_tokens)
                        total_output_tokens = usage.get("output_tokens", total_output_tokens)

            proc.wait()

            # Final stop chunk
            stop_chunk = {
                "id": req_id,
                "object": "chat.completion.chunk",
                "created": created,
                "model": model,
                "choices": [{
                    "index": 0,
                    "delta": {},
                    "finish_reason": "stop",
                }],
                "usage": {
                    "prompt_tokens": total_input_tokens,
                    "completion_tokens": total_output_tokens,
                    "total_tokens": total_input_tokens + total_output_tokens,
                },
            }
            self.wfile.write(f"data: {json.dumps(stop_chunk)}\n\n".encode("utf-8"))
            self.wfile.write(b"data: [DONE]\n\n")
            self.wfile.flush()

        else:
            # Non-streaming mode: accumulate
            full_response = []
            total_input_tokens = 0
            total_output_tokens = 0

            for line in proc.stdout:
                line = line.strip()
                if not line:
                    continue
                try:
                    event = json.loads(line)
                except Exception:
                    continue

                event_type = event.get("event")
                if event_type == "step_update":
                    step = event.get("step_update", {})
                    delta = step.get("text_delta")
                    if delta:
                        full_response.append(delta)
                    usage = step.get("usage")
                    if usage:
                        total_input_tokens = usage.get("input_tokens", total_input_tokens)
                        total_output_tokens = usage.get("output_tokens", total_output_tokens)
                elif event_type == "result":
                    result = event.get("result", {})
                    resp_text = result.get("response")
                    if resp_text and not full_response:
                        full_response.append(resp_text)
                    usage = result.get("usage")
                    if usage:
                        total_input_tokens = usage.get("input_tokens", total_input_tokens)
                        total_output_tokens = usage.get("output_tokens", total_output_tokens)

            proc.wait()
            content = "".join(full_response)

            resp_payload = {
                "id": req_id,
                "object": "chat.completion",
                "created": created,
                "model": model,
                "choices": [{
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": content,
                    },
                    "finish_reason": "stop",
                }],
                "usage": {
                    "prompt_tokens": total_input_tokens,
                    "completion_tokens": total_output_tokens,
                    "total_tokens": total_input_tokens + total_output_tokens,
                },
            }

            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(resp_payload).encode("utf-8"))


def main():
    server = HTTPServer(("0.0.0.0", PORT), AgyHandler)
    sys.stdout.write(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] agy-daemon started on port {PORT} for account '{ACCOUNT_LABEL}'\n")
    sys.stdout.flush()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        sys.stdout.write("\nShutting down agy-daemon.\n")
        server.server_close()


if __name__ == "__main__":
    main()
