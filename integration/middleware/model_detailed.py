#!/usr/bin/env python3
"""
NewAPI Model Metadata Enrichment Middleware

Reverse proxy that intercepts GET /v1/models and returns enriched metadata
for DeepSeek and MiniMax models, while transparently proxying all other
requests to the NewAPI backend.

Enriched Models:
  - deepseek-chat, deepseek-reasoner (DeepSeek)
  - MiniMax-M2.7, MiniMax-M2.5 (MiniMax Token Plan)

Usage:
    python3 model_detailed.py [--port PORT] [--backend BACKEND_URL]

Default:
    Listens on port 3001, proxies to http://localhost:3000 (NewAPI inside container)

Pricing Logic:
  All prices are per-token (not per-million). To convert from $/1M:
    per_token_price = price_per_1M / 1_000_000

  ModelRatio (new-api quota multiplier):
    ratio = input_price_per_1M / 2   (same formula as DeepSeek: $0.28 -> 0.14)

  CompletionRatio (new-api output multiplier):
    ratio = output_price_per_1M / input_price_per_1M

  MiniMax adds prompt_cache_miss since it has separate cache write pricing.
"""

import json
import os
import sys
import urllib.request
import urllib.error
from http.server import HTTPServer, BaseHTTPRequestHandler

# Configuration
PORT = int(os.environ.get("MIDDLEWARE_PORT", 3001))
BACKEND_URL = os.environ.get("NEWAPI_BACKEND_URL", "http://localhost:3000")

# DeepSeek model metadata
DEEPSEEK_METADATA = {
    "deepseek-chat": {
        "name": "DeepSeek V3.2",
        "context_length": 131072,
        "max_completion_tokens": 8192,
        "pricing": {
            "prompt": "0.00000028",       # $0.28 / 1M (cache miss)
            "completion": "0.00000042",   # $0.42 / 1M
            "prompt_cache_hit": "0.000000028",  # $0.028 / 1M
        },
    },
    "deepseek-reasoner": {
        "name": "DeepSeek V3.2 Reasoner",
        "context_length": 131072,
        "max_completion_tokens": 65536,
        "pricing": {
            "prompt": "0.00000028",
            "completion": "0.00000042",
            "prompt_cache_hit": "0.000000028",
        },
    },
}

# MiniMax model metadata
MINIMAX_METADATA = {
    "MiniMax-M2.7": {
        "name": "MiniMax M2.7",
        "context_length": 245760,
        "max_completion_tokens": 50000,
        "pricing": {
            "prompt": "0.0000003",        # $0.30 / 1M
            "completion": "0.0000012",    # $1.20 / 1M
            "prompt_cache_hit": "0.00000006",   # $0.06 / 1M
            "prompt_cache_miss": "0.000000375", # $0.375 / 1M
        },
    },
    "MiniMax-M2.5": {
        "name": "MiniMax M2.5",
        "context_length": 245760,
        "max_completion_tokens": 50000,
        "pricing": {
            "prompt": "0.0000003",        # $0.30 / 1M
            "completion": "0.0000012",    # $1.20 / 1M
            "prompt_cache_hit": "0.00000003",   # $0.03 / 1M
            "prompt_cache_miss": "0.000000375", # $0.375 / 1M
        },
    },
}

# Combined metadata lookup
ALL_MODEL_METADATA = {**DEEPSEEK_METADATA, **MINIMAX_METADATA}

# Standard created timestamp (more recent than NewAPI's hard-coded 1626777600)
MODEL_CREATED_TS = 1735689600  # 2025-01-01


def enrich_models_response(upstream_data):
    """Enrich the /v1/models response with metadata for DeepSeek and MiniMax models."""
    enriched = []
    for model in upstream_data.get("data", []):
        model_id = model.get("id", "")
        metadata = ALL_MODEL_METADATA.get(model_id)

        if metadata:
            # Determine owned_by from model id prefix
            if model_id.startswith("MiniMax"):
                owned_by = "minimax"
            else:
                owned_by = model.get("owned_by", "deepseek")

            enriched_model = {
                "id": model_id,
                "object": "model",
                "created": MODEL_CREATED_TS,
                "owned_by": owned_by,
                "name": metadata["name"],
                "context_length": metadata["context_length"],
                "top_provider": {
                    "max_completion_tokens": metadata["max_completion_tokens"],
                },
                "pricing": metadata["pricing"],
            }
            # Preserve any additional fields from upstream
            if "supported_endpoint_types" in model:
                enriched_model["supported_endpoint_types"] = model["supported_endpoint_types"]
            enriched.append(enriched_model)
        else:
            # Non-enriched models: pass through unchanged
            enriched.append(model)

    return {
        "data": enriched,
        "object": "list",
        "success": True,
    }


