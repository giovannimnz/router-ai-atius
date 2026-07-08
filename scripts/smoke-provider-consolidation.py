#!/usr/bin/env python3
"""Validate provider consolidation routing without printing secrets."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
CLI = REPO_ROOT / "bin" / "clianything"
def _normalize_base_url(value: str) -> str:
    base_url = value.rstrip("/")
    if base_url.endswith("/v1"):
        return base_url[:-3]
    return base_url


BASE_URL = _normalize_base_url(os.environ.get("ATIUS_ROUTER_BASE_URL", "http://127.0.0.1:3000"))
PUBLIC_BASE_URL = _normalize_base_url(os.environ.get("ATIUS_ROUTER_PUBLIC_BASE_URL", "https://router.atius.com.br"))
MAX_BODY_CHARS = 220
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))
USER_AGENT = "curl/8.5.0"
ACTIVE_ONLY = os.environ.get("ATIUS_ROUTER_ACTIVE_ONLY", "").strip().lower() in {"1", "true", "yes"}


@dataclass(frozen=True)
class Case:
    name: str
    endpoint_type: str
    model: str
    expected_channel_id: int
    expected_channel_name: str
    accepted_codes: set[int]
    stream: bool = False
    negative: bool = False


def _token() -> str:
    token = os.environ.get("ATIUS_ROUTER_TOKEN", "").strip()
    if not token:
        print("Missing ATIUS_ROUTER_TOKEN", file=sys.stderr)
        raise SystemExit(2)
    return token


def _scrub(text: str, token: str) -> str:
    return text.replace(token, "<redacted>")[:MAX_BODY_CHARS].replace("\n", " ")


def _run_json(args: list[str]) -> list[dict[str, Any]]:
    proc = subprocess.run(args, cwd=REPO_ROOT, text=True, capture_output=True, check=False, timeout=20)
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or proc.stdout.strip())
    return json.loads(proc.stdout or "[]")


def _query(sql: str) -> list[dict[str, Any]]:
    return _run_json([str(CLI), "query", sql, "--format", "json"])


def _max_log_id() -> int:
    rows = _query("select coalesce(max(id), 0) as max_id from logs")
    return int(rows[0]["max_id"]) if rows else 0


def _request(path: str, token: str, payload: dict[str, Any] | None, headers: dict[str, str]) -> tuple[int, str]:
    body = None if payload is None else json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        BASE_URL + path,
        data=body,
        headers=headers,
        method="GET" if payload is None else "POST",
    )
    try:
        with OPENER.open(request, timeout=45) as response:
            return response.status, response.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read().decode("utf-8", errors="replace")


def _public_get(path: str, token: str) -> tuple[int, str]:
    request = urllib.request.Request(
        PUBLIC_BASE_URL + path,
        headers={"Authorization": f"Bearer {token}", "Accept": "application/json", "User-Agent": USER_AGENT},
        method="GET",
    )
    try:
        with OPENER.open(request, timeout=45) as response:
            return response.status, response.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read().decode("utf-8", errors="replace")


def _payload(case: Case) -> tuple[str, dict[str, Any], dict[str, str]]:
    marker = f"atius-route-uat-{int(time.time() * 1000)}-{case.name}"
    if case.endpoint_type == "openai":
        payload = {
            "model": case.model,
            "messages": [{"role": "user", "content": f"Reply with ok. {marker}"}],
            "max_tokens": 4,
            "temperature": 0,
            "stream": case.stream,
        }
        return "/v1/chat/completions", payload, {
            "Authorization": f"Bearer {_token()}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": USER_AGENT,
        }
    if case.endpoint_type == "anthropic":
        payload = {
            "model": case.model,
            "max_tokens": 4,
            "messages": [{"role": "user", "content": f"Reply with ok. {marker}"}],
        }
        return "/v1/messages", payload, {
            "x-api-key": _token(),
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": USER_AGENT,
        }
    if case.endpoint_type == "embeddings":
        payload: dict[str, Any] = {"model": case.model, "input": f"hello {marker}"}
        if case.model == "embo-01":
            payload["type"] = "query"
        return "/v1/embeddings", payload, {
            "Authorization": f"Bearer {_token()}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": USER_AGENT,
        }
    raise ValueError(f"unknown endpoint type: {case.endpoint_type}")


def _find_log(after_id: int, case: Case) -> dict[str, Any] | None:
    for _ in range(16):
        rows = _query(
            "select l.id, l.model_name, l.channel_id, coalesce(nullif(l.channel_name, ''), c.name) as channel_name, "
            "l.request_id, l.quota, l.created_at "
            "from logs l left join channels c on c.id = l.channel_id "
            f"where l.id > {after_id} and l.model_name = '{case.model.replace(chr(39), chr(39)+chr(39))}' "
            "order by l.id desc limit 5"
        )
        if rows:
            return rows[0]
        time.sleep(0.5)
    return None


def _catalog_checks(token: str) -> list[tuple[str, bool, str]]:
    checks: list[tuple[str, bool, str]] = []
    required_ids = {"MiniMax-M3", "gpt-5.4"} if ACTIVE_ONLY else {"MiniMax-M3", "text-embedding-3-small"}
    forbidden_ids = {
        "embo-01",
        "deepseek-v4-pro",
        "deepseek-v4-flash",
        "text-embedding-3-small",
        "text-embedding-3-large",
    } if ACTIVE_ONLY else set()
    for label, base in [("local", BASE_URL), ("public", PUBLIC_BASE_URL)]:
        request = urllib.request.Request(
            base + "/v1/models",
            headers={"Authorization": f"Bearer {token}", "Accept": "application/json", "User-Agent": USER_AGENT},
            method="GET",
        )
        try:
            with OPENER.open(request, timeout=45) as response:
                code = response.status
                text = response.read().decode("utf-8", errors="replace")
        except urllib.error.HTTPError as exc:
            code = exc.code
            text = exc.read().decode("utf-8", errors="replace")
        ok = code == 200
        detail = f"HTTP {code}"
        if ok:
            data = json.loads(text)
            ids = [item.get("id") for item in data.get("data", [])]
            leaked = any(any(key in item for key in ("pricing_version", "pricing_source", "pricing_estimated")) for item in data.get("data", []))
            ok = list(data.keys()) == ["data"] and not leaked and required_ids.issubset(ids) and not (forbidden_ids & set(ids))
            detail += f" models={len(ids)} first={ids[:4]} leaked_internal={leaked}"
        checks.append((f"catalog-{label}", ok, detail))

    code, text = _public_get("/v1/models?api_format=anthropic", token)
    ok = code == 200
    detail = f"HTTP {code}"
    if ok:
        data = json.loads(text)
        ids = [item.get("id") for item in data.get("data", [])]
        required_anthropic_ids = {"MiniMax-M3"} if ACTIVE_ONLY else {"MiniMax-M3", "deepseek-v4-pro"}
        forbidden_anthropic_ids = {"embo-01", "gpt-5.4", "deepseek-v4-pro", "deepseek-v4-flash"} if ACTIVE_ONLY else {"embo-01", "gpt-5.4"}
        ok = list(data.keys()) == ["data"] and required_anthropic_ids.issubset(ids) and not (forbidden_anthropic_ids & set(ids))
        detail += f" models={len(ids)} first={ids[:4]} embeddings_present={'embo-01' in ids}"
    checks.append(("catalog-public-anthropic", ok, detail))
    return checks


def _cases() -> list[Case]:
    strict_success = {200}
    active_cases = [
        Case("openai-minimax-m3", "openai", "MiniMax-M3", 1, "MiniMax", strict_success),
        Case("openai-minimax-m27-highspeed", "openai", "MiniMax-M2.7-highspeed", 1, "MiniMax", strict_success),
        Case("openai-minimax-m27", "openai", "MiniMax-M2.7", 1, "MiniMax", strict_success),
        Case("anthropic-minimax-m3", "anthropic", "MiniMax-M3", 1, "MiniMax", strict_success),
        Case("anthropic-minimax-m27-highspeed", "anthropic", "MiniMax-M2.7-highspeed", 1, "MiniMax", strict_success),
        Case("anthropic-minimax-m27", "anthropic", "MiniMax-M2.7", 1, "MiniMax", strict_success),
        Case("openai-codex-gpt55", "openai", "gpt-5.5", 5, "OpenAI - Codex", strict_success, stream=True),
        Case("openai-codex-gpt54", "openai", "gpt-5.4", 5, "OpenAI - Codex", strict_success, stream=True),
        Case("openai-codex-gpt54-mini", "openai", "gpt-5.4-mini", 5, "OpenAI - Codex", strict_success, stream=True),
        Case("openai-codex-spark", "openai", "gpt-5.3-codex-spark", 5, "OpenAI - Codex", strict_success, stream=True),
    ]
    if ACTIVE_ONLY:
        return active_cases
    return active_cases + [
        Case("embeddings-minimax-embo", "embeddings", "embo-01", 1, "MiniMax", strict_success),
        Case("openai-deepseek-pro", "openai", "deepseek-v4-pro", 2, "DeepSeek", strict_success),
        Case("openai-deepseek-flash", "openai", "deepseek-v4-flash", 2, "DeepSeek", strict_success),
        Case("anthropic-deepseek-pro", "anthropic", "deepseek-v4-pro", 2, "DeepSeek", strict_success),
        Case("anthropic-deepseek-flash", "anthropic", "deepseek-v4-flash", 2, "DeepSeek", strict_success),
        Case("embeddings-codex-small", "embeddings", "text-embedding-3-small", 5, "OpenAI - Codex", strict_success),
        Case("embeddings-codex-large", "embeddings", "text-embedding-3-large", 5, "OpenAI - Codex", strict_success),
    ]


def _negative_cases() -> list[Case]:
    if not ACTIVE_ONLY:
        return []
    non_success = set(range(400, 600))
    return [
        Case("disabled-embeddings-minimax-embo", "embeddings", "embo-01", 1, "MiniMax", non_success, negative=True),
        Case("disabled-openai-deepseek-pro", "openai", "deepseek-v4-pro", 2, "DeepSeek", non_success, negative=True),
        Case("disabled-anthropic-deepseek-pro", "anthropic", "deepseek-v4-pro", 2, "DeepSeek", non_success, negative=True),
        Case("disabled-embeddings-codex-small", "embeddings", "text-embedding-3-small", 5, "OpenAI - Codex", non_success, negative=True),
    ]


def main() -> int:
    token = _token()
    failures: list[str] = []
    print(f"catalog checks base={BASE_URL} public={PUBLIC_BASE_URL}", flush=True)
    for name, ok, detail in _catalog_checks(token):
        print(f"{'PASS' if ok else 'FAIL'} {name}: {detail}", flush=True)
        if not ok:
            failures.append(name)

    print("routing checks", flush=True)
    for case in _cases():
        before = _max_log_id()
        path, payload, headers = _payload(case)
        code, body = _request(path, token, payload, headers)
        log = _find_log(before, case)
        route_ok = bool(log) and int(log.get("channel_id") or 0) == case.expected_channel_id and log.get("channel_name") == case.expected_channel_name
        code_ok = code in case.accepted_codes
        ok = route_ok and code_ok
        suffix = ""
        if not ok:
            suffix = " body=" + _scrub(body, token)
            failures.append(case.name)
        log_detail = "no-log" if not log else f"log={log['id']} channel={log['channel_id']}:{log['channel_name']}"
        print(f"{'PASS' if ok else 'FAIL'} {case.name}: HTTP {code} {log_detail}{suffix}", flush=True)

    if ACTIVE_ONLY:
        print("negative routing checks", flush=True)
    for case in _negative_cases():
        before = _max_log_id()
        path, payload, headers = _payload(case)
        code, body = _request(path, token, payload, headers)
        log = _find_log(before, case)
        route_blocked = not log or int(log.get("channel_id") or 0) != case.expected_channel_id
        code_ok = code in case.accepted_codes
        ok = route_blocked and code_ok
        suffix = ""
        if not ok:
            suffix = " body=" + _scrub(body, token)
            failures.append(case.name)
        log_detail = "no-log" if not log else f"log={log['id']} channel={log['channel_id']}:{log['channel_name']}"
        print(f"{'PASS' if ok else 'FAIL'} {case.name}: HTTP {code} {log_detail}{suffix}", flush=True)

    if failures:
        print("failures: " + ", ".join(failures), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
