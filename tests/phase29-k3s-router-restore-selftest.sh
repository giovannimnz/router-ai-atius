#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

scripts/k3s-router-backup.sh --self-test
scripts/k3s-router-restore-rehearsal.sh --self-test

run_signal_case() {
  local signal="$1" stubborn="${2:-0}" signal_dir signal_pid active_pgid alive
  signal_dir="$(mktemp -d)"
  mkdir -m 700 "$signal_dir/home"
  env HOME="$signal_dir/home" PHASE29_SIGNAL_TEST_DIR="$signal_dir" PHASE29_SIGNAL_STUBBORN="$stubborn" \
    PHASE29_ACTIVE_PID_FILE="$signal_dir/active-pgid" \
    python3 -c 'import os, signal, sys; signal.signal(signal.SIGINT, signal.SIG_DFL); signal.signal(signal.SIGTERM, signal.SIG_DFL); os.execv(sys.argv[1], sys.argv[1:])' \
    scripts/k3s-router-restore-rehearsal.sh --self-test-signal >/dev/null 2>&1 &
  signal_pid=$!
  for _ in $(seq 1 50); do [ -f "$signal_dir/ready" ] && [ -f "$signal_dir/active-pgid" ] && break; sleep 0.02; done
  if [ ! -f "$signal_dir/ready" ] || [ ! -f "$signal_dir/active-pgid" ]; then fail "$signal child did not become ready"; fi
  active_pgid="$(cat "$signal_dir/active-pgid")"
  if env HOME="$signal_dir/home" PHASE29_SIGNAL_TEST_DIR="$signal_dir" timeout 2 scripts/k3s-router-restore-rehearsal.sh --self-test-signal >/dev/null 2>&1; then
    fail 'concurrent restore acquired the same evidence lock'
  fi
  kill -s "$signal" "$signal_pid"
  alive=true
  for _ in $(seq 1 150); do
    if ! kill -0 "$signal_pid" 2>/dev/null; then alive=false; break; fi
    sleep 0.02
  done
  if $alive; then kill -KILL "$signal_pid" 2>/dev/null || true; fail "$signal did not terminate restore promptly"; fi
  if wait "$signal_pid"; then fail "$signal child exited successfully"; fi
  if kill -0 "-$active_pgid" 2>/dev/null; then fail "$signal left its active process group alive"; fi
  jq -e '.status == "no-go" and .restore_passed == false' "$signal_dir/restore.json" >/dev/null ||
    fail "$signal did not persist no-go evidence"
  rm -rf "$signal_dir"
}

run_signal_case TERM
run_signal_case INT
run_signal_case TERM 1

run_distinct_lock_case() {
  local first second shared_home pid
  first="$(mktemp -d)"; second="$(mktemp -d)"
  shared_home="$(mktemp -d)"
  env HOME="$shared_home" PHASE29_SIGNAL_TEST_DIR="$first" PHASE29_ACTIVE_PID_FILE="$first/active-pgid" \
    python3 -c 'import os, signal, sys; signal.signal(signal.SIGINT, signal.SIG_DFL); signal.signal(signal.SIGTERM, signal.SIG_DFL); os.execv(sys.argv[1], sys.argv[1:])' \
    scripts/k3s-router-restore-rehearsal.sh --self-test-signal >/dev/null 2>&1 &
  pid=$!
  for _ in $(seq 1 50); do [ -f "$first/ready" ] && break; sleep 0.02; done
  [ -f "$first/ready" ] || fail 'global lock holder did not become ready'
  if env HOME="$shared_home" PHASE29_SIGNAL_TEST_DIR="$second" timeout 2 scripts/k3s-router-restore-rehearsal.sh --self-test-signal >/dev/null 2>&1; then
    fail 'distinct evidence directories bypassed the target-global restore lock'
  fi
  kill -TERM "$pid"
  if wait "$pid"; then fail 'global lock holder exited successfully after TERM'; fi
  jq -e '.status == "no-go"' "$first/restore.json" >/dev/null || fail 'global lock holder did not record no-go'
  rm -rf "$first" "$second" "$shared_home"
}

run_distinct_lock_case

