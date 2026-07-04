#!/usr/bin/env bash
set -euo pipefail

MODEL="${MODEL:-all}"
SIZES="${SIZES-small 10000 50000 100000 250000 300000 500000 750000 1000000}"
LOG_DIR="${LOG_DIR:-logs/long-context-aliases}"
MAX_COMPLETION_TOKENS="${MAX_COMPLETION_TOKENS:-256}"
REQUEST_TIMEOUT_SECONDS="${REQUEST_TIMEOUT_SECONDS:-1800}"
CONNECT_TIMEOUT_SECONDS="${CONNECT_TIMEOUT_SECONDS:-30}"
RUN_STREAM_SMOKE="${RUN_STREAM_SMOKE:-1}"
SKIP_PREFLIGHT="${SKIP_PREFLIGHT:-1}"
ENABLE_1M="${ENABLE_1M:-}"
BASE_EXPECT_REJECT_FROM="${BASE_EXPECT_REJECT_FROM:-300000}"
AUTO_CONFIRM_LARGE_STEPS="${AUTO_CONFIRM_LARGE_STEPS:-}"

if [[ -z "${ROUTER_BASE_URL:-}" ]]; then
  echo "ROUTER_BASE_URL is required" >&2
  exit 2
fi

if [[ -z "${ROUTER_TEST_KEY:-}" ]]; then
  echo "ROUTER_TEST_KEY is required" >&2
  exit 2
fi

case "$MODEL" in
  all|aliases|base|gpt-5.5|gpt-5.5-1m|gpt-5.4|gpt-5.4-1m) ;;
  *)
    echo "MODEL must be all, aliases, base, gpt-5.5, gpt-5.5-1m, gpt-5.4, or gpt-5.4-1m" >&2
    exit 2
    ;;
esac

mkdir -p "$LOG_DIR"
run_id="$(date -u +%Y%m%dT%H%M%SZ)"
log_file="$LOG_DIR/${run_id}-long-context-aliases.jsonl"

echo "Model selection: $MODEL"
echo "Base URL: $ROUTER_BASE_URL"
echo "Sizes: $SIZES"
echo "Catalog preflight: $([[ "$SKIP_PREFLIGHT" == "1" ]] && echo "skipped" || echo "enabled")"
echo "Base model reject threshold: ${BASE_EXPECT_REJECT_FROM:-disabled}"
echo "Log file: $log_file"
echo "Warning: this sends real requests and may consume paid quota."
echo "ROUTER_TEST_KEY will not be printed."

models_for_run() {
  case "$MODEL" in
    all)
      printf '%s\n' "gpt-5.5" "gpt-5.5-1m" "gpt-5.4" "gpt-5.4-1m"
      ;;
    aliases)
      printf '%s\n' "gpt-5.5-1m" "gpt-5.4-1m"
      ;;
    base)
      printf '%s\n' "gpt-5.5" "gpt-5.4"
      ;;
    *)
      printf '%s\n' "$MODEL"
      ;;
  esac
}

is_number() {
  [[ "$1" =~ ^[0-9]+$ ]]
}

is_base_model() {
  [[ "$1" == "gpt-5.5" || "$1" == "gpt-5.4" ]]
}

expected_outcome_for() {
  local model="$1"
  local size="$2"
  if is_base_model "$model" && [[ -n "$BASE_EXPECT_REJECT_FROM" ]] && is_number "$size" && (( size >= BASE_EXPECT_REJECT_FROM )); then
    printf 'reject'
    return
  fi
  printf 'accept'
}

