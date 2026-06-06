FROM python:3.12-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY main.py .

# Expose sidecar port
EXPOSE 1456

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:1456/health')" || exit 1

# Run with uvicorn (single worker — SDK manages its own subprocess)
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "1456", "--log-level", "info"]
