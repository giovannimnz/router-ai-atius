#!/usr/bin/env python3
"""Operational battery for Atius router models, embeddings and MiniMax rate behavior."""

from __future__ import annotations

import argparse
import json
import os
import statistics
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_BASE_URL = "http://127.0.0.1:3001/v1"
DB_CONTAINER = os.environ.get("CLIANYTHING_DB_CONTAINER", "postgres")
DB_NAME = os.environ.get("CLIANYTHING_DB_NAME", "DBRouterAiAtius")
DB_USER = os.environ.get("CLIANYTHING_DB_USER", "admin")
USER_AGENT = os.environ.get("ATIUS_ROUTER_USER_AGENT", "Mozilla/5.0 AtiusRouterSmoke/1.0")
UPSTREAM_CODES = {400, 402, 408, 409, 429, 500, 502, 503, 504, 529}


@dataclass
class Result:
    family: str
    model: str
    endpoint: str
    status: str
    http: int | str
    latency_ms: int
    detail: str


def _short(text: object, limit: int = 180) -> str:
    value = str(text or "").replace("\n", " ").strip()
    return value[:limit] if value else "<empty>"


def _run(cmd: list[str], *, input_text: str | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        cmd,
        cwd=REPO_ROOT,
        input=input_text,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
        timeout=30,
    )


def token_from_db(token_id: int | None, token_name: str | None) -> str | None:
    where = ""
    if token_id is not None:
        where = f"id = {int(token_id)}"
    elif token_name:
        safe = token_name.replace("'", "''")
        where = f"name = '{safe}'"
    else:
        return None
    sql = f"select key from tokens where deleted_at is null and status = 1 and {where} order by id asc limit 1"
    proc = _run(
        [
            "podman",
            "exec",
            "-i",
            DB_CONTAINER,
            "psql",
            "-U",
            DB_USER,
            "-d",
            DB_NAME,
            "-X",
            "-q",
            "-t",
            "-A",
            "-c",
            sql,
        ]
    )
    if proc.returncode != 0:
        raise RuntimeError(_short(proc.stderr or proc.stdout))
    token = proc.stdout.strip()
    return token or None


def resolve_token(args: argparse.Namespace) -> str:
    token = os.environ.get("ATIUS_ROUTER_TOKEN", "").strip()
    if token:
        return token
    token = token_from_db(args.token_id, args.token_name)
    if token:
        return token
    raise RuntimeError("ATIUS_ROUTER_TOKEN ausente; informe --token-id ou --token-name.")


def providers() -> list[dict[str, Any]]:
    proc = _run([str(REPO_ROOT / "bin" / "clianything"), "providers", "--all", "--format", "json"])
    if proc.returncode != 0:
        raise RuntimeError(_short(proc.stderr or proc.stdout))
    return json.loads(proc.stdout)


def request_json(base_url: str, path: str, token: str, payload: dict[str, Any]) -> tuple[int | str, Any, str, int]:
    url = base_url.rstrip("/") + path
    body = json.dumps(payload).encode("utf-8")
    req = Request(
        url,
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": USER_AGENT,
        },
        method="POST",
    )
    started = time.perf_counter()
    try:
        with urlopen(req, timeout=45.0) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            latency = int((time.perf_counter() - started) * 1000)
            try:
                return resp.status, json.loads(raw), raw, latency
            except json.JSONDecodeError:
                return resp.status, None, raw, latency
    except HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        latency = int((time.perf_counter() - started) * 1000)
        try:
            return exc.code, json.loads(raw), raw, latency
        except json.JSONDecodeError:
            return exc.code, None, raw, latency
    except URLError as exc:
        latency = int((time.perf_counter() - started) * 1000)
        return "network", None, str(exc), latency


def classify(code: int | str, payload: Any, raw: str) -> tuple[str, str]:
    if code == 200:
        return "ok", "ok"
    message = raw
    if isinstance(payload, dict):
        message = payload.get("error", {}).get("message") or payload.get("message") or raw
    if code in UPSTREAM_CODES:
        return "upstream", _short(f"HTTP {code} {message}")
    return "fail", _short(f"HTTP {code} {message}")


def test_openai_chat(base_url: str, token: str, model: str) -> Result:
    payload: dict[str, Any] = {
        "model": model,
        "messages": [{"role": "user", "content": "Responda apenas OK."}],
        "max_tokens": 8,
    }
    if model.startswith("gpt-"):
        payload["stream"] = True
    code, data, raw, latency = request_json(base_url, "/chat/completions", token, payload)
    status, detail = classify(code, data, raw)
    if code == 200 and model.startswith("gpt-"):
        detail = "stream accepted"
    return Result("openai-chat", model, "/v1/chat/completions", status, code, latency, detail)