confirm_large_step() {
  local model="$1"
  local size="$2"

  if [[ "$size" == "1000000" && "$ENABLE_1M" != "YES_I_ACCEPT_COSTS" ]]; then
    echo "Skipped 1000000 for $model. Set ENABLE_1M=YES_I_ACCEPT_COSTS and confirm manually to run it."
    return 1
  fi

  if is_number "$size" && (( size >= 250000 )); then
    if [[ "$AUTO_CONFIRM_LARGE_STEPS" == "YES_I_ACCEPT_COSTS" ]]; then
      echo "Auto-confirmed ${size} for ${model} via AUTO_CONFIRM_LARGE_STEPS."
      return 0
    fi
    echo
    echo "About to run an expensive approximate ${size}-token request for $model."
    echo "This may cost real money and may take several minutes."
    if [[ -r /dev/tty ]]; then
      read -r -p "Type RUN ${size} ${model} to continue: " answer < /dev/tty
    else
      echo "Skipped ${size} for ${model}: no TTY for manual confirmation." >&2
      return 1
    fi
    if [[ "$answer" != "RUN ${size} ${model}" ]]; then
      echo "Skipped ${size} for ${model}."
      return 1
    fi
  fi

  return 0
}

curl_common() {
  curl -sS \
    --connect-timeout "$CONNECT_TIMEOUT_SECONDS" \
    --max-time "$REQUEST_TIMEOUT_SECONDS" \
    "$@"
}

log_json() {
  local json_line="$1"
  printf '%s\n' "$json_line" | tee -a "$log_file"
}

preflight_models() {
  local tmp_response
  tmp_response="$(mktemp)"
  trap 'rm -f "$tmp_response"' RETURN

  local http_status
  http_status="$(
    curl_common -o "$tmp_response" -w "%{http_code}" \
      "$ROUTER_BASE_URL/v1/models" \
      -H "Authorization: Bearer $ROUTER_TEST_KEY"
  )"

  python3 - "$http_status" "$tmp_response" "$log_file" "$(models_for_run | paste -sd, -)" <<'PY'
import json
import sys
from pathlib import Path

status, response_path, log_path, models_csv = sys.argv[1:]
raw = Path(response_path).read_text(errors="replace")
models = [m for m in models_csv.split(",") if m]
expected = {
    "gpt-5.5": {"input_price": 5, "output_price": 30, "upstream": "gpt-5.5"},
    "gpt-5.5-1m": {"input_price": 10, "output_price": 45, "upstream": "gpt-5.5"},
    "gpt-5.4": {"input_price": 2.5, "output_price": 15, "upstream": "gpt-5.4"},
    "gpt-5.4-1m": {"input_price": 5, "output_price": 22.5, "upstream": "gpt-5.4"},
}
entry = {
    "kind": "preflight_models",
    "http_status": int(status),
    "models": models,
    "passed": False,
    "errors": [],
}
try:
    body = json.loads(raw)
    data = {item.get("id"): item for item in body.get("data", [])}
    for model in models:
        item = data.get(model)
        if item is None:
            entry["errors"].append(f"{model} missing from /v1/models")
            continue
        want = expected[model]
        if item.get("owned_by") != "codex":
            entry["errors"].append(f"{model} owned_by={item.get('owned_by')!r}")
        if item.get("provider") != "OpenAI Codex":
            entry["errors"].append(f"{model} provider={item.get('provider')!r}")
        if item.get("endpoint_routes", {}).get("openai") != "/v1/chat/completions":
            entry["errors"].append(f"{model} openai route mismatch")
        if item.get("input_price") != want["input_price"]:
            entry["errors"].append(f"{model} input_price={item.get('input_price')!r}")
        if item.get("output_price") != want["output_price"]:
            entry["errors"].append(f"{model} output_price={item.get('output_price')!r}")
        pricing = item.get("pricing", {})
        if pricing.get("input") != want["input_price"] or pricing.get("output") != want["output_price"]:
            entry["errors"].append(f"{model} pricing object mismatch")
        if "default" not in item.get("enable_groups", []):
            entry["errors"].append(f"{model} default group missing")
except Exception as exc:
    entry["errors"].append(f"failed to parse /v1/models: {exc}")

entry["passed"] = int(status) == 200 and not entry["errors"]
with open(log_path, "a", encoding="utf-8") as f:
    f.write(json.dumps(entry, ensure_ascii=False) + "\n")
print(json.dumps(entry, ensure_ascii=False))
sys.exit(0 if entry["passed"] else 1)
PY
}

