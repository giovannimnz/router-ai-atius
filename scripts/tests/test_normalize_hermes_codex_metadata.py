import importlib.util
import unittest
from pathlib import Path


SCRIPT = Path(__file__).parents[1] / "normalize-hermes-codex-metadata.py"
SPEC = importlib.util.spec_from_file_location("normalize_hermes_codex_metadata", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)


class NormalizeHermesCodexMetadataTest(unittest.TestCase):
    def test_replaces_retired_default_and_normalizes_codex_entries(self):
        source = """model:
  provider: custom:atius-router
  default: gpt-5.4-mini
  context_length: 1048576

custom_providers:
  atius-router:
    models:
      gpt-5.6-sol:
        context_length: 1050000
      gpt-5.6-luna:
        context_length: 1000000
      gpt-5.3-codex-spark:
        context_length: 272000
"""

        normalized, changes = MODULE.normalize_config(source, replacement_default="gpt-5.6-sol")

        self.assertIn("default: gpt-5.6-sol", normalized)
        self.assertEqual(3, normalized.count("context_length: 272000"))
        self.assertIn("gpt-5.3-codex-spark:\n        context_length: 128000", normalized)
        self.assertGreaterEqual(len(changes), 4)

    def test_is_idempotent_and_inserts_missing_model_context(self):
        source = """model:
  default: gpt-5.6-terra
  context_length: 272000
custom_providers:
  atius-router:
    models:
      gpt-5.6-terra:
        context_length: 272000
      gpt-5.5:
        label: GPT-5.5
"""

        normalized, changes = MODULE.normalize_config(source)
        second, second_changes = MODULE.normalize_config(normalized)

        self.assertIn("gpt-5.5:\n        context_length: 272000", normalized)
        self.assertTrue(changes)
        self.assertEqual(normalized, second)
        self.assertEqual([], second_changes)


if __name__ == "__main__":
    unittest.main()