def test_anthropic_messages(base_url: str, token: str, model: str) -> Result:
    payload = {
        "model": model,
        "max_tokens": 8,
        "messages": [{"role": "user", "content": "Responda apenas OK."}],
    }
    code, data, raw, latency = request_json(base_url, "/messages", token, payload)
    status, detail = classify(code, data, raw)
    return Result("anthropic-messages", model, "/v1/messages", status, code, latency, detail)


def test_embedding(base_url: str, token: str, model: str, embedding_type: str | None = None) -> Result:
    payload: dict[str, Any] = {"model": model, "input": "hello"}
    label = model
    if model == "embo-01":
        payload["type"] = embedding_type or "query"
        label = f"{model}:{payload['type']}"
    code, data, raw, latency = request_json(base_url, "/embeddings", token, payload)
    status, detail = classify(code, data, raw)
    if code == 200 and isinstance(data, dict):
        rows = data.get("data") or []
        vector = rows[0].get("embedding") if rows and isinstance(rows[0], dict) else None
        if isinstance(vector, list):
            detail = f"dimension={len(vector)}"
        else:
            status, detail = "fail", "missing embedding vector"
    return Result("embeddings", label, "/v1/embeddings", status, code, latency, detail)


def split_models(value: str) -> list[str]:
    return [item.strip() for item in str(value or "").split(",") if item.strip()]


def build_cases(provider_rows: list[dict[str, Any]]) -> tuple[list[str], list[str], list[str]]:
    openai_models: list[str] = []
    anthropic_models: list[str] = []
    embedding_models: list[str] = []
    for row in provider_rows:
        if int(row.get("status") or 0) != 1:
            continue
        name = str(row.get("name") or "")
        models = split_models(str(row.get("models") or ""))
        if "Embeddings" in name:
            embedding_models.extend(models)
        elif "Anthropic-Compatible" in name:
            anthropic_models.extend(models)
        elif "OpenAI-Compatible" in name or "Codex OAuth" in name:
            openai_models.extend(models)
    return sorted(set(openai_models)), sorted(set(anthropic_models)), sorted(set(embedding_models))


def print_table(rows: list[Result]) -> None:
    columns = ["family", "model", "endpoint", "status", "http", "latency_ms", "detail"]
    data = [row.__dict__ for row in rows]
    widths = {col: len(col) for col in columns}
    for row in data:
        for col in columns:
            widths[col] = min(max(widths[col], len(_short(row.get(col)))), 60)
    print("  ".join(col.ljust(widths[col]) for col in columns))
    print("  ".join("-" * widths[col] for col in columns))
    for row in data:
        print("  ".join(_short(row.get(col)).ljust(widths[col]) for col in columns))


def run_rate_probe(base_url: str, token: str, requests: int, delay: float) -> list[Result]:
    rows = []
    for index in range(requests):
        result = test_openai_chat(base_url, token, "MiniMax-M3")
        result.family = "minimax-rate"
        result.model = f"MiniMax-M3#{index + 1}"
        rows.append(result)
        if delay > 0 and index + 1 < requests:
            time.sleep(delay)
    return rows


def print_rate_summary(rows: list[Result]) -> None:
    if not rows:
        return
    latencies = [row.latency_ms for row in rows if isinstance(row.latency_ms, int)]
    counts: dict[str, int] = {}
    for row in rows:
        key = f"{row.status}:{row.http}"
        counts[key] = counts.get(key, 0) + 1
    print("\nrate_summary:")
    print(f"  counts={counts}")
    if latencies:
        print(f"  latency_ms_min={min(latencies)}")
        print(f"  latency_ms_p50={int(statistics.median(latencies))}")
        print(f"  latency_ms_max={max(latencies)}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Run Atius router model battery without printing secrets.")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)
    parser.add_argument("--token-id", type=int)
    parser.add_argument("--token-name")
    parser.add_argument("--rate-requests", type=int, default=8)
    parser.add_argument("--rate-delay", type=float, default=0.25)
    parser.add_argument("--skip-rate", action="store_true")
    args = parser.parse_args()

    token = resolve_token(args)
    provider_rows = providers()
    openai_models, anthropic_models, embedding_models = build_cases(provider_rows)

    results: list[Result] = []
    for model in openai_models:
        results.append(test_openai_chat(args.base_url, token, model))
    for model in anthropic_models:
        results.append(test_anthropic_messages(args.base_url, token, model))
    for model in embedding_models:
        if model == "embo-01":
            results.append(test_embedding(args.base_url, token, model, "query"))
            results.append(test_embedding(args.base_url, token, model, "db"))
        else:
            results.append(test_embedding(args.base_url, token, model))
    if not args.skip_rate:
        results.extend(run_rate_probe(args.base_url, token, args.rate_requests, args.rate_delay))

    print_table(results)
    print_rate_summary([row for row in results if row.family == "minimax-rate"])
    hard_fail = any(row.status == "fail" for row in results)
    return 1 if hard_fail else 0


if __name__ == "__main__":
    raise SystemExit(main())
