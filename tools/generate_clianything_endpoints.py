#!/usr/bin/env python3
"""Generate CLIAnything endpoint parity manifest from generated management MDX."""

from __future__ import annotations

import json
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DOCS_ROOT = ROOT / "docs" / "atius-router-docs" / "content" / "docs" / "en" / "api" / "management"
OUTPUT = ROOT / "tools" / "clianything_endpoints.json"

MUTATING_METHODS = {"POST", "PUT", "PATCH", "DELETE"}


def normalize_resource(group: str, path: str) -> str:
    if group == "channel-management":
        return "channels"
    if group == "model-management":
        return "models"
    if group == "token-management":
        return "tokens"
    if group == "user-management":
        return "users"
    if group == "groups":
        return "prefill-groups" if "prefill_group" in path else "groups"
    if group == "redemption":
        return "redemptions"
    if group == "vendors":
        return "vendors"
    if group == "logs":
        return "logs"
    return group.replace("_", "-")


def is_crud_path(group: str, method: str, path: str) -> bool:
    base_patterns = {
        "channel-management": r"^/api/channel/?(?:\{id\})?$",
        "model-management": r"^/api/models/?(?:\{id\})?$",
        "token-management": r"^/api/token/?(?:\{id\})?$",
        "user-management": r"^/api/user/?(?:\{id\})?$",
        "groups": r"^/api/(?:prefill_group|group)/?(?:\{id\})?$",
        "redemption": r"^/api/redemption/?(?:\{id\})?$",
        "vendors": r"^/api/vendors/?(?:\{id\})?$",
    }
    if group in base_patterns and re.fullmatch(base_patterns[group], path):
        return True
    if group in {"channel-management", "model-management", "token-management", "user-management", "redemption", "vendors"}:
        return method == "GET" and path.endswith("/search")
    return False


def classify(group: str, method: str, path: str) -> str:
    lower = path.lower()
    if "webhook" in lower or lower.endswith("/notify"):
        return "external-webhook"
    if group in {"oauth", "two-factor-auth", "user-auth", "security-verification"}:
        return "auth-flow"
    if any(marker in lower for marker in ["passkey", "/2fa", "login", "logout", "register", "reset_password", "verification", "verify"]):
        return "auth-flow"
    if group == "payment":
        return "read-only" if method == "GET" and ("topup-info" in lower or "topup/self" in lower) else "api-action"
    if group == "system":
        return "api-action" if method in MUTATING_METHODS else "read-only"
    if group == "system-settings":
        return "read-only" if method == "GET" else "api-action"
    if group in {"statistics", "tasks"}:
        return "read-only"
    if group == "logs":
        return "api-action" if method in MUTATING_METHODS else "read-only"
    if group == "channel-management":
        if is_crud_path(group, method, path) or path in {"/api/channel/models", "/api/channel/models_enabled", "/api/channel/tag/models"}:
            return "read-only" if method == "GET" and path.endswith(("models", "models_enabled", "tag/models")) else "db-crud"
        return "api-action"
    if group == "model-management":
        if path.endswith("/missing"):
            return "read-only"
        if "sync_upstream" in path:
            return "read-only" if path.endswith("/preview") else "api-action"
        return "db-crud" if is_crud_path(group, method, path) else "api-action"
    if group == "token-management":
        if path == "/api/usage/token/":
            return "read-only"
        if path.endswith("/search"):
            return "read-only"
        return "db-crud" if is_crud_path(group, method, path) else "api-action"
    if group == "user-management":
        if path.endswith(("/self", "/self/groups", "/models", "/token", "/aff", "/topup")) and method == "GET":
            return "read-only"
        return "db-crud" if is_crud_path(group, method, path) else "api-action"
    if group in {"groups", "redemption", "vendors"}:
        if path.endswith("/search"):
            return "read-only"
        if path.endswith("/invalid"):
            return "api-action"
        return "db-crud" if is_crud_path(group, method, path) else "api-action"
    if group == "default":
        return "api-action"
    return "read-only" if method == "GET" else "api-action"


