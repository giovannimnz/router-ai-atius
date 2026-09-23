#!/usr/bin/env python3
"""
agy-daemon.py — OpenAI-compatible HTTP wrapper around the agy CLI.

Supports single-instance and multi-instance round-robin load balancing.

Environment variables:
  AGY_DAEMON_PORT    Listen port (default: 8080)
  AGY_ACCOUNT_LABEL  Label for logs (default: default)
  AGY_POOL_URLS      Comma-separated upstream agy-daemon URLs for LB mode.
                     When set, this instance acts as a load-balancer proxy
                     and does NOT spawn agy directly.
                     Example: http://127.0.0.1:18081,http://127.0.0.1:18082
"""

import json
import os
import signal
import subprocess
import sys
import threading
import time
import uuid
import urllib.request
import urllib.error
from http.server import BaseHTTPRequestHandler, HTTPServer
from socketserver import ThreadingMixIn


class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    daemon_threads = True
    allow_reuse_address = True


PORT = int(os.environ.get("AGY_DAEMON_PORT", "8080"))
ACCOUNT_LABEL = os.environ.get("AGY_ACCOUNT_LABEL", "default")
DEFAULT_MODEL = "gemini-3.8-flash"

# Limite seguro de concorrência do pool (padrão 6 caminhos simultâneos)
MAX_CONCURRENT_REQUESTS = int(os.environ.get("AGY_MAX_CONCURRENCY", "6"))
_concurrency_semaphore = threading.BoundedSemaphore(MAX_CONCURRENT_REQUESTS)
_active_requests_lock = threading.Lock()
_active_requests = 0

SUPPORTED_MODELS = [
    "gemini-3.8-flash",
    "gemini-3.7-flash",
    "gemini-3.6-flash",
    "gemini-3.1-pro",
    "claude-sonnet-4-6",
    "claude-opus-4-6-thinking",
    "gpt-oss-120b",
    # Backward-compatible explicit effort models:
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
    "gpt-oss-120b-medium",
]

# --- Load Balancer State ---
_pool_urls = [u.strip() for u in os.environ.get("AGY_POOL_URLS", "").split(",") if u.strip()]
_pool_lock = threading.Lock()
_pool_index = 0
_pool_failures: dict[str, int] = {}  # url -> consecutive failures
MAX_FAILURES = 3  # mark unhealthy after this many consecutive failures


def _next_pool_url() -> str | None:
    """Round-robin over healthy pool URLs."""
    global _pool_index
    if not _pool_urls:
        return None
    with _pool_lock:
        start = _pool_index
        for i in range(len(_pool_urls)):
            idx = (start + i) % len(_pool_urls)
            url = _pool_urls[idx]
            if _pool_failures.get(url, 0) < MAX_FAILURES:
                _pool_index = (idx + 1) % len(_pool_urls)
                return url
        # All unhealthy — reset and try first
        for url in _pool_urls:
            _pool_failures[url] = 0
        _pool_index = 1 % len(_pool_urls)
        return _pool_urls[0]


def _mark_pool_failure(url: str) -> None:
    with _pool_lock:
        _pool_failures[url] = _pool_failures.get(url, 0) + 1
        if _pool_failures[url] >= MAX_FAILURES:
            sys.stdout.write(
                f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] WARN: pool url {url} marked unhealthy "
                f"after {MAX_FAILURES} failures\n"
            )
            sys.stdout.flush()


def _mark_pool_success(url: str) -> None:
    with _pool_lock:
        _pool_failures[url] = 0


def _proxy_to_pool(handler: "AgyHandler", path: str, body: bytes | None, method: str) -> bool:
    """Proxy request to next pool URL with failover. Returns True if handled."""
    tried: set[str] = set()
    while True:
        url = _next_pool_url()
        if url is None or url in tried:
            return False
        tried.add(url)
        target = url.rstrip("/") + path
        try:
            req = urllib.request.Request(
                target,
                data=body,
                method=method,
                headers={"Content-Type": "application/json"} if body else {},
            )
            resp = urllib.request.urlopen(req, timeout=300)
            _mark_pool_success(url)
            handler.send_response(resp.status)
            for key, val in resp.headers.items():
                if key.lower() not in ("transfer-encoding", "connection"):
                    handler.send_header(key, val)
            handler.send_header("Connection", "close")
            handler.end_headers()
            handler.close_connection = True
            # Stream response bytes
            while True:
                chunk = resp.read(4096)
                if not chunk:
                    break
                handler.wfile.write(chunk)
                handler.wfile.flush()
            return True
        except urllib.error.HTTPError as e:
            _mark_pool_success(url)  # server responded, not a connection failure
            handler.send_response(e.code)
            handler.send_header("Content-Type", "application/json")
            handler.end_headers()
            handler.wfile.write(e.read())
            return True
        except Exception as e:
            sys.stdout.write(
                f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] pool {url} error: {e}\n"
            )
            sys.stdout.flush()
            _mark_pool_failure(url)
            continue  # try next

    sys.stdout.write(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] All pool backends failed for {path}\n")
    sys.stdout.flush()
    handler.send_error(502, "All pool backends failed")
    return False


# --- Message Formatting ---

def format_messages_to_prompt(messages: list) -> str:
    parts = []
    for msg in messages:
        role = msg.get("role", "user")
        content = msg.get("content", "")
        if isinstance(content, list):
            content = " ".join(
                c.get("text", "") for c in content if isinstance(c, dict) and c.get("type") == "text"
            )
        if role == "system":
            parts.append(f"[System]: {content}")
        elif role == "assistant":
            parts.append(f"[Assistant]: {content}")
        else:
            parts.append(content)
    return "\n".join(parts)