class EnrichmentProxyHandler(BaseHTTPRequestHandler):
    """HTTP handler that enriches /v1/models and proxies everything else."""

    def log_message(self, format, *args):
        """Log to stdout for Docker visibility."""
        sys.stdout.write(f"[middleware] {args[0]}\n")
        sys.stdout.flush()

    def _proxy_request(self, method):
        """Forward request to NewAPI backend and return response."""
        # Build backend URL
        path = self.path
        backend_url = f"{BACKEND_URL}{path}"

        # Read request body if present
        body = None
        content_length = int(self.headers.get("Content-Length", 0))
        if content_length > 0:
            body = self.rfile.read(content_length)

        # Build headers to forward
        headers = {}
        for key in self.headers:
            if key.lower() not in ("host", "connection"):
                headers[key] = self.headers[key]

        # Create backend request
        req = urllib.request.Request(backend_url, data=body, method=method, headers=headers)

        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                status = resp.status
                resp_body = resp.read()
                resp_headers = dict(resp.headers)
                return status, resp_headers, resp_body
        except urllib.error.HTTPError as e:
            return e.code, dict(e.headers), e.read()
        except Exception as e:
            self.send_error(502, f"Backend error: {e}")
            return None

    def do_GET(self):
        if self.path == "/v1/models":
            self._handle_models_list()
        else:
            result = self._proxy_request("GET")
            if result:
                status, headers, body = result
                self.send_response(status)
                for key, value in headers.items():
                    if key.lower() not in ("transfer-encoding", "connection"):
                        self.send_header(key, value)
                self.end_headers()
                self.wfile.write(body)

    def _handle_models_list(self):
        """Intercept /v1/models, enrich, and return."""
        result = self._proxy_request("GET")
        if not result:
            return

        status, headers, body = result
        if status != 200:
            # If backend failed, pass through error
            self.send_response(status)
            for key, value in headers.items():
                if key.lower() not in ("transfer-encoding", "connection"):
                    self.send_header(key, value)
            self.end_headers()
            self.wfile.write(body)
            return

        try:
            data = json.loads(body)
            enriched = enrich_models_response(data)
            response_body = json.dumps(enriched, indent=2).encode("utf-8")
        except (json.JSONDecodeError, KeyError) as e:
            self.log_message(f"Failed to parse models response: {e}")
            # Fall back to original response
            response_body = body

        self.send_response(200)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(response_body)))
        # Add header to indicate enrichment
        self.send_header("X-Model-Metadata-Enriched", "true")
        self.end_headers()
        self.wfile.write(response_body)

    def do_POST(self):
        result = self._proxy_request("POST")
        if result:
            status, headers, body = result
            self.send_response(status)
            for key, value in headers.items():
                if key.lower() not in ("transfer-encoding", "connection"):
                    self.send_header(key, value)
            self.end_headers()
            self.wfile.write(body)

    def do_PUT(self):
        result = self._proxy_request("PUT")
        if result:
            status, headers, body = result
            self.send_response(status)
            for key, value in headers.items():
                if key.lower() not in ("transfer-encoding", "connection"):
                    self.send_header(key, value)
            self.end_headers()
            self.wfile.write(body)

    def do_DELETE(self):
        result = self._proxy_request("DELETE")
        if result:
            status, headers, body = result
            self.send_response(status)
            for key, value in headers.items():
                if key.lower() not in ("transfer-encoding", "connection"):
                    self.send_header(key, value)
            self.end_headers()
            self.wfile.write(body)


def main():
    print(f"[middleware] Starting NewAPI Model Enrichment Middleware")
    print(f"[middleware] Listening on port {PORT}")
    print(f"[middleware] Backend: {BACKEND_URL}")
    all_models = list(DEEPSEEK_METADATA.keys()) + list(MINIMAX_METADATA.keys())
    print(f"[middleware] Enriching models: {', '.join(all_models)}")

    server = HTTPServer(("0.0.0.0", PORT), EnrichmentProxyHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[middleware] Shutting down...")
        server.shutdown()


if __name__ == "__main__":
    main()