def crud_command(group: str, method: str, path: str) -> str:
    resource = normalize_resource(group, path)
    if method == "GET" and "{id}" in path:
        return f"clianything get {resource} --id ID"
    if method == "GET":
        return f"clianything list {resource}"
    if method == "POST":
        return f"clianything create {resource} --set coluna=valor --execute"
    if method == "PUT":
        return f"clianything update {resource} --id ID --set coluna=valor --execute"
    if method == "DELETE":
        return f"clianything delete {resource} --id ID --execute"
    return f"clianything endpoint invoke {method} {path}"


def cli_command(group: str, method: str, path: str, classification: str) -> str:
    specific = {
        ("GET", "/api/channel/test/{id}"): "clianything channel test --id ID --execute",
        ("GET", "/api/channel/fetch_models/{id}"): "clianything channel fetch-models --id ID --execute",
        ("GET", "/api/channel/update_balance/{id}"): "clianything channel balance --id ID --execute",
        ("GET", "/api/channel/update_balance"): "clianything channel balance --execute",
        ("POST", "/api/channel/copy/{id}"): "clianything channel copy --id ID --execute",
        ("GET", "/api/channel/models"): "clianything channel models",
        ("GET", "/api/channel/models_enabled"): "clianything channel models-enabled",
        ("GET", "/api/channel/search"): "clianything channel search TEXT",
        ("GET", "/api/models/missing"): "clianything model missing",
        ("GET", "/api/models/sync_upstream/preview"): "clianything model sync-upstream --preview",
        ("POST", "/api/models/sync_upstream"): "clianything model sync-upstream --execute",
        ("GET", "/api/option/"): "clianything option get [KEY]",
        ("PUT", "/api/option/"): "clianything option set KEY VALUE --execute",
        ("GET", "/api/ratio_sync/channels"): "clianything ratio channels",
        ("POST", "/api/ratio_sync/fetch"): "clianything ratio fetch --execute",
        ("POST", "/api/option/rest_model_ratio"): "clianything ratio reset --execute",
        ("GET", "/api/usage/token/"): "clianything token usage",
        ("GET", "/api/log/stat"): "clianything log stat",
        ("DELETE", "/api/log/"): "clianything log delete --confirm-delete-all --execute",
        ("GET", "/api/task/"): "clianything task list",
        ("GET", "/api/task/self"): "clianything task self",
        ("GET", "/api/mj/"): "clianything task mj",
        ("GET", "/api/mj/self"): "clianything task mj-self",
        ("GET", "/api/vendors/search"): "clianything vendor search TEXT",
    }
    if (method, path) in specific:
        return specific[(method, path)]
    if classification == "db-crud":
        return crud_command(group, method, path)
    return f"clianything endpoint invoke {method} {path}"


def endpoint_name(doc: str, method: str, path: str) -> str:
    stem = Path(doc).stem.replace("_", "-")
    return f"{stem}:{method.lower()}"


def main() -> int:
    entries: list[dict[str, object]] = []
    for file_path in sorted(DOCS_ROOT.rglob("*.mdx")):
        rel = file_path.relative_to(DOCS_ROOT).as_posix()
        group = rel.split("/", 1)[0] if "/" in rel else rel.replace(".mdx", "")
        text = file_path.read_text(encoding="utf-8")
        matches = re.findall(r'\{"path":"([^"]+)","method":"([^"]+)"\}', text)
        for path, raw_method in matches:
            method = raw_method.upper()
            classification = classify(group, method, path)
            command = cli_command(group, method, path, classification)
            requires_auth = "Requires" in text and "Public" not in text
            entries.append(
                {
                    "group": group,
                    "name": endpoint_name(rel, method, path),
                    "method": method,
                    "path": path,
                    "doc": rel,
                    "classification": classification,
                    "cli_command": command,
                    "safe_default": bool(method == "GET" and classification in {"read-only", "db-crud"}),
                    "requires_auth": bool(requires_auth),
                    "notes": "Generated from management MDX. Use endpoint invoke when no dedicated shortcut exists.",
                }
            )
    entries.sort(key=lambda item: (str(item["group"]), str(item["path"]), str(item["method"])))
    OUTPUT.write_text(json.dumps(entries, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {OUTPUT} ({len(entries)} endpoints)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
