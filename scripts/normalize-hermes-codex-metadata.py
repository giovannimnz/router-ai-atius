#!/usr/bin/env python3
"""Normalize Codex model context metadata in a Hermes config without reformatting YAML."""

from __future__ import annotations

import argparse
import json
import re
import shutil
from datetime import datetime, timezone
from pathlib import Path


MODEL_CONTEXT_WINDOWS = {
    "gpt-5.6-sol": 272_000,
    "gpt-5.6-terra": 272_000,
    "gpt-5.6-luna": 272_000,
    "gpt-5.5": 272_000,
    "gpt-5.3-codex-spark": 128_000,
}
RETIRED_CODEX_MODELS = {"gpt-5.4", "gpt-5.4-mini"}

MODEL_KEY_PATTERN = re.compile(
    r"^(?P<indent>\s*)(?P<model>gpt-5\.6-(?:sol|terra|luna)|gpt-5\.5|gpt-5\.3-codex-spark):\s*(?:#.*)?$"
)
CONTEXT_PATTERN = re.compile(r"^(?P<indent>\s*)context_length:\s*\d+\s*(?P<comment>#.*)?$")
DEFAULT_PATTERN = re.compile(r"^(?P<indent>\s*)default:\s*(?P<model>[^\s#]+)(?P<suffix>\s*(?:#.*)?)$")


def _indent_width(line: str) -> int:
    return len(line) - len(line.lstrip(" "))


def _block_end(lines: list[str], start: int, indent: int) -> int:
    for index in range(start + 1, len(lines)):
        stripped = lines[index].strip()
        if stripped and not stripped.startswith("#") and _indent_width(lines[index]) <= indent:
            return index
    return len(lines)


def _replace_context_line(line: str, context_length: int) -> str:
    match = CONTEXT_PATTERN.match(line)
    if not match:
        return line
    comment = match.group("comment") or ""
    separator = " " if comment else ""
    return f"{match.group('indent')}context_length: {context_length}{separator}{comment}"


def normalize_config(text: str, replacement_default: str | None = None) -> tuple[str, list[str]]:
    had_trailing_newline = text.endswith("\n")
    lines = text.splitlines()
    changes: list[str] = []

    model_block_start = next((i for i, line in enumerate(lines) if line.strip() == "model:" and _indent_width(line) == 0), None)
    if model_block_start is not None:
        model_block_end = _block_end(lines, model_block_start, 0)
        default_index = None
        default_model = None
        context_index = None
        for index in range(model_block_start + 1, model_block_end):
            default_match = DEFAULT_PATTERN.match(lines[index])
            if default_match and _indent_width(lines[index]) > 0 and default_index is None:
                default_index = index
                default_model = default_match.group("model")
            if CONTEXT_PATTERN.match(lines[index]) and _indent_width(lines[index]) > 0 and context_index is None:
                context_index = index

        if default_index is not None and default_model in RETIRED_CODEX_MODELS and replacement_default:
            match = DEFAULT_PATTERN.match(lines[default_index])
            assert match is not None
            lines[default_index] = f"{match.group('indent')}default: {replacement_default}{match.group('suffix')}"
            changes.append(f"default:{default_model}->{replacement_default}")
            default_model = replacement_default

        expected_context = MODEL_CONTEXT_WINDOWS.get(default_model or "")
        if expected_context is not None:
            if context_index is None:
                insert_at = (default_index + 1) if default_index is not None else model_block_start + 1
                lines.insert(insert_at, f"  context_length: {expected_context}")
                changes.append(f"model.context_length:missing->{expected_context}")
            else:
                updated = _replace_context_line(lines[context_index], expected_context)
                if updated != lines[context_index]:
                    old_value = lines[context_index].split(":", 1)[1].strip().split()[0]
                    lines[context_index] = updated
                    changes.append(f"model.context_length:{old_value}->{expected_context}")

    index = 0
    while index < len(lines):
        match = MODEL_KEY_PATTERN.match(lines[index])
        if not match:
            index += 1
            continue
        model_name = match.group("model")
        expected_context = MODEL_CONTEXT_WINDOWS[model_name]
        indent = _indent_width(lines[index])
        end = _block_end(lines, index, indent)
        context_index = next(
            (candidate for candidate in range(index + 1, end) if CONTEXT_PATTERN.match(lines[candidate])),
            None,
        )
        if context_index is None:
            lines.insert(index + 1, f"{' ' * (indent + 2)}context_length: {expected_context}")
            changes.append(f"{model_name}.context_length:missing->{expected_context}")
            index += 2
            continue
        updated = _replace_context_line(lines[context_index], expected_context)
        if updated != lines[context_index]:
            old_value = lines[context_index].split(":", 1)[1].strip().split()[0]
            lines[context_index] = updated
            changes.append(f"{model_name}.context_length:{old_value}->{expected_context}")
        index = end

    normalized = "\n".join(lines)
    if had_trailing_newline:
        normalized += "\n"
    return normalized, changes


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", type=Path, default=Path.home() / ".hermes" / "config.yaml")
    parser.add_argument("--default", choices=sorted(MODEL_CONTEXT_WINDOWS))
    parser.add_argument("--write", action="store_true")
    args = parser.parse_args()

    original = args.config.read_text(encoding="utf-8")
    normalized, changes = normalize_config(original, replacement_default=args.default)
    result: dict[str, object] = {
        "config": str(args.config),
        "changed": bool(changes),
        "changes": changes,
        "written": False,
    }

    if changes and args.write:
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        backup_dir = args.config.parent / "backups" / "codex-model-metadata"
        backup_dir.mkdir(parents=True, exist_ok=True)
        backup_path = backup_dir / f"{timestamp}-{args.config.name}"
        shutil.copy2(args.config, backup_path)
        args.config.write_text(normalized, encoding="utf-8")
        result["written"] = True
        result["backup"] = str(backup_path)

    print(json.dumps(result, ensure_ascii=True, sort_keys=True))
    return 1 if changes and not args.write else 0


if __name__ == "__main__":
    raise SystemExit(main())