write_reasoning_request() {
  local model="$1"
  local size="$2"
  local body_path="$3"
  local expected_path="$4"

  python3 - "$model" "$size" "$MAX_COMPLETION_TOKENS" "$body_path" "$expected_path" <<'PY'
import json
import sys
from pathlib import Path

model, size_label, max_tokens, body_path, expected_path = sys.argv[1:]
max_tokens = int(max_tokens)
numeric_size = None if size_label == "small" else int(size_label)

start_anchor = f"ANCHOR_START_{model}_{size_label}_A17"
middle_anchor = f"ANCHOR_MIDDLE_{model}_{size_label}_B29"
end_anchor = f"ANCHOR_END_{model}_{size_label}_C43"
expected_code = 17 + 29 + 43

instructions = (
    "You are validating long-context retrieval and reasoning. "
    "Read the full context. Return compact JSON only with keys "
    "start_anchor, middle_anchor, end_anchor, verification_code, verdict. "
    f"The verification_code is the sum of the three anchor numbers: {expected_code}. "
    "Use the exact anchor strings from the context."
)

if numeric_size is None:
    context = (
        f"{start_anchor}\n"
        "Small smoke context for alias routing and billing validation.\n"
        f"{middle_anchor}\n"
        "The answer must preserve every anchor exactly.\n"
        f"{end_anchor}\n"
    )
else:
    # Approximate tokens: common short words are intentionally used for stable,
    # predictable payload growth. The target is approximate, not a tokenizer claim.
    word_count = max(1, int(numeric_size * 0.79))
    vocab = "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu".split()
    first = word_count // 3
    second = word_count // 3
    third = word_count - first - second

    def words(n: int) -> str:
        return " ".join(vocab[i % len(vocab)] for i in range(n))

    context = (
        f"{start_anchor}\n{words(first)}\n"
        f"{middle_anchor}\n{words(second)}\n"
        f"{end_anchor}\n{words(third)}\n"
    )

request = {
    "model": model,
    "messages": [
        {"role": "system", "content": "Return only the requested compact JSON."},
        {"role": "user", "content": f"{instructions}\n\nCONTEXT:\n{context}"},
    ],
    "max_completion_tokens": max_tokens,
    "stream": False,
}
expected = {
    "model": model,
    "size": size_label,
    "start_anchor": start_anchor,
    "middle_anchor": middle_anchor,
    "end_anchor": end_anchor,
    "verification_code": expected_code,
    "request_bytes": len(json.dumps(request, ensure_ascii=False).encode("utf-8")),
}

Path(body_path).write_text(json.dumps(request, ensure_ascii=False), encoding="utf-8")
Path(expected_path).write_text(json.dumps(expected, ensure_ascii=False), encoding="utf-8")
PY
}

evaluate_chat_response() {
  local model="$1"
  local size="$2"
  local http_status="$3"
  local duration_ms="$4"
  local response_path="$5"
  local expected_path="$6"
  local expected_outcome="$7"

  python3 - "$model" "$size" "$http_status" "$duration_ms" "$response_path" "$expected_path" "$expected_outcome" "$log_file" <<'PY'
import json
import sys
from pathlib import Path

model, size, status, duration_ms, response_path, expected_path, expected_outcome, log_path = sys.argv[1:]
raw = Path(response_path).read_text(errors="replace")
expected = json.loads(Path(expected_path).read_text())

try:
    body = json.loads(raw)
except json.JSONDecodeError:
    body = {"raw": raw[:2000]}

content = ""
try:
    content = body["choices"][0]["message"].get("content") or ""
except Exception:
    pass

required = [
    expected["start_anchor"],
    expected["middle_anchor"],
    expected["end_anchor"],
    str(expected["verification_code"]),
]
missing = [item for item in required if item not in content]
is_reject_expected = expected_outcome == "reject"
status_code = int(status)
response_model = body.get("model")
if not is_reject_expected and response_model != model:
    missing.append(f"response_model={response_model!r}")
passed = status_code == 200 and not missing and not body.get("error")
if is_reject_expected:
    passed = status_code != 200 and bool(body.get("error"))
entry = {
    "kind": "base_limit_guard" if is_reject_expected else "chat_reasoning",
    "model": model,
    "size": size,
    "expected_outcome": expected_outcome,
    "http_status": status_code,
    "duration_ms": int(duration_ms),
    "request_bytes": expected["request_bytes"],
    "response_model": response_model,
    "usage": body.get("usage"),
    "error": body.get("error"),
    "passed": passed,
    "missing": [] if is_reject_expected else missing,
    "response_excerpt": content[:1000],
}
with open(log_path, "a", encoding="utf-8") as f:
    f.write(json.dumps(entry, ensure_ascii=False) + "\n")
print(json.dumps(entry, ensure_ascii=False))
PY
}

