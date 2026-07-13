#!/usr/bin/env python3
"""Focused tests for bin/clianything backend selection and k3s execution."""

from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from importlib.machinery import SourceFileLoader
from pathlib import Path
from types import SimpleNamespace
from unittest import mock


REPO_ROOT = Path(__file__).resolve().parents[1]
LAUNCHER_PATH = REPO_ROOT / "bin" / "clianything"


def load_launcher():
    loader = SourceFileLoader("clianything_launcher_under_test", str(LAUNCHER_PATH))
    spec = importlib.util.spec_from_loader(loader.name, loader)
    if spec is None:
        raise RuntimeError(f"Could not load {LAUNCHER_PATH}")
    module = importlib.util.module_from_spec(spec)
    loader.exec_module(module)
    return module


launcher = load_launcher()


def ready_pod(name: str = "router-ai-atius-postgres-0") -> dict:
    return {
        "metadata": {
            "name": name,
            "namespace": "router-ai-atius",
            "labels": {"app.kubernetes.io/name": "router-ai-atius-postgres"},
            "ownerReferences": [
                {
                    "apiVersion": "apps/v1",
                    "kind": "StatefulSet",
                    "name": "router-ai-atius-postgres",
                    "controller": True,
                }
            ],
        },
        "spec": {
            "nodeName": "atius-srv-1",
            "containers": [{"name": "postgres"}],
        },
        "status": {
            "phase": "Running",
            "conditions": [{"type": "Ready", "status": "True"}],
            "containerStatuses": [{"name": "postgres", "ready": True}],
        },
    }


class BackendArgumentTests(unittest.TestCase):
    def test_extracts_backend_and_read_only_anywhere(self):
        backend, read_only, forwarded = launcher.parse_launcher_args(
            ["query", "--backend", "k3s", "--read-only", "select 1", "--format", "raw"]
        )

        self.assertEqual(backend, "k3s")
        self.assertTrue(read_only)
        self.assertEqual(forwarded, ["query", "select 1", "--format", "raw"])

    def test_rejects_duplicate_or_invalid_backend(self):
        with self.assertRaises(launcher.LauncherError):
            launcher.parse_launcher_args(["status", "--backend", "k3s", "--backend=podman"])
        with self.assertRaises(launcher.LauncherError):
            launcher.parse_launcher_args(["status", "--backend", "docker"])

    def test_requires_explicit_choice_when_podman_and_k3s_exist(self):
        with mock.patch.object(launcher, "podman_postgres_exists", return_value=True), mock.patch.object(
            launcher, "k3s_postgres_exists", return_value=True
        ), mock.patch.dict(os.environ, {}, clear=True):
            with self.assertRaisesRegex(launcher.LauncherError, "informe --backend"):
                launcher.choose_backend(None)

        self.assertEqual(launcher.choose_backend("podman"), "podman")
        self.assertEqual(launcher.choose_backend("k3s"), "k3s")

    def test_explicit_podman_backend_is_forwarded_to_existing_cli(self):
        captured = {}

        def load_cli():
            captured["backend"] = os.environ.get("CLIANYTHING_DB_BACKEND")

            def fake_main(argv):
                captured["argv"] = argv
                return 0

            return SimpleNamespace(main=fake_main)

        with mock.patch.object(launcher, "load_cli", side_effect=load_cli), mock.patch.dict(
            os.environ, {}, clear=True
        ):
            result = launcher.main(["resources", "--backend", "podman"])

        self.assertEqual(result, 0)
        self.assertEqual(captured, {"backend": "podman", "argv": ["resources"]})


class K3sPodResolutionTests(unittest.TestCase):
    def test_resolves_single_ready_postgres_pod_on_srv1(self):
        self.assertEqual(
            launcher.resolve_k3s_postgres_pod({"items": [ready_pod()]}),
            "router-ai-atius-postgres-0",
        )

    def test_zero_or_multiple_candidates_fail(self):
        for items in ([], [ready_pod("postgres-0"), ready_pod("postgres-1")]):
            with self.subTest(count=len(items)):
                with self.assertRaisesRegex(launcher.LauncherError, "exatamente um"):
                    launcher.resolve_k3s_postgres_pod({"items": items})

    def test_wrong_node_or_not_ready_fails(self):
        wrong_node = ready_pod()
        wrong_node["spec"]["nodeName"] = "atius-srv-2"
        not_ready = ready_pod()
        not_ready["status"]["conditions"][0]["status"] = "False"

        with self.assertRaisesRegex(launcher.LauncherError, "atius-srv-1"):
            launcher.resolve_k3s_postgres_pod({"items": [wrong_node]})
        with self.assertRaisesRegex(launcher.LauncherError, "nao esta Ready"):
            launcher.resolve_k3s_postgres_pod({"items": [not_ready]})


