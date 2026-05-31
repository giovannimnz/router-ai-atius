#!/bin/bash
# Use envsubst to substitute port in uvicorn command
export PORT="${MIDDLEWARE_PORT:-3001}"
exec uvicorn model_detailed_fastapi:app \
  --host 0.0.0.0 \
  --port "$PORT" \
  --workers 4 \
  --log-level info