run_sequential_lineage_case() {
  local first second shared_home pid
  first="$(mktemp -d)"; second="$(mktemp -d)"; shared_home="$(mktemp -d)"
  env HOME="$shared_home" PHASE29_SIGNAL_TEST_DIR="$first" PHASE29_ACTIVE_PID_FILE="$first/active-pgid" \
    scripts/k3s-router-restore-rehearsal.sh --self-test-signal >/dev/null 2>&1 &
  pid=$!
  for _ in $(seq 1 50); do [ -f "$first/ready" ] && break; sleep 0.02; done
  [ -f "$first/ready" ] || fail 'sequential lineage holder did not become ready'
  kill -TERM "$pid"
  if wait "$pid"; then fail 'sequential lineage holder exited successfully after TERM'; fi
  if env HOME="$shared_home" PHASE29_SIGNAL_TEST_DIR="$second" timeout 2 \
    scripts/k3s-router-restore-rehearsal.sh --self-test-signal >/dev/null 2>&1; then
    fail 'fresh evidence directory bypassed canonical no-go retry lineage'
  fi
  jq '.status = "go"' "$first/restore.json" > "$first/go.json"
  mv "$first/go.json" "$first/restore.json"
  chmod 600 "$first/restore.json"
  jq --arg sha "$(sha256sum "$first/restore.json" | awk '{print $1}')" \
    '.status = "go" | .evidence_sha256 = $sha' \
    "$shared_home/.local/state/router-ai-atius/phase29/restore-target-state.json" > "$shared_home/state.json"
  mv "$shared_home/state.json" "$shared_home/.local/state/router-ai-atius/phase29/restore-target-state.json"
  chmod 600 "$shared_home/.local/state/router-ai-atius/phase29/restore-target-state.json"
  if env HOME="$shared_home" PHASE29_SIGNAL_TEST_DIR="$second" timeout 2 \
    scripts/k3s-router-restore-rehearsal.sh --self-test-signal >/dev/null 2>&1; then
    fail 'fresh evidence directory bypassed canonical successful restore state'
  fi
  rm -rf "$first" "$second" "$shared_home"
}

run_sequential_lineage_case

grep -Fq 'image: docker.io/library/postgres@sha256:b797483593b82cbea9a7ee41c88f324a90d10d9c2504d40e755d91c75456366d' k8s/router-ai-atius/postgres.yaml ||
  fail 'k3s target PostgreSQL image is not pinned by digest'
grep -Fq 'cpu: 500m' k8s/router-ai-atius/postgres.yaml ||
  fail 'PostgreSQL pod does not retain the 500m CPU contract'

if rg -n 'podman[[:space:]]+exec[[:space:]]+postgres.*pg_dump' \
  scripts/k3s-router-backup.sh scripts/k3s-router-restore-rehearsal.sh >/dev/null; then
  fail 'Podman PostgreSQL must never be a backup source'
fi

grep -Fq 'k3s-router-apply-shadow.sh' scripts/k3s-router-restore-rehearsal.sh ||
  fail 'restore does not delegate the staged PostgreSQL apply'
grep -Fq -- '--stage postgres' scripts/k3s-router-restore-rehearsal.sh ||
  fail 'restore does not request the PostgreSQL-only stage'
grep -Eq 'psql .*-(X|-X).*ON_ERROR_STOP' scripts/k3s-router-restore-rehearsal.sh ||
  fail 'restore is not fail-closed on SQL errors'
grep -Fq -- '--single-transaction' scripts/k3s-router-restore-rehearsal.sh ||
  fail 'restore is not atomic'
grep -Fq '.status = "no-go"' scripts/k3s-router-restore-rehearsal.sh ||
  fail 'restore failure does not persist no-go evidence'
grep -Fq 'DROP SCHEMA public RESTRICT' scripts/k3s-router-restore-rehearsal.sh ||
  fail 'restore does not prove the entire public schema is clean'
grep -Fq -- '--retry-no-go' scripts/k3s-router-restore-rehearsal.sh ||
  fail 'restore has no controlled retry path after no-go evidence'

if rg -n '(POSTGRES_PASSWORD=|postgresql://|Secret[[:space:]]+YAML)' \
  scripts/k3s-router-backup.sh scripts/k3s-router-restore-rehearsal.sh >/dev/null; then
  fail 'scripts contain a secret-bearing output pattern'
fi

echo 'phase29 backup/restore self-tests: PASS'
