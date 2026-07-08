#!/usr/bin/env python3
"""Unit tests for CLIAnything helper behavior.

These tests stay local: no Podman, no database, no network.
"""

from __future__ import annotations

import asyncio
import importlib.util
import json
import sys
import unittest
import types
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
MODULE_PATH = REPO_ROOT / "tools" / "clianything.py"
MODEL_DETAILED_PATH = REPO_ROOT / "runtime" / "model-detailed" / "model_detailed_fastapi.py"
SMOKE_EMBEDDINGS_PATH = REPO_ROOT / "scripts" / "smoke-embeddings.py"


def load_clianything():
    spec = importlib.util.spec_from_file_location("clianything_under_test", MODULE_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Could not load {MODULE_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def load_model_detailed():
    spec = importlib.util.spec_from_file_location("model_detailed_under_test", MODEL_DETAILED_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Could not load {MODEL_DETAILED_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    try:
        spec.loader.exec_module(module)
    except ModuleNotFoundError:
        install_model_detailed_stubs()
        spec = importlib.util.spec_from_file_location("model_detailed_under_test", MODEL_DETAILED_PATH)
        module = importlib.util.module_from_spec(spec)
        sys.modules[spec.name] = module
        spec.loader.exec_module(module)
    return module


def install_model_detailed_stubs():
    if "httpx" not in sys.modules:
        httpx_stub = types.ModuleType("httpx")

        class _AsyncClient:
            def __init__(self, *args, **kwargs):
                self.is_closed = False

            async def aclose(self):
                self.is_closed = True

        class _Timeout:
            def __init__(self, *args, **kwargs):
                pass

        class _Limits:
            def __init__(self, *args, **kwargs):
                pass

        httpx_stub.AsyncClient = _AsyncClient
        httpx_stub.Timeout = _Timeout
        httpx_stub.Limits = _Limits
        httpx_stub.TimeoutException = type("TimeoutException", (Exception,), {})
        httpx_stub.RequestError = type("RequestError", (Exception,), {})
        sys.modules["httpx"] = httpx_stub

    if "fastapi" not in sys.modules:
        fastapi_stub = types.ModuleType("fastapi")

        class _HTTPException(Exception):
            def __init__(self, status_code=None, detail=None, headers=None):
                super().__init__(detail)
                self.status_code = status_code
                self.detail = detail
                self.headers = headers or {}

        class _Request:
            def __init__(self, *args, **kwargs):
                self.headers = {}
                self.query_params = {}
                self.url = types.SimpleNamespace(path="/")

        class _FastAPI:
            def __init__(self, *args, **kwargs):
                pass

            def get(self, *args, **kwargs):
                def decorator(func):
                    return func

                return decorator

            def post(self, *args, **kwargs):
                def decorator(func):
                    return func

                return decorator

            def api_route(self, *args, **kwargs):
                def decorator(func):
                    return func

                return decorator

            def add_middleware(self, *args, **kwargs):
                return None

            def mount(self, *args, **kwargs):
                return None

        def _identity(*args, **kwargs):
            def decorator(func):
                return func

            return decorator

        fastapi_stub.FastAPI = _FastAPI
        fastapi_stub.Request = _Request
        fastapi_stub.HTTPException = _HTTPException
        fastapi_stub.Depends = lambda *args, **kwargs: None
        fastapi_stub.Cookie = lambda *args, **kwargs: None
        sys.modules["fastapi"] = fastapi_stub

        responses_stub = types.ModuleType("fastapi.responses")

        class _BaseResponse:
            def __init__(self, *args, **kwargs):
                self.content = kwargs.get("content")
                self.status_code = kwargs.get("status_code", 200)
                self.media_type = kwargs.get("media_type")
                self.headers = kwargs.get("headers", {})

        responses_stub.JSONResponse = _BaseResponse
        responses_stub.Response = _BaseResponse
        responses_stub.RedirectResponse = _BaseResponse
        responses_stub.HTMLResponse = _BaseResponse
        sys.modules["fastapi.responses"] = responses_stub

        middleware_stub = types.ModuleType("fastapi.middleware")
        cors_stub = types.ModuleType("fastapi.middleware.cors")
        cors_stub.CORSMiddleware = object
        sys.modules["fastapi.middleware"] = middleware_stub
        sys.modules["fastapi.middleware.cors"] = cors_stub

        security_stub = types.ModuleType("fastapi.security")
        security_stub.HTTPBasic = lambda *args, **kwargs: None
        security_stub.HTTPBasicCredentials = type("HTTPBasicCredentials", (), {})
        sys.modules["fastapi.security"] = security_stub

        static_stub = types.ModuleType("fastapi.staticfiles")
        class _StaticFiles:
            def __init__(self, *args, **kwargs):
                pass

        static_stub.StaticFiles = _StaticFiles
        sys.modules["fastapi.staticfiles"] = static_stub


def load_smoke_embeddings():
    spec = importlib.util.spec_from_file_location("smoke_embeddings_under_test", SMOKE_EMBEDDINGS_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Could not load {SMOKE_EMBEDDINGS_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


cli = load_clianything()
model_detailed = load_model_detailed()
smoke_embeddings = load_smoke_embeddings()


class RedactionTests(unittest.TestCase):
    def test_redact_value_sensitive_columns(self):
        cases = [
            ("key", "test-key-value"),
            ("password", "test-password"),
            ("access_token", "test-access-token"),
            ("client_secret", "test-client-secret"),
            ("secret", "test-secret"),
            ("header_override", '{"Authorization":"Bearer test-token"}'),
            ("private_data", '{"credential":"test"}'),
            ("api_key", "test-api-key"),
        ]

        for column, value in cases:
            with self.subTest(column=column):
                self.assertEqual(cli.redact_value(column, value), "<redacted>")

    def test_redact_value_preserves_empty_values(self):
        self.assertIsNone(cli.redact_value("password", None))
        self.assertEqual(cli.redact_value("password", ""), "")

    def test_redact_rows_redacts_option_value_for_secret_key(self):
        rows = [
            {
                "id": 1,
                "key": "openai_api_key",
                "value": "test-secret-value",
                "name": "Visible Name",
            }
        ]

        redacted = cli.redact_rows(rows)

        self.assertEqual(redacted[0]["key"], "<redacted>")
        self.assertEqual(redacted[0]["value"], "<redacted>")
        self.assertEqual(redacted[0]["name"], "Visible Name")

    def test_redact_json_for_api_payloads(self):
        payload = {
            "ok": True,
            "data": {
                "access_token": "test-access-token",
                "client_secret": "test-client-secret",
                "nested": [{"password": "test-password"}],
            },
        }

        redacted = cli.redact_json(payload)

        self.assertTrue(redacted["ok"])
        self.assertEqual(redacted["data"]["access_token"], "<redacted>")
        self.assertEqual(redacted["data"]["client_secret"], "<redacted>")
        self.assertEqual(redacted["data"]["nested"][0]["password"], "<redacted>")

    def test_redact_text_for_non_json_api_bodies(self):
        body = "Bearer " + "testtoken123456 " + "password" + "=plaintext " + "sk-" + "testtoken1234567890"

        redacted = cli.redact_text(body)

        self.assertIn("Bearer <redacted>", redacted)
        self.assertIn("password=<redacted>", redacted)
        self.assertIn("sk-redacted", redacted)
        self.assertNotIn("testtoken123456", redacted)
        self.assertNotIn("plaintext", redacted)

    def test_emit_json_redacts_rows_before_printing(self):
        rows = [{"id": 1, "key": "provider_api_key", "value": "test-secret"}]

        # Capture by temporarily replacing stdout instead of invoking any live command.
        from io import StringIO

        original_stdout = sys.stdout
        try:
            sys.stdout = StringIO()
            cli.emit(rows, "json")
            output = sys.stdout.getvalue()
        finally:
            sys.stdout = original_stdout

        parsed = json.loads(output)
        self.assertEqual(parsed[0]["key"], "<redacted>")
        self.assertEqual(parsed[0]["value"], "<redacted>")


class ReadOnlyGuardTests(unittest.TestCase):
    def test_ensure_read_only_allows_expected_prefixes(self):
        allowed = [
            "select 1",
            "WITH q AS (select 1) select * from q",
            "show server_version",
            "explain select * from channels",
            "/* comment */ -- another comment\n select count(*) from channels",
        ]

        for sql in allowed:
            with self.subTest(sql=sql):
                cli.ensure_read_only(sql)

    def test_ensure_read_only_blocks_mutating_prefixes(self):
        blocked = [
            "insert into channels(id) values (1)",
            "update channels set priority = 0",
            "delete from channels",
            "drop table channels",
            "alter table channels add column test int",
            "truncate channels",
            "create table test(id int)",
            "grant select on channels to public",
            "revoke select on channels from public",
            "copy channels to stdout",
            "call refresh_channels()",
            "do $$ begin end $$",
            "vacuum channels",
            "reindex table channels",
        ]

        for sql in blocked:
            with self.subTest(sql=sql):
                with self.assertRaises(cli.CliError):
                    cli.ensure_read_only(sql)

    def test_ensure_read_only_blocks_mutating_verb_inside_allowed_query(self):
        blocked = [
            "select * from channels; update channels set priority = 0",
            "with q as (delete from channels returning *) select * from q",
            "explain drop table channels",
        ]

        for sql in blocked:
            with self.subTest(sql=sql):
                with self.assertRaises(cli.CliError):
                    cli.ensure_read_only(sql)


class ParserAndIdentifierTests(unittest.TestCase):
    def test_psql_base_cmd_defaults_to_host_canonical_db(self):
        self.assertEqual(cli.DB_BACKEND, "host")
        self.assertEqual(cli.DB_NAME, "DBRouterAiAtius")
        self.assertEqual(cli.psql_base_cmd("psql"), ["sudo", "-u", "postgres", "psql", "-d", "DBRouterAiAtius"])

    def test_psql_base_cmd_supports_legacy_podman_backend(self):
        original_backend = cli.DB_BACKEND
        original_name = cli.DB_NAME
        try:
            cli.DB_BACKEND = "podman"
            cli.DB_NAME = "DBRouterAiAtius"
            self.assertEqual(
                cli.psql_base_cmd("psql"),
                ["podman", "exec", "-i", "postgres", "psql", "-U", "admin", "-d", "DBRouterAiAtius"],
            )
        finally:
            cli.DB_BACKEND = original_backend
            cli.DB_NAME = original_name

    def test_parse_key_values_accepts_values_with_equals(self):
        parsed = cli.parse_key_values(["priority=0", "base_url=https://example.test/a=b", "enabled=true"])

        self.assertEqual(
            parsed,
            {
                "priority": "0",
                "base_url": "https://example.test/a=b",
                "enabled": "true",
            },
        )

    def test_parse_key_values_rejects_invalid_items(self):
        invalid = ["priority", "=0"]

        for item in invalid:
            with self.subTest(item=item):
                with self.assertRaises(cli.CliError):
                    cli.parse_key_values([item])

    def test_parse_key_values_last_duplicate_wins(self):
        self.assertEqual(cli.parse_key_values(["priority=1", "priority=0"]), {"priority": "0"})

    def test_quote_ident_accepts_simple_postgres_identifiers(self):
        self.assertEqual(cli.quote_ident("channels"), '"channels"')
        self.assertEqual(cli.quote_ident("usage_tracking_1"), '"usage_tracking_1"')
        self.assertEqual(cli.quote_ident("_private"), '"_private"')

    def test_quote_ident_rejects_invalid_identifiers(self):
        invalid = ["", "1channels", "channel-name", "public.channels", "channels;drop", "has space"]

        for name in invalid:
            with self.subTest(name=name):
                with self.assertRaises(cli.CliError):
                    cli.quote_ident(name)

    def test_normalize_resource_accepts_canonical_names_and_aliases(self):
        self.assertEqual(cli.normalize_resource("channels").table, "channels")
        self.assertEqual(cli.normalize_resource("channel").table, "channels")
        self.assertEqual(cli.normalize_resource("providers").table, "channels")
        self.assertEqual(cli.normalize_resource("usage-tracking").table, "usage_tracking")
        self.assertEqual(cli.normalize_resource("group").table, "prefill_groups")
        self.assertEqual(cli.normalize_resource("tasks-log").table, "tasks")

    def test_normalize_resource_rejects_unknown_resource(self):
        with self.assertRaises(cli.CliError):
            cli.normalize_resource("not-a-resource")


class EndpointCoverageTests(unittest.TestCase):
    def _manifest_entry(self, **overrides):
        entry = {
            "group": "test",
            "name": "test-endpoint",
            "method": "GET",
            "path": "/api/test",
            "doc": "test.mdx",
            "classification": "read-only",
            "cli_command": "clianything endpoint invoke GET /api/test",
            "safe_default": True,
            "requires_auth": True,
            "notes": "unit fixture",
        }
        entry.update(overrides)
        return entry

    def test_management_docs_and_manifest_are_complete(self):
        if not cli.MANAGEMENT_DOCS_ROOT.exists():
            self.skipTest(f"management docs tree not present: {cli.MANAGEMENT_DOCS_ROOT}")
        endpoints, docs_without_ops = cli.parse_management_docs()
        entries = cli.load_endpoint_manifest()
        if hasattr(cli, "coverage_rows"):
            _rows, problems = cli.coverage_rows(entries, endpoints)
            missing = []
            extra = []
        else:
            payload = cli.render_coverage_payload()
            problems = payload["problems"]
            missing = payload["missing_from_manifest"]
            extra = payload["extra_in_manifest"]

        self.assertEqual(len(endpoints), 158)
        self.assertEqual([item["doc"] for item in docs_without_ops], ["auth.mdx"])
        self.assertEqual(len(entries), 158)
        self.assertEqual(problems, [])
        self.assertEqual(missing, [])
        self.assertEqual(extra, [])

    def test_manifest_entries_have_required_contract(self):
        entries = cli.load_endpoint_manifest()
        required = {
            "group",
            "name",
            "method",
            "path",
            "doc",
            "classification",
            "cli_command",
            "safe_default",
            "requires_auth",
            "notes",
        }

        for entry in entries:
            with self.subTest(endpoint=f"{entry.get('method')} {entry.get('path')}"):
                self.assertTrue(required.issubset(entry))
                self.assertIn(entry["classification"], cli.ENDPOINT_CLASSIFICATIONS)
                self.assertIsInstance(entry["requires_auth"], bool)

    def test_validate_manifest_rejects_generic_api_wrapper_variants(self):
        entries = [
            self._manifest_entry(
                classification="api-action",
                cli_command="clianything api GET /api/channel/test/{id}",
            )
        ]

        problems = cli.validate_endpoint_manifest(entries)

        self.assertTrue(any("api-action exige cli_command tipado" in item for item in problems))
        self.assertTrue(any("wrapper generico nao conta como paridade" in item for item in problems))

    def test_manifest_request_match_handles_templates_and_query(self):
        self.assertTrue(cli.endpoint_template_matches("/api/channel/test/{id}", "/api/channel/test/3"))
        self.assertFalse(cli.endpoint_template_matches("/api/channel/test/{id}", "/api/channel/test/3/extra"))
        entry = cli.find_manifest_entry_for_request("GET", "/api/channel/test/3?probe=1")
        self.assertIsNotNone(entry)
        self.assertEqual(entry["path"], "/api/channel/test/{id}")


class Phase19ProviderRoutingTests(unittest.TestCase):
    def test_write_channel_from_source_key_preview_is_secret_safe(self):
        sql = cli.write_channel_from_source_key(
            source_id=2,
            name="DeepSeek",
            type_value=43,
            base_url="https://api.deepseek.com",
            models="deepseek-v4-pro,deepseek-v4-flash",
            group="default",
            priority=0,
            weight=0,
            status=1,
        )

        self.assertIn("select", sql.lower())
        self.assertIn("c.key", sql)
        self.assertIn("DeepSeek", sql)
        self.assertIn("https://api.deepseek.com", sql)
        self.assertNotIn("sample-secret-value", sql)

    def test_legacy_split_channel_detection_blocks_old_shapes(self):
        self.assertTrue(
            cli.is_legacy_split_channel(
                "DeepSeek - Anthropic-Compatible",
                "https://api.deepseek.com/anthropic",
            )
        )
        self.assertTrue(
            cli.is_legacy_split_channel(
                "MiniMax - Embeddings",
                "https://api.minimax.io",
            )
        )
        self.assertFalse(cli.is_legacy_split_channel("DeepSeek", "https://api.deepseek.com"))
        self.assertFalse(cli.is_legacy_split_channel("MiniMax", "https://api.minimax.io"))

    def test_phase19_actions_consolidate_provider_channels(self):
        actions = cli.build_phase19_provider_actions(None)
        labels = [label for _resource, _sql, label in actions]
        combined_sql = "\n".join(sql for _resource, sql, _label in actions)

        self.assertTrue(any("OpenAI - Codex" in label for label in labels))
        self.assertTrue(any("consolidate MiniMax" in label for label in labels))
        self.assertTrue(any("consolidate DeepSeek" in label for label in labels))
        self.assertTrue(any("disable merged provider channels" in label for label in labels))
        self.assertIn("type = 35", combined_sql)
        self.assertIn("type = 43", combined_sql)
        self.assertIn("MiniMax-M3,MiniMax-M2.7,MiniMax-M2.5-highspeed,MiniMax-M2.5,embo-01", combined_sql)
        self.assertIn("deepseek-v4-pro,deepseek-v4-flash", combined_sql)
        self.assertNotIn("OpenAI - Embeddings", combined_sql)
        self.assertNotIn("DeepSeek - Anthropic-Compatible", combined_sql)

    def test_embeddings_overview_sql_targets_embeddings_channels(self):
        sql = cli.build_embeddings_overview_sql()

        self.assertIn("Embeddings", sql)
        self.assertIn("embo-01", sql)
        self.assertIn("text-embedding-3-small", sql)
        self.assertIn("text-embedding-3-large", sql)
        self.assertIn("enabled_abilities", sql)

    def test_enrich_openai_models_response_anthropic_uses_deepseek_allowlist(self):
        payload = {
            "data": [
                {"id": "deepseek-v4-flash", "supported_endpoint_types": ["openai"]},
                {"id": "deepseek-v4-beta", "supported_endpoint_types": ["openai"]},
                {"id": "MiniMax-M3", "supported_endpoint_types": ["openai"]},
            ]
        }

        enriched = model_detailed.enrich_openai_models_response_anthropic(payload)
        ids = [item["id"] for item in enriched["data"]]

        self.assertIn("deepseek-v4-flash", ids)
        self.assertIn("MiniMax-M3", ids)
        self.assertNotIn("deepseek-v4-beta", ids)

    def test_backend_pricing_map_converts_ratios_to_per_token_prices(self):
        pricing_map = model_detailed.build_backend_pricing_map(
            {
                "data": [
                    {
                        "model_name": "gpt-5.5",
                        "model_ratio": 2.5,
                        "completion_ratio": 6,
                    },
                    {
                        "model_name": "text-embedding-3-small",
                        "model_ratio": 0.01,
                        "completion_ratio": 1,
                    },
                ]
            }
        )

        self.assertEqual(pricing_map["gpt-5.5"]["pricing"]["prompt"], "0.000005")
        self.assertEqual(pricing_map["gpt-5.5"]["pricing"]["completion"], "0.00003")
        self.assertEqual(pricing_map["gpt-5.5"]["input_price"], 5)
        self.assertEqual(pricing_map["gpt-5.5"]["output_price"], 30)
        self.assertFalse(pricing_map["gpt-5.5"]["pricing_estimated"])
        self.assertEqual(pricing_map["text-embedding-3-small"]["input_price"], 0.02)

    def test_model_enrichment_uses_backend_pricing_for_codex_and_embeddings(self):
        backend_pricing = model_detailed.build_backend_pricing_map(
            {
                "data": [
                    {"model_name": "gpt-5.5", "model_ratio": 2.5, "completion_ratio": 6},
                    {"model_name": "embo-01", "model_ratio": 0.0345, "completion_ratio": 1},
                ]
            }
        )
        payload = {
            "data": [
                {"id": "gpt-5.5", "object": "model", "created": 1, "owned_by": "codex"},
                {"id": "embo-01", "object": "model", "created": 1, "owned_by": "minimax"},
            ]
        }

        enriched = asyncio.run(model_detailed.enrich_models_response(payload, backend_pricing))
        by_id = {item["id"]: item for item in enriched["data"]}

        self.assertEqual(by_id["gpt-5.5"]["pricing"]["prompt"], "0.000005")
        self.assertEqual(by_id["gpt-5.5"]["pricing"]["completion"], "0.00003")
        self.assertNotIn("pricing_source", by_id["gpt-5.5"])
        self.assertNotIn("pricing_estimated", by_id["gpt-5.5"])
        self.assertEqual(by_id["embo-01"]["pricing"]["prompt"], "0.000000069")
        self.assertEqual(by_id["embo-01"]["supported_endpoint_types"], ["embeddings"])
        self.assertNotIn("pricing_source", by_id["embo-01"])
        self.assertNotIn("pricing_estimated", by_id["embo-01"])

    def test_anthropic_enrichment_uses_backend_price_fields(self):
        payload = {"data": [{"model": "MiniMax-M3", "channel_type": 14}]}
        backend_pricing = model_detailed.build_backend_pricing_map(
            {"data": [{"model_name": "MiniMax-M3", "model_ratio": 0.15, "completion_ratio": 4}]}
        )

        enriched = model_detailed.enrich_models_response_anthropic(payload, backend_pricing)

        self.assertEqual(enriched["data"][0]["input_price"], 0.3)
        self.assertEqual(enriched["data"][0]["output_price"], 1.2)
        self.assertNotIn("pricing_source", enriched["data"][0])
        self.assertNotIn("pricing_estimated", enriched["data"][0])

    def test_normalise_embedding_input_preserves_query_and_db(self):
        body_query, should_query = model_detailed._normalise_embedding_input(
            {"model": "embo-01", "input": "hello"}
        )
        body_db, should_db = model_detailed._normalise_embedding_input(
            {"model": "embo-01", "input": "hello", "type": "db"}
        )

        self.assertTrue(should_query)
        self.assertEqual(body_query["type"], "query")
        self.assertEqual(body_query["input"], ["hello"])
        self.assertTrue(should_db)
        self.assertEqual(body_db["type"], "db")
        self.assertEqual(body_db["input"], ["hello"])

    def test_strip_thinking_blocks_removes_anthropic_thinking_payloads(self):
        payload = {
            "content": [
                {"type": "thinking", "thinking": "internal chain"},
                {"type": "text", "text": "<think>\ninternal chain\n</think>\n\nOK"},
            ]
        }

        cleaned = json.loads(model_detailed.strip_thinking_blocks(json.dumps(payload).encode("utf-8")))

        self.assertEqual(cleaned["content"], [{"type": "text", "text": "OK"}])

    def test_rate_queue_provider_maps_model_families(self):
        self.assertEqual(
            model_detailed._rate_queue_provider("v1/chat/completions", {"model": "MiniMax-M3"}),
            "minimax-chat",
        )
        self.assertEqual(
            model_detailed._rate_queue_provider("v1/messages", {"model": "deepseek-v4-flash"}),
            "deepseek-chat",
        )
        self.assertEqual(
            model_detailed._rate_queue_provider("v1/chat/completions", {"model": "gpt-5.5"}),
            "codex-chat",
        )
        self.assertEqual(
            model_detailed._rate_queue_provider("v1/embeddings", {"model": "embo-01"}),
            "minimax-embeddings",
        )
        self.assertEqual(
            model_detailed._rate_queue_provider("v1/embeddings", {"model": "text-embedding-3-small"}),
            "openai-embeddings",
        )
        self.assertIsNone(model_detailed._rate_queue_provider("v1/models", {"model": "MiniMax-M3"}))

    def test_rate_queue_retry_classification_avoids_quota_retries(self):
        class FakeResponse:
            def __init__(self, status_code, text="", data=None, headers=None):
                self.status_code = status_code
                self.text = text
                self.content = text.encode("utf-8")
                self.headers = headers or {}
                self._data = data

            def json(self):
                if self._data is None:
                    raise ValueError("not json")
                return self._data

        self.assertTrue(
            model_detailed._is_retryable_upstream_response(
                FakeResponse(529, "The server cluster is currently under high load. Please retry.")
            )
        )
        self.assertFalse(
            model_detailed._is_retryable_upstream_response(
                FakeResponse(429, "The usage limit has been reached")
            )
        )
        self.assertTrue(
            model_detailed._is_retryable_upstream_response(
                FakeResponse(200, data={"base_resp": {"status_code": 1002, "status_msg": "rate limit exceeded(RPM)"}})
            )
        )

    def test_rate_queue_headers_are_added_without_secrets(self):
        headers = model_detailed._rate_queue_response_headers(
            {"provider": "minimax-chat", "wait_ms": 1500, "retries": 2}
        )

        self.assertEqual(headers["X-Atius-Rate-Queue"], "minimax-chat")
        self.assertEqual(headers["X-Atius-Rate-Queue-Wait-Ms"], "1500")
        self.assertEqual(headers["X-Atius-Rate-Retry-Count"], "2")

    def test_smoke_embeddings_helpers_cover_payload_shape_and_redaction(self):
        payload = smoke_embeddings.build_embedding_payload(
            model="embo-01",
            input_text="hello",
            embedding_type="db",
        )
        self.assertEqual(payload["type"], "db")
        self.assertEqual(payload["input"], "hello")

        openai_payload = smoke_embeddings.build_embedding_payload(
            model="embedding-gte-v1",
            input_text="hello",
            openai_dimensions=768,
        )
        self.assertNotIn("type", openai_payload)
        self.assertEqual(openai_payload["dimensions"], 768)

        array_payload = smoke_embeddings.build_embedding_payload(
            model="embedding-gte-v1",
            input_text="ignored",
            input_items=["a", "b"],
        )
        self.assertEqual(array_payload["input"], ["a", "b"])
        self.assertNotIn("type", array_payload)

        self.assertEqual(smoke_embeddings.expected_embedding_dimension("embedding-gte-v1"), 768)
        self.assertEqual(smoke_embeddings.expected_embedding_dimension("embo-01"), 768)
        self.assertEqual(smoke_embeddings.expected_embedding_dimension("text-embedding-3-large"), 3072)
        self.assertEqual(smoke_embeddings.expected_embedding_dimension("text-embedding-3-small"), 1536)
        self.assertIsNone(smoke_embeddings.expected_embedding_dimension("embedding-gte-v1", "skip"))
        self.assertEqual(smoke_embeddings.expected_embedding_dimension("embedding-gte-v1", "1024"), 1024)

        self.assertEqual(smoke_embeddings.assert_embedding_vector_shape([1.0, 2.0, 3.0], 3, "embo-01"), 3)
        with self.assertRaises(ValueError):
            smoke_embeddings.assert_embedding_vector_shape(["x"], 1, "embo-01")

        scrubbed = smoke_embeddings._scrub(
            "GroupId=123 Authorization=Bearer abc ATIUS_ROUTER_TOKEN=secret",
            ["secret", "abc"],
        )
        self.assertNotIn("secret", scrubbed)
        self.assertNotIn("abc", scrubbed)
        self.assertNotIn("GroupId", scrubbed)


if __name__ == "__main__":
    unittest.main()
