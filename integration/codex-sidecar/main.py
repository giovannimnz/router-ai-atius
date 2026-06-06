#!/usr/bin/env python3
"""
Codex SDK Sidecar — HTTP bridge for openai-codex SDK.

FastAPI app that wraps the openai-codex SDK and exposes:
  GET  /health          — health check + available models
  POST /v1/codex/run    — one-shot prompt via thread.run()
  POST /v1/codex/thread — stateful thread management

Phase 05 — SDk-01: Sidecar Python + HTTP Bridge
"""

import json
import os
import time
import logging
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("codex-sidecar")

# ---------------------------------------------------------------------------
# Models (mirrors relay/channel/codex/constants.go)
# ---------------------------------------------------------------------------
SUPPORTED_MODELS = [
    "gpt-5", "gpt-5-codex", "gpt-5-codex-mini",
    "gpt-5.1", "gpt-5.1-codex", "gpt-5.1-codex-max", "gpt-5.1-codex-mini",
    "gpt-5.2", "gpt-5.2-codex",
    "gpt-5.3-codex", "gpt-5.3-codex-spark",
    "gpt-5.4",
]

# ---------------------------------------------------------------------------
# In-memory thread store (D-04)
# ---------------------------------------------------------------------------
_threads: dict[str, object] = {}
_thread_counter: int = 0

# ---------------------------------------------------------------------------
# SDK handle (initialized at startup)
# ---------------------------------------------------------------------------
_codex_sdk = None


def _get_sdk():
    """Return the Codex SDK handle, or raise if not initialized."""
    if _codex_sdk is None:
        raise HTTPException(status_code=503, detail="Codex SDK not initialized")
    return _codex_sdk


def _load_license() -> Optional[dict]:
    """Load license from data/codex/license.json (SDK-02) or fallback to ~/.codex/auth.json."""
    paths = [
        os.path.join(os.path.dirname(__file__), "..", "..", "data", "codex", "license.json"),
        os.path.expanduser("~/.codex/auth.json"),
    ]
    for p in paths:
        try:
            if os.path.exists(p):
                with open(p) as f:
                    return json.load(f)
        except (OSError, json.JSONDecodeError):
            continue
    return None


# ---------------------------------------------------------------------------
# Request/Response schemas
# ---------------------------------------------------------------------------
class RunRequest(BaseModel):
    model: str
    prompt: str
    stream: bool = False


class ThreadRequest(BaseModel):
    model: str
    prompt: str
    thread_id: Optional[str] = None
    stream: bool = False


class CodexThread:
    """Wrapper for a Codex thread handle."""

    def __init__(self, handle):
        self.handle = handle


# ---------------------------------------------------------------------------
# App lifecycle
# ---------------------------------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI):
    global _codex_sdk
    try:
        from openai_codex import Codex
    except ImportError:
        logger.error("openai-codex not installed. Run: pip install openai-codex")
        raise

    license_data = _load_license()
    logger.info("Starting Codex SDK (license=%s)", "present" if license_data else "absent")

    _codex_sdk = Codex()
    _codex_sdk.__enter__()
    logger.info("Codex SDK ready")

    yield

    if _codex_sdk is not None:
        try:
            _codex_sdk.__exit__(None, None, None)
        except Exception:
            pass
        _codex_sdk = None
        logger.info("Codex SDK shutdown")


app = FastAPI(title="Codex SDK Sidecar", version="0.1.0", lifespan=lifespan)


# ---------------------------------------------------------------------------
# Health
# ---------------------------------------------------------------------------
@app.get("/health")
async def health():
    return {
        "status": "ok",
        "models": SUPPORTED_MODELS,
        "threads_active": len(_threads),
    }


# ---------------------------------------------------------------------------
# POST /v1/codex/run — one-shot
# ---------------------------------------------------------------------------
@app.post("/v1/codex/run")
async def codex_run(req: RunRequest):
    if req.model not in SUPPORTED_MODELS:
        raise HTTPException(status_code=400, detail=f"Unsupported model: {req.model}")

    sdk = _get_sdk()
    thread = sdk.thread_start(model=req.model)
    result = thread.run(req.prompt)

    if req.stream:
        return StreamingResponse(
            _stream_response(result),
            media_type="text/event-stream",
        )

    return {
        "final_response": result.final_response,
        "usage": _usage_to_dict(result),
        "thread_id": str(id(thread.handle)) if hasattr(thread, "handle") else None,
    }


# ---------------------------------------------------------------------------
# POST /v1/codex/thread — stateful
# ---------------------------------------------------------------------------
@app.post("/v1/codex/thread")
async def codex_thread(req: ThreadRequest):
    if req.model not in SUPPORTED_MODELS:
        raise HTTPException(status_code=400, detail=f"Unsupported model: {req.model}")

    sdk = _get_sdk()
    global _thread_counter

    if req.thread_id and req.thread_id in _threads:
        thread = _threads[req.thread_id]
    else:
        thread = sdk.thread_start(model=req.model)
        _thread_counter += 1
        tid = req.thread_id or f"thread-{_thread_counter}"
        _threads[tid] = thread

    result = thread.run(req.prompt)

    if req.stream:
        return StreamingResponse(
            _stream_response(result),
            media_type="text/event-stream",
        )

    return {
        "final_response": result.final_response,
        "usage": _usage_to_dict(result),
        "thread_id": req.thread_id or f"thread-{_thread_counter}",
    }


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
async def _stream_response(result):
    """SSE streaming adapter."""
    yield f"data: {json.dumps({'final_response': result.final_response})}\n\n"
    if hasattr(result, "usage"):
        yield f"data: {json.dumps({'usage': _usage_to_dict(result)})}\n\n"
    yield "data: [DONE]\n\n"


def _usage_to_dict(result) -> dict:
    """Extract usage info from TurnResult."""
    if hasattr(result, "usage") and result.usage is not None:
        u = result.usage
        return {
            "input_tokens": getattr(u, "input_tokens", 0),
            "output_tokens": getattr(u, "output_tokens", 0),
            "total_tokens": getattr(u, "total_tokens", 0),
        }
    return {"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
