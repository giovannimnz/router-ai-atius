#!/usr/bin/env python3
import re
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("test-long-context-aliases.sh")


class LongContextAliasScriptTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.source = SCRIPT.read_text(encoding="utf-8")

    def test_model_allowlist_is_restricted_to_codex_models(self):
        self.assertIn("all|aliases|base|gpt-5.5|gpt-5.5-1m|gpt-5.4|gpt-5.4-1m", self.source)
        self.assertIn('"gpt-5.5" "gpt-5.5-1m" "gpt-5.4" "gpt-5.4-1m"', self.source)
        self.assertNotIn("MiniMax", self.source)
        self.assertNotIn("deepseek", self.source)

    def test_one_million_step_requires_explicit_cost_acknowledgement(self):
        self.assertIn('ENABLE_1M=YES_I_ACCEPT_COSTS', self.source)
        self.assertRegex(
            self.source,
            r'"\$size" == "1000000".*"\$ENABLE_1M" != "YES_I_ACCEPT_COSTS"',
            msg="1M requests must be guarded before execution",
        )

    def test_large_steps_require_model_and_size_confirmation(self):
        self.assertIn('AUTO_CONFIRM_LARGE_STEPS="${AUTO_CONFIRM_LARGE_STEPS:-}"', self.source)
        self.assertIn('AUTO_CONFIRM_LARGE_STEPS" == "YES_I_ACCEPT_COSTS"', self.source)
        self.assertIn("Type RUN ${size} ${model} to continue", self.source)
        self.assertIn('[[ "$answer" != "RUN ${size} ${model}" ]]', self.source)

    def test_base_models_have_limit_guard_expectation(self):
        self.assertIn('BASE_EXPECT_REJECT_FROM="${BASE_EXPECT_REJECT_FROM:-300000}"', self.source)
        self.assertIn('[[ "$1" == "gpt-5.5" || "$1" == "gpt-5.4" ]]', self.source)
        self.assertIn('printf \'reject\'', self.source)
        self.assertIn('"kind": "base_limit_guard"', self.source)

    def test_router_token_is_never_echoed(self):
        risky_patterns = [
            r"echo\s+.*\$ROUTER_TEST_KEY",
            r"printf\s+.*\$ROUTER_TEST_KEY",
            r"tee\s+.*\$ROUTER_TEST_KEY",
        ]
        for pattern in risky_patterns:
            self.assertIsNone(re.search(pattern, self.source))
        self.assertIn("ROUTER_TEST_KEY will not be printed", self.source)

    def test_logs_are_written_under_gitignored_logs_directory(self):
        self.assertIn('LOG_DIR="${LOG_DIR:-logs/long-context-aliases}"', self.source)
        self.assertIn('"chat_reasoning"', self.source)
        self.assertIn('"kind": "stream_smoke"', self.source)
        self.assertIn('"kind": "preflight_models"', self.source)


if __name__ == "__main__":
    unittest.main()
