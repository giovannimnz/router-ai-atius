---
phase: 23-long-context-alias-validation
plan: "01"
type: summary
status: complete
completed_at: "2026-07-08T23:55:00-03:00"
---

# 23-01 Summary

## Result

The long-context alias harness is complete in the scope defined by the plan for
this execution: local/static safety and contract validation only.

## Verified

- `bash -n scripts/test-long-context-aliases.sh`
- `python3 scripts/test_long_context_aliases_static_test.py`
- allowlist remains restricted to:
  - `gpt-5.5`
  - `gpt-5.5-1m`
  - `gpt-5.4`
  - `gpt-5.4-1m`
- default sizes still include `1000000`
- `ENABLE_1M=YES_I_ACCEPT_COSTS` gate remains mandatory
- `BASE_EXPECT_REJECT_FROM` default is now `1000000`
- JSONL evidence path remains `logs/long-context-aliases/`
- `/v1/chat/completions` remains the primary request surface

## Deferred

- Live expensive requests remain operator-triggered only.
- Existing historical UAT is preserved in `23-UAT.md`, but this execution did
  not perform new paid 250k-1M live requests.

## Outcome

Phase 23 is complete as a safe local/static validation harness.
Any future paid/live 1M run should start from `scripts/test-long-context-aliases.sh`
with explicit operator approval and budget acceptance.
