#!/usr/bin/env python3
"""
NewAPI Model Metadata Enrichment Middleware - FastAPI Version

High-performance async proxy that intercepts GET /v1/models and returns enriched
metadata for DeepSeek and MiniMax models, while transparently proxying all other
requests to the NewAPI backend.

Advantages over BaseHTTPRequestHandler:
- Async/await for concurrent request handling
- Uvicorn running with multiple workers
- Proper connection pooling
- Better error handling
- Production-ready

Usage:
    uvicorn model_detailed_fastapi:app --host 0.0.0.0 --port 3001 --workers 4
"""

import json
import os
from contextlib import asynccontextmanager

import httpx
from fastapi import FastAPI, Request, HTTPException
from fastapi.responses import JSONResponse, Response
from fastapi.middleware.cors import CORSMiddleware

# Configuration
PORT = int(os.environ.get("MIDDLEWARE_PORT", 3001))
BACKEND_URL = os.environ.get("NEWAPI_BACKEND_URL", "http://localhost:3000")
TIMEOUT = int(os.environ.get("BACKEND_TIMEOUT", 60))

# DeepSeek V4 model metadata
DEEPSEEK_METADATA = {
    "deepseek-v4-flash": {
        "name": "DeepSeek V4 Flash",
        "context_length": 131072,
        "max_completion_tokens": 8192,
        "pricing": {
            "prompt": "0.00000014",
            "completion": "0.00000028",
            "prompt_cache_hit": "0.000000014",
        },
    },
    "deepseek-v4-pro": {
        "name": "DeepSeek V4 Pro",
        "context_length": 131072,
        "max_completion_tokens": 8192,
        "pricing": {
            "prompt": "0.000000435",
            "completion": "0.00000087",
            "prompt_cache_hit": "0.0000000435",
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
            "prompt": "0.0000003",
            "completion": "0.0000012",
            "prompt_cache_hit": "0.00000006",
            "prompt_cache_miss": "0.000000375",
        },
    },
    "MiniMax-M2.7-highspeed": {
        "name": "MiniMax M2.7 Highspeed",
        "context_length": 245760,
        "max_completion_tokens": 50000,
        "pricing": {
            "prompt": "0.0000003",
            "completion": "0.0000012",
            "prompt_cache_hit": "0.00000006",
            "prompt_cache_miss": "0.000000375",
        },
    },
    "MiniMax-M2.5": {
        "name": "MiniMax M2.5",
        "context_length": 245760,
        "max_completion_tokens": 50000,
        "pricing": {
            "prompt": "0.0000003",
            "completion": "0.0000012",
            "prompt_cache_hit": "0.00000003",
            "prompt_cache_miss": "0.000000375",
        },
    },
    "MiniMax-M2.5-highspeed": {
        "name": "MiniMax M2.5 Highspeed",
        "context_length": 245760,
        "max_completion_tokens": 50000,
        "pricing": {
            "prompt": "0.0000003",
            "completion": "0.0000012",
            "prompt_cache_hit": "0.00000003",
            "prompt_cache_miss": "0.000000375",
        },
    },
}

# Combined metadata lookup
ALL_MODEL_METADATA = {**DEEPSEEK_METADATA, **MINIMAX_METADATA}

# Standard created timestamp (more recent than NewAPI's hard-coded 1626777600)
MODEL_CREATED_TS = 1735689600  # 2025-01-01

# HTTP Client with connection pooling
http_client: httpx.AsyncClient | None = None


async def get_http_client() -> httpx.AsyncClient:
    global http_client
    if http_client is None or http_client.is_closed:
        http_client = httpx.AsyncClient(
            timeout=httpx.Timeout(TIMEOUT),
            limits=httpx.Limits(max_keepalive_connections=20, max_connections=100),
            follow_redirects=True,
        )
    return http_client


