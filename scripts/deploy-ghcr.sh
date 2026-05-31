#!/bin/bash
# Deploy Atius Router from GHCR after workflow build
set -e

REPO="ghcr.io/giovannimnz/router-ai-atius"
TAG="${1:-main}"

echo "=== Pulling new image ${REPO}:${TAG} ==="
docker pull "${REPO}:${TAG}" || docker pull "${REPO}:${TAG}-amd64"

echo "=== Stopping old container ==="
docker stop new-api || true
docker rm new-api || true

echo "=== Running new container ==="
docker run -d \
  --name new-api \
  --network atius-shared \
  --restart always \
  -p 3301:3000 \
  -v /home/ubuntu/docker/Atius/router-ai-atius/data:/data \
  -e PORT=3000 \
  "${REPO}:${TAG}"

echo "=== Waiting for health check ==="
sleep 10

echo "=== Verifying ==="
docker logs new-api --tail 5
echo ""
echo "=== Version check ==="
curl -s http://127.0.0.1:3301/ -H "Host: router.atius.com.br" | head -c 200
echo ""
echo "=== X-New-Api-Version header ==="
curl -si http://127.0.0.1:3301/ -H "Host: router.atius.com.br" | grep -i "X-New-Api-Version"