# --- HTTP Handler ---

class AgyHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        sys.stdout.write(
            f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] [{ACCOUNT_LABEL}] {fmt % args}\n"
        )
        sys.stdout.flush()

    def do_GET(self):
        if self.path == "/health":
            # In LB mode check all pool members
            if _pool_urls:
                statuses = []
                for url in _pool_urls:
                    healthy = _pool_failures.get(url, 0) < MAX_FAILURES
                    statuses.append({"url": url, "healthy": healthy})
                body = json.dumps({
                    "status": "ok",
                    "mode": "load-balancer",
                    "capacity_limit": MAX_CONCURRENT_REQUESTS,
                    "active_requests": _active_requests,
                    "pool": statuses,
                }).encode()
            else:
                body = json.dumps({
                    "status": "ok",
                    "account": ACCOUNT_LABEL,
                    "capacity_limit": MAX_CONCURRENT_REQUESTS,
                    "active_requests": _active_requests,
                    "mode": "direct",
                }).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)

        elif self.path == "/v1/models":
            if _pool_urls and _proxy_to_pool(self, self.path, None, "GET"):
                return
            body = json.dumps({
                "object": "list",
                "data": [
                    {"id": m, "object": "model", "owned_by": "google-antigravity"}
                    for m in SUPPORTED_MODELS
                ],
            }).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)
        else:
            self.send_error(404, "Not found")

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404, "Not found")
            return

        content_length = int(self.headers.get("Content-Length", 0))
        if content_length == 0:
            self.send_error(400, "Empty request body")
            return

        body = self.rfile.read(content_length)

        # Load-balancer mode: forward to pool
        if _pool_urls:
            _proxy_to_pool(self, self.path, body, "POST")
            return

        # Direct agy mode: controle de capacidade com semáforo (limite seguro de 6 caminhos)
        global _active_requests
        acquired = _concurrency_semaphore.acquire(timeout=45)
        if not acquired:
            self.send_error(429, f"Agy instance busy: maximum concurrency reached ({MAX_CONCURRENT_REQUESTS})")
            return
        with _active_requests_lock:
            _active_requests += 1

        try:
            self._handle_direct_post(body)
        finally:
            with _active_requests_lock:
                _active_requests = max(0, _active_requests - 1)
            _concurrency_semaphore.release()

    def _handle_direct_post(self, body: bytes):
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

        effort = str(req_data.get("reasoning_effort") or "high").lower().strip()
        if model == "gemini-3.8-flash":
            target_model = f"gemini-3.8-flash-{effort}"
        elif model == "gemini-3.7-flash":
            target_model = f"gemini-3.7-flash-{effort}"
        elif model == "gemini-3.6-flash":
            target_model = f"gemini-3.6-flash-{effort}"
        elif model == "gemini-3.1-pro":
            target_model = "gemini-3.1-pro-high" if effort in ("high", "medium") else "gemini-3.1-pro-low"
        elif model == "gpt-oss-120b":
            target_model = "gpt-oss-120b-medium"
        else:
            target_model = model

        cmd = [
            "agy",
            "-p", prompt,
            "--output-format", "stream-json",
            "--dangerously-skip-permissions",
        ]
        if target_model in SUPPORTED_MODELS:
            cmd.extend(["--model", target_model])

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

            initial_chunk = {
                "id": req_id, "object": "chat.completion.chunk",
                "created": created, "model": model,
                "choices": [{"index": 0, "delta": {"role": "assistant"}, "finish_reason": None}],
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
                            "id": req_id, "object": "chat.completion.chunk",
                            "created": created, "model": model,
                            "choices": [{"index": 0, "delta": {"content": delta}, "finish_reason": None}],
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

            stop_chunk = {
                "id": req_id, "object": "chat.completion.chunk",
                "created": created, "model": model,
                "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}],
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
                "id": req_id, "object": "chat.completion",
                "created": created, "model": model,
                "choices": [{"index": 0, "message": {"role": "assistant", "content": content}, "finish_reason": "stop"}],
                "usage": {
                    "prompt_tokens": total_input_tokens,
                    "completion_tokens": total_output_tokens,
                    "total_tokens": total_input_tokens + total_output_tokens,
                },
            }

            resp_bytes = json.dumps(resp_payload).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(resp_bytes)))
            self.send_header("Connection", "close")
            self.end_headers()
            self.wfile.write(resp_bytes)
            self.wfile.flush()
            self.close_connection = True


def main():
    mode = "load-balancer" if _pool_urls else "direct"
    server = ThreadedHTTPServer(("0.0.0.0", PORT), AgyHandler)

    def _sig_handler(signum, frame):
        sys.stdout.write(f"\nReceived signal {signum}, shutting down agy-daemon...\n")
        sys.stdout.flush()
        # In a separate thread so serve_forever unblocks cleanly
        threading.Thread(target=server.shutdown).start()

    signal.signal(signal.SIGTERM, _sig_handler)
    signal.signal(signal.SIGINT, _sig_handler)

    sys.stdout.write(
        f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] agy-daemon started on port {PORT} "
        f"account='{ACCOUNT_LABEL}' mode={mode}"
        + (f" pool={_pool_urls}" if _pool_urls else "") + "\n"
    )
    sys.stdout.flush()
    try:
        server.serve_forever()
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
