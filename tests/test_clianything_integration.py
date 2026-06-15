#!/usr/bin/env python3
"""Subprocess smoke tests for bin/clianything.

The whole class skips unless Podman and the Postgres container are available.
All normal tests are read-only or dry-run.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
CLI = REPO_ROOT / "bin" / "clianything"
DB_CONTAINER = os.environ.get("CLIANYTHING_DB_CONTAINER", "postgres")
DB_NAME = os.environ.get("CLIANYTHING_DB_NAME", "DBRouterAiAtius")
DB_USER = os.environ.get("CLIANYTHING_DB_USER", "admin")
TIMEOUT = float(os.environ.get("CLIANYTHING_TEST_TIMEOUT", "20"))


def safe_env() -> dict[str, str]:
    env = os.environ.copy()
    for name in [
        "ATIUS_ROUTER_ADMIN_TOKEN",
        "ATIUS_ROUTER_TOKEN",
        "OPENAI_API_KEY",
        "ANTHROPIC_API_KEY",
    ]:
        env.pop(name, None)
    env["PYTHONUNBUFFERED"] = "1"
    return env


class CLIAnythingIntegrationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not CLI.exists():
            raise unittest.SkipTest(f"{CLI} not found")
        if shutil.which("podman") is None:
            raise unittest.SkipTest("podman not available")

        probe = subprocess.run(
            [
                "podman",
                "exec",
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
                "select 1",
            ],
            cwd=REPO_ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=TIMEOUT,
            env=safe_env(),
            check=False,
        )
        if probe.returncode != 0 or probe.stdout.strip() != "1":
            detail = (probe.stderr or probe.stdout).strip().splitlines()[:1]
            suffix = f": {detail[0]}" if detail else ""
            raise unittest.SkipTest(f"postgres container unavailable{suffix}")

    def run_cli(self, *args: str, timeout: float = TIMEOUT) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(CLI), *args],
            cwd=REPO_ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=timeout,
            env=safe_env(),
            check=False,
        )

    def assert_success(self, proc: subprocess.CompletedProcess[str]) -> None:
        if proc.returncode != 0:
            self.fail(
                "command failed with exit "
                f"{proc.returncode}\nSTDOUT:\n{proc.stdout[-1000:]}\nSTDERR:\n{proc.stderr[-1000:]}"
            )

    def query_json(self, sql: str):
        proc = self.run_cli("query", sql, "--format", "json")
        self.assert_success(proc)
        return json.loads(proc.stdout)

    def test_resources_lists_core_resources(self):
        proc = self.run_cli("resources")

        self.assert_success(proc)
        self.assertIn("channels", proc.stdout)
        self.assertIn("models", proc.stdout)
        self.assertIn("options", proc.stdout)
        self.assertNotRegex(proc.stdout, r"sk-[A-Za-z0-9]{10,}")

    def test_status_reports_pod_http_and_db_checks(self):
        proc = self.run_cli("status")

        self.assert_success(proc)
        self.assertIn("pod", proc.stdout)
        self.assertIn("containers", proc.stdout)
        self.assertIn("db", proc.stdout)
        self.assertRegex(proc.stdout, r"\b(ok|fail)\b")
        self.assertNotRegex(proc.stdout + proc.stderr, r"Bearer\s+[A-Za-z0-9._~+/=-]+")

    def test_status_strict_if_supported(self):
        help_proc = self.run_cli("status", "--help")
        if "--strict" not in help_proc.stdout:
            self.skipTest("status --strict not supported by this CLI build")

        proc = self.run_cli("status", "--strict")
        self.assertIn(proc.returncode, {0, 1, 2})
        combined = (proc.stdout + "\n" + proc.stderr).lower()
        self.assertRegex(combined, r"\b(ok|degraded|fail)\b")

    def test_providers_all_json_is_redacted(self):
        proc = self.run_cli("providers", "--all", "--format", "json")

        self.assert_success(proc)
        providers = json.loads(proc.stdout)
        self.assertIsInstance(providers, list)
        if providers:
            self.assertIn("id", providers[0])
            self.assertIn("name", providers[0])
            self.assertIn("type", providers[0])
        self.assertNotIn("key", proc.stdout.lower())
        self.assertNotRegex(proc.stdout, r"sk-[A-Za-z0-9]{10,}")

    def test_embeddings_json_lists_embedding_channels(self):
        proc = self.run_cli("embeddings", "--format", "json")

        self.assert_success(proc)
        rows = json.loads(proc.stdout)
        self.assertIsInstance(rows, list)
        self.assertGreaterEqual(len(rows), 1)
        joined = " ".join(json.dumps(row) for row in rows)
        self.assertIn("Embeddings", joined)
        self.assertNotRegex(joined, r"sk-[A-Za-z0-9]{10,}")

    def test_query_count_channels_json(self):
        rows = self.query_json("select count(*) as channels from channels")

        self.assertIsInstance(rows, list)
        self.assertEqual(len(rows), 1)
        self.assertIn("channels", rows[0])
        self.assertGreaterEqual(int(rows[0]["channels"]), 0)

    def test_update_channels_is_dry_run_and_does_not_change_row(self):
        before = self.query_json("select id, priority from channels where id = 1")

        proc = self.run_cli("update", "channels", "--id", "1", "--set", "priority=0")

        self.assert_success(proc)
        self.assertIn("DRY-RUN", proc.stdout)
        self.assertIn("nada foi alterado", proc.stdout)
        self.assertNotIn("--execute", " ".join(proc.args))
        self.assertNotIn("Backup antes da escrita", proc.stdout)

        after = self.query_json("select id, priority from channels where id = 1")
        self.assertEqual(after, before)

    def test_backup_channels_opt_in(self):
        if os.environ.get("CLIANYTHING_RUN_BACKUP_TEST") != "1":
            self.skipTest("set CLIANYTHING_RUN_BACKUP_TEST=1 to run backup drill")

        before = set((REPO_ROOT / "backups" / "clianything").glob("*_channels.sql"))
        proc = self.run_cli("backup", "channels", timeout=max(TIMEOUT, 45))
        self.assert_success(proc)
        backup_path = Path(proc.stdout.strip())
        self.assertTrue(backup_path.exists())
        self.assertEqual(backup_path.parent, REPO_ROOT / "backups" / "clianything")
        self.assertGreater(backup_path.stat().st_size, 0)
        self.assertRegex(backup_path.read_text(encoding="utf-8", errors="replace"), r"INSERT INTO public\.channels")
        self.assertIn(backup_path, set((REPO_ROOT / "backups" / "clianything").glob("*_channels.sql")) - before)

    def test_smoke_embeddings_without_token_exits_two(self):
        proc = subprocess.run(
            [sys.executable, str(REPO_ROOT / "scripts" / "smoke-embeddings.py")],
            cwd=REPO_ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=TIMEOUT,
            env=safe_env(),
            check=False,
        )
        self.assertEqual(proc.returncode, 2)
        self.assertIn("Missing ATIUS_ROUTER_TOKEN", proc.stderr)

    def test_smoke_routing_matrix_without_token_exits_two(self):
        proc = subprocess.run(
            [sys.executable, str(REPO_ROOT / "scripts" / "smoke-routing-matrix.py")],
            cwd=REPO_ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=TIMEOUT,
            env=safe_env(),
            check=False,
        )
        self.assertEqual(proc.returncode, 2)
        self.assertIn("Missing ATIUS_ROUTER_TOKEN", proc.stderr)


if __name__ == "__main__":
    unittest.main()