async def enrich_models_response(upstream_data: dict) -> dict:
    """Enrich the /v1/models response with metadata for DeepSeek and MiniMax models."""
    enriched = []
    for model in upstream_data.get("data", []):
        model_id = model.get("id", "")
        metadata = ALL_MODEL_METADATA.get(model_id)

        if metadata:
            # Determine owned_by from model id prefix
            if model_id.startswith("MiniMax"):
                owned_by = "minimax"
            elif model_id.startswith("deepseek"):
                owned_by = "deepseek"
            else:
                owned_by = model.get("owned_by", "unknown")

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


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    print(f"[middleware] Starting FastAPI Model Enrichment Middleware")
    print(f"[middleware] Backend: {BACKEND_URL}")
    print(f"[middleware] Timeout: {TIMEOUT}s")
    all_models = list(DEEPSEEK_METADATA.keys()) + list(MINIMAX_METADATA.keys())
    print(f"[middleware] Enriching models: {', '.join(all_models)}")
    yield
    # Shutdown
    if http_client and not http_client.is_closed:
        await http_client.aclose()


app = FastAPI(
    title="Model Metadata Enrichment Middleware",
    description="FastAPI middleware for enriching model metadata",
    version="2.0.0",
    lifespan=lifespan,
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# IMPORTANT: Define specific routes BEFORE catch-all proxy route


@app.get("/health")
async def health_check():
    """Health check endpoint - must be before catch-all."""
    client = await get_http_client()
    try:
        # Test backend connectivity
        resp = await client.get(f"{BACKEND_URL}/v1/models", timeout=5)
        if resp.status_code == 200:
            return {"status": "healthy", "backend": "connected"}
        return {"status": "degraded", "backend": f"returned {resp.status_code}"}
    except Exception as e:
        return {"status": "unhealthy", "backend": "disconnected", "error": str(e)}


@app.get("/v1/models")
async def get_models(request: Request):
    """Intercept /v1/models, enrich, and return."""
    client = await get_http_client()
    
    # Forward authorization header
    headers = {}
    auth = request.headers.get("authorization")
    if auth:
        headers["authorization"] = auth
    
    try:
        response = await client.get(f"{BACKEND_URL}/v1/models", headers=headers)
        if response.status_code != 200:
            return Response(
                content=response.content,
                status_code=response.status_code,
                media_type="application/json",
            )

        try:
            data = response.json()
            enriched = await enrich_models_response(data)
            return JSONResponse(content=enriched, headers={"X-Model-Metadata-Enriched": "true"})
        except json.JSONDecodeError as e:
            print(f"[middleware] Failed to parse models response: {e}")
            return Response(content=response.content, status_code=200, media_type="application/json")
    except httpx.TimeoutException:
        raise HTTPException(status_code=504, detail="Backend timeout")
    except httpx.RequestError as e:
        raise HTTPException(status_code=502, detail=f"Backend error: {e}")


@app.get("/models")
async def get_models_legacy(request: Request):
    """Legacy /models endpoint (without /v1 prefix)."""
    return await get_models(request)


@app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"])
async def proxy_request(path: str, request: Request):
    """Proxy all other requests to the backend."""
    client = await get_http_client()

    # Get request body
    body = await request.body()

    # Build headers to forward - filter problematic headers
    headers = {}
    skip_headers = {"host", "connection", "content-length", "transfer-encoding", "content-encoding"}
    for key, value in request.headers.items():
        if key.lower() not in skip_headers:
            headers[key] = value

    try:
        backend_url = f"{BACKEND_URL}/{path}"
        response = await client.request(
            method=request.method,
            url=backend_url,
            headers=headers,
            content=body if body else None,
        )

        # Build response headers - exclude problematic ones
        response_headers = {}
        skip_response_headers = {"transfer-encoding", "content-encoding", "content-length"}
        for key, value in response.headers.items():
            if key.lower() not in skip_response_headers:
                response_headers[key] = value

        return Response(
            content=response.content,
            status_code=response.status_code,
            headers=response_headers,
            media_type=response.headers.get("content-type", "application/json"),
        )
    except httpx.TimeoutException:
        raise HTTPException(status_code=504, detail="Backend timeout")
    except httpx.RequestError as e:
        raise HTTPException(status_code=502, detail=f"Backend error: {e}")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "model_detailed_fastapi:app",
        host="0.0.0.0",
        port=PORT,
        workers=4,
        log_level="info",
    )
