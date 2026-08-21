#!/usr/bin/env bash
set -euo pipefail

auth_file="${CODEX_AUTH_FILE:-$HOME/.codex/auth.json}"
channel_id="${CODEX_CHANNEL_ID:-5}"
router_container="${ROUTER_CONTAINER:-router-ai-atius}"
router_unit="${ROUTER_UNIT:-container-router-ai-atius.service}"
mode="${1:---check}"

if [[ "$mode" != "--check" && "$mode" != "--apply" ]]; then
  echo "usage: $0 [--check|--apply]" >&2
  exit 2
fi
if [[ ! "$channel_id" =~ ^[0-9]+$ ]]; then
  echo "atius-codex-credential-sync: CODEX_CHANNEL_ID must be numeric" >&2
  exit 2
fi

for command in jq psql podman systemctl date md5sum; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "atius-codex-credential-sync: missing command: $command" >&2
    exit 1
  fi
done

if [[ ! -r "$auth_file" ]]; then
  echo "atius-codex-credential-sync: auth file is not readable: $auth_file" >&2
  exit 1
fi

runtime_dir="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
lock_file="$runtime_dir/atius-codex-channel-sync.lock"
mkdir -p "$runtime_dir"
exec 9>"$lock_file"
flock --wait 60 9

auth_mode="$(jq -r '.auth_mode // empty' "$auth_file")"
account_id="$(jq -r '.tokens.account_id // empty' "$auth_file")"
last_refresh="$(jq -r '.last_refresh // empty' "$auth_file")"
access_token="$(jq -r '.tokens.access_token // empty' "$auth_file")"
refresh_token="$(jq -r '.tokens.refresh_token // empty' "$auth_file")"
id_token="$(jq -r '.tokens.id_token // empty' "$auth_file")"

if [[ "$auth_mode" != "chatgpt" || -z "$account_id" || -z "$last_refresh" || -z "$access_token" || -z "$refresh_token" || -z "$id_token" ]]; then
  echo "atius-codex-credential-sync: incomplete ChatGPT-managed Codex credential" >&2
  exit 1
fi

access_exp="$(jq -r '.tokens.access_token | split(".")[1] | @base64d | fromjson | .exp // empty' "$auth_file")"
email="$(jq -r '.tokens.id_token | split(".")[1] | @base64d | fromjson | .email // empty' "$auth_file")"
if [[ ! "$access_exp" =~ ^[0-9]+$ ]] || (( access_exp <= $(date +%s) + 300 )); then
  echo "atius-codex-credential-sync: local access token is expired or expires too soon" >&2
  exit 1
fi
expired="$(date -u -d "@$access_exp" +'%Y-%m-%dT%H:%M:%S.000Z')"

sql_dsn="$(
  /usr/bin/podman inspect "$router_container" |
    jq -r '.[0].Config.Env[] | select(startswith("SQL_DSN=")) | sub("^SQL_DSN="; "")'
)"
if [[ -z "$sql_dsn" || "$sql_dsn" == "null" ]]; then
  echo "atius-codex-credential-sync: SQL_DSN not found in $router_container" >&2
  exit 1
fi

IFS='|' read -r db_account db_access_md5 db_last_refresh db_credential_source <<<"$(
  psql -X -qAt "$sql_dsn" -c "
    SELECT
      COALESCE((key::jsonb)->>'account_id', '') || '|' ||
      md5(COALESCE((key::jsonb)->>'access_token', '')) || '|' ||
      COALESCE((key::jsonb)->>'last_refresh', '') || '|' ||
      COALESCE((NULLIF(setting, '')::jsonb)->>'codex_credential_source', '')
    FROM channels
    WHERE id = $channel_id AND type = 57;
  "
)"

if [[ -z "$db_account" ]]; then
  echo "atius-codex-credential-sync: Codex channel $channel_id was not found" >&2
  exit 1
fi
if [[ "$db_account" != "$account_id" ]]; then
  echo "atius-codex-credential-sync: account mismatch; refusing credential replacement" >&2
  exit 1
fi

local_access_md5="$(printf '%s' "$access_token" | md5sum | awk '{print $1}')"
if [[ "$local_access_md5" == "$db_access_md5" && "$db_credential_source" == "external_file" ]]; then
  echo "atius-codex-credential-sync: channel=$channel_id action=noop last_refresh=$db_last_refresh"
  exit 0
fi

if [[ "$mode" == "--check" ]]; then
  echo "atius-codex-credential-sync: channel=$channel_id action=would-update last_refresh=$last_refresh expires_at=$expired"
  exit 0
fi

umask 077
secret_file="$(mktemp "$runtime_dir/atius-codex-channel-key.XXXXXX")"
trap '/usr/bin/unlink "$secret_file"' EXIT

jq -cn \
  --arg id_token "$id_token" \
  --arg access_token "$access_token" \
  --arg refresh_token "$refresh_token" \
  --arg account_id "$account_id" \
  --arg last_refresh "$last_refresh" \
  --arg email "$email" \
  --arg expired "$expired" \
  '{
    id_token: $id_token,
    access_token: $access_token,
    refresh_token: $refresh_token,
    account_id: $account_id,
    last_refresh: $last_refresh,
    email: $email,
    type: "codex",
    expired: $expired
  }' >"$secret_file"

update_result="$(
  psql -X -v ON_ERROR_STOP=1 -qAt \
    -v account_id="$account_id" \
    -v access_md5="$local_access_md5" \
    "$sql_dsn" <<SQL
BEGIN;
CREATE TEMP TABLE codex_key_import(payload text NOT NULL);
\copy codex_key_import(payload) FROM '$secret_file'
UPDATE channels
SET key = (SELECT payload FROM codex_key_import LIMIT 1),
    setting = jsonb_set(
      jsonb_set(
        COALESCE(NULLIF(setting, '')::jsonb, '{}'::jsonb),
        '{codex_credential_health}',
        '{"last_probe_status":"pending","last_upstream_status":0,"requires_regeneration":false}'::jsonb,
        true
      ),
      '{codex_credential_source}',
      '"external_file"'::jsonb,
      true
    )::text
WHERE id = $channel_id
  AND type = 57
  AND (key::jsonb)->>'account_id' = :'account_id';
SELECT CASE WHEN COUNT(*) = 1 THEN 'updated=1' ELSE 'updated=0' END
FROM channels
WHERE id = $channel_id
  AND type = 57
  AND md5(COALESCE((key::jsonb)->>'access_token', '')) = :'access_md5';
COMMIT;
SQL
)"
if [[ "$update_result" != "updated=1" ]]; then
  echo "atius-codex-credential-sync: credential update was not confirmed" >&2
  exit 1
fi

systemctl --user restart "$router_unit"
for _ in $(seq 1 45); do
  if [[ "$(systemctl --user is-active "$router_unit" 2>/dev/null || true)" == "active" ]] &&
    /usr/bin/curl -fsS --max-time 3 http://127.0.0.1:3000/api/status >/dev/null 2>&1; then
    echo "atius-codex-credential-sync: channel=$channel_id action=updated last_refresh=$last_refresh expires_at=$expired"
    exit 0
  fi
  sleep 2
done

echo "atius-codex-credential-sync: router did not become healthy after credential update" >&2
exit 1
