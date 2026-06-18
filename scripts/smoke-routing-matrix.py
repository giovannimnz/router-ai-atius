#!/usr/bin/env python3
"""Provider-family routing matrix for the Atius router."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Any, Iterable
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


REPO_ROOT = Path(__file__).resolve().parents[1]
TOKEN_ENV = "ATIUS_ROUTER_TOKEN"
BASE_URL = os.environ.get("ATIUS_ROUTER_OPENAI_BASE_URL", "http://127.0.0.1:3000/v1").strip()
USER_AGENT = os.environ.get("ATIUS_ROUTER_USER_AGENT", "Mozilla/5.0 AtiusRouterSmoke/1.0")
MAX_OUTPUT_CHARS = 180


def _env(name: str, default: str | None = None) -> str | None:
    value = os.environ.get(name)
    if value is None or not value.strip():
        return default
    return value.strip()


def _short_text(value: object) -> str:
    text = str(value or "").replace("\n", " ").strip()
    return text[:MAX_OUTPUT_CHARS] if text else "<empty>"


def _scrub(message: str, secrets: Iterable[str]) -> str:
    scrubbed = message
    for secret in secrets:
        if secret:
            scrubbed = scrubbed.replace(secret, "<redacted>")
    scrubbed = scrubbed.replace("Authorization", "<redacted-auth>")
    scrubbed = scrubbed.replace("x-api-key", "<redacted-auth>")
    scrubbed = scrubbed.replace("GroupId", "<redacted-group-id>")
    return _short_text(scrubbed)


def _provider_names() -> dict[str, int]:
    cli = REPO_ROOT / "bin" / "clianything"
    proc = subprocess.run(
        [str(cli), "providers", "--all", "--format", "json"],
        cwd=REPO_ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if proc.returncode != 0:
        return {}
    try:
        rows = json.loads(proc.stdout)
    except json.JSONDecodeError:
        return {}
    return {str(row.get("name", "")): int(row.get("status") or 0) for row in rows if row.get("name")}


def _recent_channel_names() -> list[str]:
    cli = REPO_ROOT / "bin" / "clianything"
    proc = subprocess.run(
        [
            str(cli),
            "query",
            (
                "select coalesce(nullif(l.channel_name, ''), c.name) as channel_name, "
                "l.model_name, l.id from logs l left join channels c on c.id = l.channel_id "
                "order by l.id desc limit 12"
            ),
            "--format",
            "json",
        ],
        cwd=REPO_ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if proc.returncode != 0:
        return []
    try:
        rows = json.loads(proc.stdout)
    except json.JSONDecodeError:
        return []
    names = []
    for row in rows:
        channel_name = row.get("channel_name")
        if channel_name:
            names.append(str(channel_name))
    return names


def _http_json(path: str, token: str) -> tuple[int, Any, str]:
    url = BASE_URL.rstrip("/") + path
    request = Request(
        url,
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/json",
            "User-Agent": USER_AGENT,
        },
        method="GET",
    )
    try:
        with urlopen(request, timeout=30.0) as response:
            text = response.read().decode("utf-8", errors="replace")
            try:
                return response.status, json.loads(text), text
            except json.JSONDecodeError:
                return response.status, text, text
    except HTTPError as exc:
        text = exc.read().decode("utf-8", errors="replace")
        try:
            return exc.code, json.loads(text), text
        except json.JSONDecodeError:
            return exc.code, text, text
    except URLError as exc:
        return 0, None, str(exc)


def _run_script(script: str, env: dict[str, str]) -> tuple[int, str, str]:
    proc = subprocess.run(
        [sys.executable, str(REPO_ROOT / "scripts" / script)],
        cwd=REPO_ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
        check=False,
    )
    return proc.returncode, proc.stdout, proc.stderr


def _classify_proc(code: int, stdout: str, stderr: str) -> tuple[str, str]:
    combined = f"{stdout}\n{stderr}"
    if code == 0:
        return "ok", _scrub(combined, [])
    if code == 2:
        return "skipped", _scrub(combined, [])
    if any(marker in combined for marker in ["402", "429", "529", "Stream must be set to true"]):
        return "upstream", _scrub(combined, [])
    return "fail", _scrub(combined, [])


def _print_table(rows: list[dict[str, Any]]) -> None:
    if not rows:
        print("[]")
        return
    columns = ["test", "model", "endpoint", "embedding", "expected_channel", "status", "detail"]
    widths = {column: len(column) for column in columns}
    for row in rows:
        for column in columns:
            widths[column] = min(max(widths[column], len(_short_text(row.get(column)))), 80)
    print("  ".join(column.ljust(widths[column]) for column in columns))
    print("  ".join("-" * widths[column] for column in columns))
    for row in rows:
        print("  ".join(_short_text(row.get(column)).ljust(widths[column]) for column in columns))


def main() -> int:
    token = _env(TOKEN_ENV)
    if token is None:
        print("Missing ATIUS_ROUTER_TOKEN; export it to run the routing matrix.", file=sys.stderr)
        return 2

    providers = _provider_names()
    recent_channels = _recent_channel_names()
    rows: list[dict[str, Any]] = []
    failure = False

    def add_row(**kwargs: Any) -> None:
        rows.append(kwargs)

    code, payload, raw = _http_json("/models", token)
    if code == 200 and isinstance(payload, dict):
        ids = {str(item.get("id", "")) for item in payload.get("data", []) if isinstance(item, dict)}
        add_row(
            test="models-openai",
            model="catalog",
            endpoint="/v1/models",
            embedding="n/a",
            expected_channel="n/a",
            status="ok" if {"MiniMax-M3", "deepseek-v4-flash", "embo-01"} & ids else "fail",
            detail=_scrub(f"ids={sorted(list(ids))[:5]}", [token]),
        )
    else:
        add_row(
            test="models-openai",
            model="catalog",
            endpoint="/v1/models",
            embedding="n/a",
            expected_channel="n/a",
            status="fail",
            detail=_scrub(f"HTTP {code} {raw}", [token]),
        )
        failure = True

    code, payload, raw = _http_json("/models?api_format=anthropic", token)
    if code == 200 and isinstance(payload, dict):
        ids = {str(item.get("id", "")) for item in payload.get("data", []) if isinstance(item, dict)}
        add_row(
            test="models-anthropic",
            model="catalog",
            endpoint="/v1/models?api_format=anthropic",
            embedding="n/a",
            expected_channel="Anthropic-Compatible",
            status="ok" if {"MiniMax-M3", "deepseek-v4-flash", "deepseek-v4-pro"} & ids else "fail",
            detail=_scrub(f"ids={sorted(list(ids))[:5]}", [token]),
        )
    else:
        add_row(
            test="models-anthropic",
            model="catalog",
            endpoint="/v1/models?api_format=anthropic",
            embedding="n/a",
            expected_channel="Anthropic-Compatible",
            status="fail",
            detail=_scrub(f"HTTP {code} {raw}", [token]),
        )
        failure = True

    cases = [
        {
            "test": "openai-mini-max",
            "script": "smoke-openai-sdk.py",
            "env": {"ATIUS_ROUTER_MODEL": "MiniMax-M3", "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "MiniMax - OpenAI-Compatible"},
            "expected_channel": "MiniMax - OpenAI-Compatible",
            "model": "MiniMax-M3",
            "endpoint": "/v1/chat/completions",
            "embedding": "n/a",
        },
        {
            "test": "openai-deepseek",
            "script": "smoke-openai-sdk.py",
            "env": {"ATIUS_ROUTER_MODEL": "deepseek-v4-flash", "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "DeepSeek - OpenAI-Compatible"},
            "expected_channel": "DeepSeek - OpenAI-Compatible",
            "model": "deepseek-v4-flash",
            "endpoint": "/v1/chat/completions",
            "embedding": "n/a",
        },
        {
            "test": "anthropic-mini-max",
            "script": "smoke-anthropic-sdk.py",
            "env": {"ATIUS_ROUTER_MODEL": "MiniMax-M3", "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "MiniMax - Anthropic-Compatible"},
            "expected_channel": "MiniMax - Anthropic-Compatible",
            "model": "MiniMax-M3",
            "endpoint": "/v1/messages",
            "embedding": "n/a",
        },
        {
            "test": "anthropic-deepseek",
            "script": "smoke-anthropic-sdk.py",
            "env": {"ATIUS_ROUTER_MODEL": "deepseek-v4-flash", "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "DeepSeek - Anthropic-Compatible"},
            "expected_channel": "DeepSeek - Anthropic-Compatible",
            "model": "deepseek-v4-flash",
            "endpoint": "/v1/messages",
            "embedding": "n/a",
        },
        {
            "test": "codex-oauth",
            "script": "smoke-openai-sdk.py",
            "env": {"ATIUS_ROUTER_MODEL": "gpt-5.5", "ATIUS_ROUTER_STREAM": "1", "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "OpenAI Codex OAuth"},
            "expected_channel": "OpenAI Codex OAuth",
            "model": "gpt-5.5",
            "endpoint": "/v1/chat/completions",
            "embedding": "n/a",
        },
        {
            "test": "embeddings-query",
            "script": "smoke-embeddings.py",
            "env": {"ATIUS_ROUTER_EMBEDDINGS_MODEL": "embo-01", "ATIUS_ROUTER_EMBEDDING_TYPE": "query", "ATIUS_ROUTER_EXPECT_EMBEDDING_DIM": "1536", "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "MiniMax - Embeddings"},
            "expected_channel": "MiniMax - Embeddings",
            "model": "embo-01",
            "endpoint": "/v1/embeddings",
            "embedding": "query/1536",
        },
        {
            "test": "embeddings-db",
            "script": "smoke-embeddings.py",
            "env": {"ATIUS_ROUTER_EMBEDDINGS_MODEL": "embo-01", "ATIUS_ROUTER_EMBEDDING_TYPE": "db", "ATIUS_ROUTER_EXPECT_EMBEDDING_DIM": "1536", "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "MiniMax - Embeddings"},
            "expected_channel": "MiniMax - Embeddings",
            "model": "embo-01",
            "endpoint": "/v1/embeddings",
            "embedding": "db/1536",
        },
    ]

    for case in cases:
        env = os.environ.copy()
        env.update(case["env"])
        code, stdout, stderr = _run_script(case["script"], env)
        status, detail = _classify_proc(code, stdout, stderr)
        if status == "fail":
            failure = True
        add_row(
            test=case["test"],
            model=case["model"],
            endpoint=case["endpoint"],
            embedding=case["embedding"],
            expected_channel=case["expected_channel"],
            status=status,
            detail=detail,
        )

    deepseek_embeddings_present = providers.get("DeepSeek - Embeddings") == 1 or "DeepSeek - Embeddings" in recent_channels
    add_row(
        test="deepseek-embeddings-block",
        model="n/a",
        endpoint="/v1/embeddings",
        embedding="blocked",
        expected_channel="DeepSeek - Embeddings",
        status="blocked" if not deepseek_embeddings_present else "fail",
        detail="blocked" if not deepseek_embeddings_present else "unexpected DeepSeek embeddings channel detected",
    )
    if deepseek_embeddings_present:
        failure = True

    openai_api_key = _env("OPENAI_API_KEY")
    openai_embeddings_channel_present = providers.get("OpenAI - Embeddings") == 1
    if openai_api_key and openai_embeddings_channel_present:
        openai_case = {
            "test": "embeddings-openai",
            "script": "smoke-embeddings.py",
            "env": {
                "ATIUS_ROUTER_EMBEDDINGS_MODEL": "text-embedding-3-small",
                "ATIUS_ROUTER_OPENAI_EMBEDDING_DIMENSIONS": "1536",
                "ATIUS_ROUTER_EXPECT_CHANNEL_NAME": "OpenAI - Embeddings",
            },
            "expected_channel": "OpenAI - Embeddings",
            "model": "text-embedding-3-small",
            "endpoint": "/v1/embeddings",
            "embedding": "openai/1536",
        }
        env = os.environ.copy()
        env.update(openai_case["env"])
        code, stdout, stderr = _run_script(openai_case["script"], env)
        status, detail = _classify_proc(code, stdout, stderr)
        if status == "fail":
            failure = True
        add_row(
            test=openai_case["test"],
            model=openai_case["model"],
            endpoint=openai_case["endpoint"],
            embedding=openai_case["embedding"],
            expected_channel=openai_case["expected_channel"],
            status=status,
            detail=detail,
        )
    else:
        add_row(
            test="embeddings-openai",
            model="text-embedding-3-small",
            endpoint="/v1/embeddings",
            embedding="openai/conditional",
            expected_channel="OpenAI - Embeddings",
            status="skipped",
            detail="conditional OpenAI embeddings channel absent or OPENAI_API_KEY missing",
        )

    _print_table(rows)
    return 1 if failure else 0


if __name__ == "__main__":
    raise SystemExit(main())