class K3sSubprocessTests(unittest.TestCase):
    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.fake_dir = Path(self.temp_dir.name)
        self.log_path = self.fake_dir / "sudo.log"
        fake_sudo = self.fake_dir / "sudo"
        pod_json = json.dumps({"items": [ready_pod()]})
        fake_sudo.write_text(
            "#!/usr/bin/env python3\n"
            "import json, os, sys\n"
            "args = sys.argv[1:]\n"
            "with open(os.environ['FAKE_SUDO_LOG'], 'a', encoding='utf-8') as fh:\n"
            "    fh.write(json.dumps(args) + '\\n')\n"
            "if 'get' in args and 'pods' in args:\n"
            f"    print({pod_json!r})\n"
            "elif 'exec' in args:\n"
            "    sql = args[-1]\n"
            "    command = ' '.join(args)\n"
            "    if 'default_transaction_read_only=on' not in command:\n"
            "        print('read-only session missing', file=sys.stderr); raise SystemExit(4)\n"
            "    if 'search_path=pg_catalog' not in command:\n"
            "        print('fixed search_path missing', file=sys.stderr); raise SystemExit(5)\n"
            "    if sql.strip().lower() == 'show transaction_read_only':\n"
            "        print('on')\n"
            "    elif sql.strip().lower() == 'show search_path':\n"
            "        print('pg_catalog')\n"
            "    if 'jsonb_agg' in sql:\n"
            "        if 'from channels c' in sql:\n"
            "            print('[{\"id\":1,\"name\":\"Provider\",\"type\":1}]')\n"
            "        elif 'embedding-gte-v1' in sql:\n"
            "            print('[{\"channel_name\":\"Embeddings\",\"models\":\"embedding-gte-v1\"}]')\n"
            "        elif 'from channels' in sql and 'channel_id' in sql:\n"
            "            print('[{\"channel_id\":1,\"channel_name\":\"Provider\",\"models\":\"gpt-test\"}]')\n"
            "        else:\n"
            "            print('[{\"database\":\"DBRouterAiAtius\",\"db_user\":\"admin\"}]')\n"
            "    elif not sql.strip().lower().startswith('show '):\n"
            "        print('1')\n"
            "else:\n"
            "    raise SystemExit(3)\n",
            encoding="utf-8",
        )
        fake_sudo.chmod(0o755)

    def tearDown(self):
        self.temp_dir.cleanup()

    def run_cli(self, *args: str) -> subprocess.CompletedProcess[str]:
        env = os.environ.copy()
        env.pop("CLIANYTHING_DB_BACKEND", None)
        env["PATH"] = f"{self.fake_dir}:{env['PATH']}"
        env["FAKE_SUDO_LOG"] = str(self.log_path)
        env["POSTGRES_PASSWORD"] = "fixture-secret-must-not-leak"
        return subprocess.run(
            [str(LAUNCHER_PATH), *args],
            cwd=REPO_ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=env,
            check=False,
        )

    def test_k3s_query_executes_read_only_sql_without_credentials(self):
        proc = self.run_cli("query", "--backend", "k3s", "--read-only", "select 1", "--format", "raw")

        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "1")
        log = self.log_path.read_text(encoding="utf-8")
        self.assertIn('"exec"', log)
        self.assertIn('router-ai-atius-postgres-0', log)
        self.assertIn('$POSTGRES_USER', log)
        self.assertIn("default_transaction_read_only=on", log)
        self.assertIn("search_path=pg_catalog", log)
        self.assertNotIn("search_path=pg_catalog,public", log)
        self.assertNotIn("fixture-secret-must-not-leak", proc.stdout + proc.stderr + log)

    def test_k3s_operational_database_commands_use_prepared_backend(self):
        commands = [
            ("providers", "--all", "--format", "json"),
            ("embeddings", "--format", "json"),
            ("models", "--from-channels", "--format", "json"),
        ]
        for command in commands:
            with self.subTest(command=command[0]):
                proc = self.run_cli(*command, "--backend", "k3s")
                self.assertEqual(proc.returncode, 0, proc.stderr)
                self.assertIsInstance(json.loads(proc.stdout), list)

        log = self.log_path.read_text(encoding="utf-8")
        self.assertEqual(log.count('"exec"'), 3)
        self.assertEqual(log.count("default_transaction_read_only=on"), 3)
        self.assertEqual(log.count("search_path=pg_catalog,public"), 3)

    def test_k3s_status_uses_kubectl_exec_and_reports_ready(self):
        proc = self.run_cli("status", "--backend=k3s", "--strict")

        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("router-ai-atius/router-ai-atius-postgres-0", proc.stdout)
        self.assertIn("Ready node=atius-srv-1", proc.stdout)
        self.assertIn("db", proc.stdout)
        self.assertNotIn("fixture-secret-must-not-leak", proc.stdout + proc.stderr)

    def test_k3s_query_requires_read_only_flag(self):
        proc = self.run_cli("query", "--backend", "k3s", "select 1")

        self.assertEqual(proc.returncode, 2)
        self.assertIn("exige --read-only", proc.stderr)
        self.assertFalse(self.log_path.exists())

    def test_k3s_read_only_query_rejects_mutating_sql_before_exec(self):
        proc = self.run_cli("query", "--backend", "k3s", "--read-only", "delete from channels")

        self.assertEqual(proc.returncode, 2)
        self.assertIn("fora da grammar allowlisted", proc.stderr)
        calls = [json.loads(line) for line in self.log_path.read_text(encoding="utf-8").splitlines()]
        self.assertEqual(len(calls), 1)
        self.assertIn("get", calls[0])
        self.assertNotIn("exec", calls[0])

    def test_k3s_read_only_query_accepts_only_closed_grammar(self):
        for sql in (
            "select 1",
            "show transaction_read_only",
            "show search_path",
            "select pg_catalog.count(*) from public.channels",
            "select id, name from public.channels order by id limit 1000",
            "select relname, relkind from pg_catalog.pg_class order by relname",
        ):
            with self.subTest(sql=sql):
                proc = self.run_cli("query", "--backend", "k3s", "--read-only", sql, "--format", "raw")
                self.assertEqual(proc.returncode, 0, proc.stderr)

    def test_k3s_read_only_query_rejects_functions_casts_and_operators_before_exec(self):
        sql_cases = (
            "select lower(1)",
            "select cast(1 as public.user_defined_type)",
            "select 1::public.user_defined_type",
            "select public.user_defined_aggregate(id) from public.channels",
            "select pg_catalog.sum(id) from public.channels",
            "select pg_backup_start('unsafe')",
            "select pg_try_advisory_lock(42)",
            "select pg_advisory_lock(42)",
            "select nextval('channels_id_seq')",
            "select public.mutating_function()",
            'select "pg_backup_start"(\'unsafe\')',
            "select set_config('application_name', 'mutated', false)",
            "select id from public.channels where id = 1",
        )
        for sql in sql_cases:
            with self.subTest(sql=sql):
                before = self.log_path.read_text(encoding="utf-8").splitlines() if self.log_path.exists() else []
                proc = self.run_cli("query", "--backend", "k3s", "--read-only", sql, "--format", "raw")
                self.assertEqual(proc.returncode, 2)
                self.assertRegex(
                    proc.stderr,
                    r"casts nao sao permitidos|identificadores quoted nao sao permitidos|"
                    r"literals textuais nao sao permitidos|operadores nao sao permitidos|"
                    r"parenteses nao sao permitidos|fora da grammar allowlisted",
                )
                after = [json.loads(line) for line in self.log_path.read_text(encoding="utf-8").splitlines()]
                new_calls = after[len(before):]
                self.assertEqual(len(new_calls), 1)
                self.assertIn("get", new_calls[0])
                self.assertNotIn("exec", new_calls[0])

    def test_k3s_read_only_query_rejects_non_allowlisted_tables_and_columns(self):
        for sql, message in (
            ("select id from public.users", "tabela nao allowlisted"),
            ("select key from public.channels", "coluna nao allowlisted"),
            ("select * from public.channels", "operadores nao sao permitidos"),
            ("select id from channels", "fora da grammar allowlisted"),
        ):
            with self.subTest(sql=sql):
                proc = self.run_cli("query", "--backend", "k3s", "--read-only", sql)
                self.assertEqual(proc.returncode, 2)
                self.assertIn(message, proc.stderr)

    def test_k3s_server_session_reports_read_only_and_fixed_search_path(self):
        transaction = self.run_cli(
            "query", "--backend", "k3s", "--read-only", "show transaction_read_only", "--format", "raw"
        )
        search_path = self.run_cli(
            "query", "--backend", "k3s", "--read-only", "show search_path", "--format", "raw"
        )

        self.assertEqual(transaction.returncode, 0, transaction.stderr)
        self.assertEqual(transaction.stdout.strip(), "on")
        self.assertEqual(search_path.returncode, 0, search_path.stderr)
        self.assertEqual(search_path.stdout.strip(), "pg_catalog")


if __name__ == "__main__":
    unittest.main()
