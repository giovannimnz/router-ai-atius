#!/usr/bin/env bash
# podman-quadlets-install.sh
#
# One-shot: install the systemd quadlets for a rootless Podman deployment
# (per-user, ~$USER/.config/containers/systemd/).
#
# Rebrand v2.11: service unit names updated to router-ai-atius-* pattern.
#
# Usage:
#   ./scripts/podman-quadlets-install.sh
#
# Pre-conditions:
#   - podman 4.4+ (quadlets shipped in-tree since 4.4)
#   - systemd --user is set up
#   - the user has been `loginctl enable-linger`'d so the user services
#     survive logout.
#
# After this:
#   systemctl --user daemon-reload
#   systemctl --user start \
#     router-ai-atius-db.service \
#     router-ai-atius-redis.service \
#     router-ai-atius.service \
#     router-ai-atius-model-detailed.service
#   systemctl --user status

set -euo pipefail
cd "$(dirname "$0")/.."

# Ensure :latest images exist before enabling the systemd services.
# (systemd will fail to start units that reference missing images.)
if command -v podman >/dev/null 2>&1; then
  if ! podman image exists ghcr.io/giovannimnz/router-ai-atius:latest 2>/dev/null \
     || ! podman image exists router-ai-atius-model-detailed:latest 2>/dev/null; then
    echo "[quadlets-install] :latest images not all present. running prepare-images..."
    ./scripts/podman-prepare-images.sh || {
      echo "ERROR: prepare-images failed; quadlets will not start without images." >&2
    }
  fi
fi

QUADLET_SRC="podman/quadlets"
DEST="${HOME}/.config/containers/systemd"
mkdir -p "$DEST"

# Drop in the .container files. systemd generates the .service from each.
install -m 0644 "$QUADLET_SRC"/*.container "$DEST/"

# Generate the env file from .env (only known-safe keys, no secrets
# embedded; we use the actual POSTGRES_PASSWORD from .env if present).
ENV_FILE="$DEST/router-ai-atius.env"
if [ -f .env ]; then
  grep -E '^(POSTGRES_USER|POSTGRES_PASSWORD|POSTGRES_DB|REDIS_PASSWORD|SESSION_SECRET)=' .env > "$ENV_FILE" || true
  chmod 0600 "$ENV_FILE"
  echo "[quadlets-install] wrote $ENV_FILE (chmod 600)"
fi

systemctl --user daemon-reload
echo "[quadlets-install] quadlets installed in $DEST:"
ls -1 "$DEST"

cat <<'EOF'

Next:
  systemctl --user enable --now \
    router-ai-atius-db.service \
    router-ai-atius-redis.service \
    router-ai-atius.service \
    router-ai-atius-model-detailed.service

  systemctl --user status
EOF
