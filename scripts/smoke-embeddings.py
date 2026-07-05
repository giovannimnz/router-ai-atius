#!/usr/bin/env python3
"""Routing smoke test for OpenAI-compatible embeddings."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import time
from pathlib import Path
from typing import Any, Iterable
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_BASE_URL = "http://127.0.0.1:3000/v1"
DEFAULT_MODEL = "embedding-gte-v1"
DEFAULT_EXPECTED_DIM = 768
USER_AGENT = os.environ.get("ATIUS_ROUTER_USER_AGENT", "Mozilla/5.0 AtiusRouterSmoke/1.0")
MAX_OUTPUT_CHARS = 180
ACCEPTABLE_UPSTREAM_CODES = {400, 402, 429}


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
    scrubbed = scrubbed.replace("GroupId", "<redacted-group-id>")
    scrubbed = scrubbed.replace("group_id", "<redacted-group-id>")
    scrubbed = scrubbed.replace("Authorization", "<redacted-auth>")
    scrubbed = scrubbed.replace("x-api-key", "<redacted-auth>")
    return _short_text(scrubbed)


def _providers_have_channel(channel_name: str) -> bool:
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
        return False
    try:
        rows = json.loads(proc.stdout)
    except json.JSONDecodeError:
        return False
    return any(str(row.get("name", "")) == channel_name and int(row.get("status") or 0) == 1 for row in rows)


def _latest_channel_names() -> list[str]:
    cli = REPO_ROOT / "bin" / "clianything"
    proc = subprocess.run(
        [
            str(cli),
            "query",
            (
                "select coalesce(nullif(l.channel_name, ''), c.name) as channel_name, "
                "l.model_name, l.id from logs l left join channels c on c.id = l.channel_id "
                "order by l.id desc limit 40"
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


def _wait_for_channel_name(expected_channel: str) -> tuple[bool, list[str]]:
    names: list[str] = []
    for _ in range(12):
        names = _latest_channel_names()
        if expected_channel in names:
            return True, names
        time.sleep(0.5)
    return False, names


def build_embedding_payload(
    *,
    model: str,
    input_text: str,
    input_items: list[str] | None = None,
    embedding_type: str | None = None,
    openai_dimensions: int | None = None,
) -> dict[str, Any]:
    payload_input: Any = input_items if input_items is not None else input_text
    payload: dict[str, Any] = {"model": model, "input": payload_input}
    if model == "embo-01":
        payload["type"] = embedding_type if embedding_type in {"query", "db"} else "query"
    if openai_dimensions is not None:
        payload["dimensions"] = openai_dimensions
    return payload


def assert_embedding_vector_shape(embedding: object, expected_dim: int | None, model: str) -> int:
    if not isinstance(embedding, list) or not embedding:
        raise ValueError(f"{model}: embedding must be a non-empty list")
    if not all(isinstance(item, (int, float)) for item in embedding):
        raise ValueError(f"{model}: embedding must contain only numeric values")
    dimension = len(embedding)
    if expected_dim is not None and dimension != expected_dim:
        raise ValueError(f"{model}: expected dimension {expected_dim}, got {dimension}")
    return dimension


def expected_embedding_dimension(model: str, expected_dim_raw: str | None = None) -> int | None:
    if expected_dim_raw == "skip":
        return None
    if expected_dim_raw:
        return int(expected_dim_raw)
    if model in {DEFAULT_MODEL, "embo-01"}:
        return DEFAULT_EXPECTED_DIM
    if model == "text-embedding-3-large":
        return 3072
    return 1536


def _request_embeddings(base_url: str, token: str, payload: dict[str, Any]) -> tuple[int, dict[str, Any] | None, str]:
    url = base_url.rstrip("/") + "/embeddings"
    body = json.dumps(payload).encode("utf-8")
    request = Request(
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
    try:
        with urlopen(request, timeout=30.0) as response:
            text = response.read().decode("utf-8", errors="replace")
            try:
                return response.status, json.loads(text), text
            except json.JSONDecodeError:
                return response.status, None, text
    except HTTPError as exc:
        text = exc.read().decode("utf-8", errors="replace")
        try:
            return exc.code, json.loads(text), text
        except json.JSONDecodeError:
            return exc.code, None, text
    except URLError as exc:
        raise RuntimeError(str(exc)) from exc


def main() -> int:
    token = _env("ATIUS_ROUTER_TOKEN")
    if token is None:
        print(
            "Missing ATIUS_ROUTER_TOKEN; export it to run the embeddings smoke test.",
            file=sys.stderr,
        )
        return 2

    base_url = _env("ATIUS_ROUTER_EMBEDDINGS_BASE_URL", DEFAULT_BASE_URL) or DEFAULT_BASE_URL
    model = _env("ATIUS_ROUTER_EMBEDDINGS_MODEL", DEFAULT_MODEL) or DEFAULT_MODEL
    embedding_type = _env("ATIUS_ROUTER_EMBEDDING_TYPE", "query") or "query"
    input_mode = (_env("ATIUS_ROUTER_EMBEDDINGS_INPUT_MODE", "single") or "single").lower()
    expected_dim_raw = _env("ATIUS_ROUTER_EXPECT_EMBEDDING_DIM")
    openai_dimensions_raw = _env("ATIUS_ROUTER_OPENAI_EMBEDDING_DIMENSIONS")
    expected_channel_name = _env("ATIUS_ROUTER_EXPECT_CHANNEL_NAME")
    accept_upstream_error = _env("ATIUS_ROUTER_ACCEPT_UPSTREAM_ERROR") == "1"

    if expected_channel_name and model.startswith("text-embedding-3-") and not _providers_have_channel(expected_channel_name):
        print(
            f"embeddings skipped: expected channel {expected_channel_name} not present",
            file=sys.stderr,
        )
        return 2

    expected_dim = expected_embedding_dimension(model, expected_dim_raw)

    if input_mode == "array":
        input_items = ["hello", "world"]
        input_text = input_items[0]
    else:
        input_mode = "single"
        input_items = None
        input_text = "hello"

    openai_dimensions = int(openai_dimensions_raw) if openai_dimensions_raw else None
    payload = build_embedding_payload(
        model=model,
        input_text=input_text,
        input_items=input_items,
        embedding_type=embedding_type,
        openai_dimensions=openai_dimensions,
    )

    try:
        code, data, raw_text = _request_embeddings(base_url, token, payload)
    except Exception as exc:  # noqa: BLE001 - smoke script should report concise routing failures.
        print(f"embeddings upstream: {_scrub(type(exc).__name__ + ': ' + str(exc), [token])}", file=sys.stderr)
        return 1

    if code != 200:
        detail = _scrub(raw_text, [token, expected_channel_name or ""])
        print(f"embeddings upstream: HTTP {code} {detail}", file=sys.stderr)
        if code in ACCEPTABLE_UPSTREAM_CODES:
            return 0 if accept_upstream_error else 1
        return 1

    if not isinstance(data, dict):
        print("embeddings error: response body is not JSON", file=sys.stderr)
        return 1

    embedding_rows = data.get("data", [])
    if not isinstance(embedding_rows, list) or not embedding_rows:
        print("embeddings error: missing data[0]", file=sys.stderr)
        return 1
    if input_items is not None and len(embedding_rows) != len(input_items):
        print(f"embeddings error: expected {len(input_items)} rows, got {len(embedding_rows)}", file=sys.stderr)
        return 1

    dimensions: list[int] = []
    for index, row in enumerate(embedding_rows):
        if not isinstance(row, dict):
            print(f"embeddings error: malformed data[{index}]", file=sys.stderr)
            return 1
        try:
            dimensions.append(assert_embedding_vector_shape(row.get("embedding"), expected_dim, model))
        except ValueError as exc:
            print(f"embeddings error: {_scrub(str(exc), [token])}", file=sys.stderr)
            return 1

    if expected_channel_name:
        found, channel_names = _wait_for_channel_name(expected_channel_name)
        if not found:
            print(
                _scrub(
                    f"embeddings channel mismatch: expected {expected_channel_name}, got {channel_names or ['<unknown>']}",
                    [token],
                ),
                file=sys.stderr,
            )
            return 1

    display_type = embedding_type if model == "embo-01" else "openai"
    print(
        f"embeddings ok: model={model} type={display_type} dimension={dimensions[0]} rows={len(embedding_rows)} mode={input_mode}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