run_reasoning_step() {
  local model="$1"
  local size="$2"

  confirm_large_step "$model" "$size" || return 0

  local tmp_body tmp_expected tmp_response
  tmp_body="$(mktemp)"
  tmp_expected="$(mktemp)"
  tmp_response="$(mktemp)"
  trap 'rm -f "$tmp_body" "$tmp_expected" "$tmp_response"' RETURN

  write_reasoning_request "$model" "$size" "$tmp_body" "$tmp_expected"

  local start_ms end_ms duration_ms http_status
  start_ms="$(date +%s%3N)"
  http_status="$(
    curl_common -o "$tmp_response" -w "%{http_code}" \
      "$ROUTER_BASE_URL/v1/chat/completions" \
      -H "Authorization: Bearer $ROUTER_TEST_KEY" \
      -H "Content-Type: application/json" \
      --data-binary "@$tmp_body"
  )"
  end_ms="$(date +%s%3N)"
  duration_ms=$((end_ms - start_ms))

  evaluate_chat_response "$model" "$size" "$http_status" "$duration_ms" "$tmp_response" "$tmp_expected" "$(expected_outcome_for "$model" "$size")"
}

run_stream_smoke() {
  local model="$1"
  local tmp_body tmp_response
  tmp_body="$(mktemp)"
  tmp_response="$(mktemp)"
  trap 'rm -f "$tmp_body" "$tmp_response"' RETURN

  python3 - "$model" "$tmp_body" <<'PY'
import json
import sys
from pathlib import Path

model, body_path = sys.argv[1:]
request = {
    "model": model,
    "messages": [{"role": "user", "content": "Count from 1 to 5. Return only the numbers."}],
    "max_completion_tokens": 64,
    "stream": True,
}
Path(body_path).write_text(json.dumps(request, ensure_ascii=False), encoding="utf-8")
PY

  local start_ms end_ms duration_ms http_status
  start_ms="$(date +%s%3N)"
  http_status="$(
    curl_common -N -o "$tmp_response" -w "%{http_code}" \
      "$ROUTER_BASE_URL/v1/chat/completions" \
      -H "Authorization: Bearer $ROUTER_TEST_KEY" \
      -H "Content-Type: application/json" \
      --data-binary "@$tmp_body"
  )"
  end_ms="$(date +%s%3N)"
  duration_ms=$((end_ms - start_ms))

  python3 - "$model" "$http_status" "$duration_ms" "$tmp_response" "$log_file" <<'PY'
import json
import sys
from pathlib import Path

model, status, duration_ms, response_path, log_path = sys.argv[1:]
raw = Path(response_path).read_text(errors="replace")
entry = {
    "kind": "stream_smoke",
    "model": model,
    "http_status": int(status),
    "duration_ms": int(duration_ms),
    "passed": int(status) == 200 and "data:" in raw,
    "response_excerpt": raw[:1000],
}
with open(log_path, "a", encoding="utf-8") as f:
    f.write(json.dumps(entry, ensure_ascii=False) + "\n")
print(json.dumps(entry, ensure_ascii=False))
PY
}

if [[ "$SKIP_PREFLIGHT" != "1" ]]; then
  preflight_models
fi

while IFS= read -r model; do
  if [[ "$RUN_STREAM_SMOKE" == "1" ]]; then
    run_stream_smoke "$model"
  fi
  for size in $SIZES; do
    run_reasoning_step "$model" "$size"
  done
done < <(models_for_run)

echo "Done. Results appended to $log_file"
