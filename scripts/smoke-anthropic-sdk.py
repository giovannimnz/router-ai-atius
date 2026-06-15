#!/usr/bin/env python3
"""Minimal Anthropic SDK smoke test for the local Atius router."""

from __future__ import annotations

import os
import json
import subprocess
import sys
import time
from pathlib import Path
from typing import Iterable, List, Optional


DEFAULT_BASE_URL = "http://127.0.0.1:3000"
DEFAULT_MODEL = "MiniMax-M3"
MAX_OUTPUT_CHARS = 120
REPO_ROOT = Path(__file__).resolve().parents[1]
USER_AGENT = os.environ.get("ATIUS_ROUTER_USER_AGENT", "Mozilla/5.0 AtiusRouterSmoke/1.0")


def _env(name: str, default: Optional[str] = None) -> Optional[str]:
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
    return _short_text(scrubbed)


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


def _message_text(message: object) -> str:
    parts: List[str] = []
    for block in getattr(message, "content", []) or []:
        text = getattr(block, "text", None)
        if text is None and isinstance(block, dict):
            text = block.get("text")
        if text:
            parts.append(str(text))
    return " ".join(parts)


def main() -> int:
    token = _env("ATIUS_ROUTER_TOKEN")
    if token is None:
        print(
            "Missing ATIUS_ROUTER_TOKEN; export it to run the Anthropic SDK smoke test.",
            file=sys.stderr,
        )
        return 2

    try:
        from anthropic import Anthropic
    except ImportError:
        print(
            "Missing dependency: install the Anthropic Python SDK, e.g. `python3 -m pip install anthropic`.",
            file=sys.stderr,
        )
        return 1

    base_url = _env("ATIUS_ROUTER_ANTHROPIC_BASE_URL", DEFAULT_BASE_URL)
    model = _env("ATIUS_ROUTER_MODEL", DEFAULT_MODEL)
    max_tokens = int(_env("ATIUS_ROUTER_MAX_TOKENS", "32") or "32")

    try:
        client = Anthropic(
            api_key=token,
            base_url=base_url,
            timeout=30.0,
            max_retries=0,
            default_headers={"User-Agent": USER_AGENT},
        )
        response = client.messages.create(
            model=model,
            max_tokens=max_tokens,
            messages=[{"role": "user", "content": "Reply with OK."}],
        )
        text = _message_text(response)
    except Exception as exc:  # noqa: BLE001 - smoke script should report concise SDK failures.
        error = _scrub("{}: {}".format(type(exc).__name__, exc), [token])
        print(f"anthropic error: {error}", file=sys.stderr)
        return 1

    expected_channel = _env("ATIUS_ROUTER_EXPECT_CHANNEL_NAME")
    if expected_channel:
        found, channel_names = _wait_for_channel_name(expected_channel)
        if not found:
            print(
                _scrub(
                    f"anthropic channel mismatch: expected {expected_channel}, got {channel_names or ['<unknown>']}",
                    [token],
                ),
                file=sys.stderr,
            )
            return 1

    print(f"anthropic ok: {_short_text(text)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
