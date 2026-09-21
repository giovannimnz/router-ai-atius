#!/usr/bin/env bash
# ==============================================================================
# podman-agy-pool.sh - Multi-Account Antigravity Podman Pool Manager
# Manages isolated rootless Podman containers for Antigravity (agy) runtimes
# with dedicated public egress IPs per container (1 Container = 1 Account = 1 IP).
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROUTER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Egress IP definitions for Phase 1
ACC1_CONTAINER="atius-agy-acc1"
ACC1_LOCAL_IP="10.0.0.38"
ACC1_PUBLIC_IP="137.131.190.161"
ACC1_PORT="18081"
ACC1_VOL="atius-agy-vol-acc1"

ACC2_CONTAINER="atius-agy-acc2"
ACC2_LOCAL_IP="10.0.0.238"
ACC2_PUBLIC_IP="137.131.140.20"
ACC2_PORT="18082"
ACC2_VOL="atius-agy-vol-acc2"

BASE_IMAGE="docker.io/library/golang:1.25-bookworm"
AGY_BIN="$(command -v agy || echo "/home/ubuntu/.local/bin/agy")"
DAEMON_SCRIPT="$SCRIPT_DIR/agy-daemon.py"

# Guardrail: 0.5 CPU per container, 1024m RAM
CPU_LIMIT="0.5"
MEM_LIMIT="1024m"

setup_volumes() {
  echo "==> Configuring volumes for agy pool..."
  for vol in "$ACC1_VOL" "$ACC2_VOL"; do
    if ! podman volume exists "$vol" 2>/dev/null; then
      echo "Creating volume $vol..."
      podman volume create "$vol" >/dev/null
    fi
  done

  # Initialize volume contents from local baseline if empty
  for vol in "$ACC1_VOL" "$ACC2_VOL"; do
    vol_path="$(podman volume inspect "$vol" --format '{{.Mountpoint}}')"
    mkdir -p "$vol_path/antigravity-cli"
    if [[ ! -f "$vol_path/antigravity-cli/antigravity-oauth-token" && -f "$HOME/.gemini/antigravity-cli/antigravity-oauth-token" ]]; then
      echo "Seeding initial token into $vol..."
      cp "$HOME/.gemini/antigravity-cli/antigravity-oauth-token" "$vol_path/antigravity-cli/"
      chmod 600 "$vol_path/antigravity-cli/antigravity-oauth-token"
    fi
    if [[ ! -f "$vol_path/antigravity-cli/settings.json" && -f "$HOME/.gemini/antigravity-cli/settings.json" ]]; then
      cp "$HOME/.gemini/antigravity-cli/settings.json" "$vol_path/antigravity-cli/"
    fi
  done
  echo "Volumes initialized successfully."
}

start_container() {
  local name="$1"
  local local_ip="$2"
  local port="$3"
  local vol="$4"
  local label="$5"

  echo "==> Starting $name on port $port with egress bound to $local_ip..."
  if podman container exists "$name" 2>/dev/null; then
    echo "Stopping existing $name..."
    podman stop -t 2 "$name" >/dev/null 2>&1 || true
    podman rm -f "$name" >/dev/null 2>&1 || true
  fi

  taskset -c 0,1 podman run -d \
    --name "$name" \
    --restart unless-stopped \
    --network="slirp4netns:outbound_addr=$local_ip" \
    -p "127.0.0.1:$port:8080" \
    -v "$vol:/root/.gemini:rw" \
    -v "$AGY_BIN:/usr/local/bin/agy:ro" \
    -v "$DAEMON_SCRIPT:/usr/local/bin/agy-daemon.py:ro" \
    -e "AGY_DAEMON_PORT=8080" \
    -e "AGY_ACCOUNT_LABEL=$label" \
    --cpus "$CPU_LIMIT" \
    --memory "$MEM_LIMIT" \
    "$BASE_IMAGE" \
    python3 /usr/local/bin/agy-daemon.py

  echo "Container $name started."
}

start_pool() {
  setup_volumes
  start_container "$ACC1_CONTAINER" "$ACC1_LOCAL_IP" "$ACC1_PORT" "$ACC1_VOL" "acc1-vpn-atius"
  start_container "$ACC2_CONTAINER" "$ACC2_LOCAL_IP" "$ACC2_PORT" "$ACC2_VOL" "acc2-home-proxy"
  echo ""
  echo "Waiting 3 seconds for daemons to initialize..."
  sleep 3
  status_pool
}

stop_pool() {
  echo "==> Stopping agy pool containers..."
  for name in "$ACC1_CONTAINER" "$ACC2_CONTAINER"; do
    if podman container exists "$name" 2>/dev/null; then
      echo "Stopping $name..."
      podman stop -t 5 "$name" || true
      podman rm -f "$name" || true
    fi
  done
  echo "Pool stopped."
}

status_pool() {
  echo "================================================================="
  echo "  Atius Antigravity Pool Status (Multi-Account Egress)"
  echo "================================================================="
  printf "%-18s | %-12s | %-16s | %-16s | %-10s\n" "Container" "Local Port" "Bind IP (Host)" "Public Egress IP" "Health"
  echo "-------------------|--------------|------------------|------------------|-----------"

  for item in "$ACC1_CONTAINER:$ACC1_PORT:$ACC1_LOCAL_IP:$ACC1_PUBLIC_IP" "$ACC2_CONTAINER:$ACC2_PORT:$ACC2_LOCAL_IP:$ACC2_PUBLIC_IP"; do
    IFS=":" read -r name port local_ip expected_public <<<"$item"
    local health="DOWN"
    local actual_egress="UNKNOWN"

    if podman container exists "$name" 2>/dev/null; then
      if curl -s -m 2 "http://127.0.0.1:$port/health" | grep -q "ready"; then
        health="READY"
      else
        health="STARTING"
      fi
      actual_egress="$(podman exec "$name" curl -s -m 4 https://api.ipify.org 2>/dev/null || echo "UNPROVEN")"
    fi

    printf "%-18s | %-12s | %-16s | %-16s | %-10s\n" "$name" "$port" "$local_ip" "$actual_egress" "$health"
  done
  echo "================================================================="
}

test_prompt() {
  local port="${1:-$ACC1_PORT}"
  echo "==> Sending test prompt to daemon at port $port (stream=true)..."
  curl -N -s -X POST "http://127.0.0.1:$port/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d '{
      "model": "gemini-3.8-flash-high",
      "stream": true,
      "messages": [
        {"role": "user", "content": "Diga em uma única palavra se a conexão está OK."}
      ]
    }'
  echo ""
}

case "${1:-status}" in
  start)
    start_pool
    ;;
  stop)
    stop_pool
    ;;
  restart)
    stop_pool
    start_pool
    ;;
  status)
    status_pool
    ;;
  setup)
    setup_volumes
    ;;
  test)
    test_prompt "${2:-$ACC1_PORT}"
    ;;
  logs)
    echo "--- Logs for $ACC1_CONTAINER ---"
    podman logs --tail 20 "$ACC1_CONTAINER" 2>&1 || true
    echo "--- Logs for $ACC2_CONTAINER ---"
    podman logs --tail 20 "$ACC2_CONTAINER" 2>&1 || true
    ;;
  *)
    echo "Usage: $0 {start|stop|restart|status|setup|test [port]|logs}"
    exit 1
    ;;
esac
