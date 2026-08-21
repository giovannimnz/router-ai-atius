---
phase: 25
phase_name: "embedding-governor-auto-workload-inference"
project: "router-ai-atius"
generated: "2026-07-09T14:37:28-03:00"
counts:
  decisions: 4
  lessons: 4
  patterns: 4
  surprises: 4
missing_artifacts: []
---

# Phase 25 Learnings: embedding-governor-auto-workload-inference

## Decisions

### Router Owns Default Workload Inference
The router was made authoritative for unlabeled governed embedding requests by enabling automatic workload inference by default.

**Rationale:** This keeps the safety contract inside the router instead of depending on every client to remember `X-Embedding-Workload`.
**Source:** 25-01-SUMMARY.md

---

### TEI Safety Uses Fail-Closed Cap Instead of Transparent Recomposition
Governed `embedding-gte-v1` arrays above four input items were rejected at the relay boundary instead of being split and recomposed automatically.

**Rationale:** The phase explicitly chose a narrow fail-closed boundary because there was no safe transparent recomposition pattern for the current relay path.
**Source:** 25-02-SUMMARY.md

---

### No-Header Path Became the Default Client Contract
The default client behavior was defined as omitting `X-Embedding-Workload`, while keeping the header only as an operator steering override.

**Rationale:** This makes Graphify, GBrain, and normal clients simpler while preserving an explicit escape hatch for operational routing.
**Source:** 25-03-SUMMARY.md

---

### Authenticated Live Smoke Remains a Manual Gate
Authenticated `/v1/embeddings` smoke stayed conditional on a secure `ATIUS_ROUTER_TOKEN` export instead of faking green validation when the token was absent.

**Rationale:** The phase treated missing secure auth material as a real validation gate and kept the no-token exit-2 behavior as the safe fallback.
**Source:** 25-03-SUMMARY.md, 25-VERIFICATION.md

---

## Lessons

### BatchModels Still Matter When Auto Inference Is Disabled
Disabling auto-workload does not force unlabeled requests to remain interactive; `BatchModels` still acts as the final fallback.

**Context:** The first version of the fallback test assumed the opposite and had to be corrected to match the real contract.
**Source:** 25-01-SUMMARY.md

---

### Compiled Relay Test Binaries Are More Reliable Than `go test ./relay` Here
The package wrapper for `go test ./relay` is not reliable in this environment, while compiling `./relay` and running the produced test binary works normally.

**Context:** Phase 25 validation had to pivot to `/tmp/relay.test.bin` to keep relay checks deterministic.
**Source:** 25-02-SUMMARY.md

---

### Default-Dimension Logic Needs Explicit Regression Coverage
The smoke helper originally let `embedding-gte-v1` fall through to the wrong default dimension and only later got fixed to `768`.

**Context:** The verifier found this during closeout, which turned a “docs/smoke complete” phase into a real runtime contract correction.
**Source:** 25-VERIFICATION.md

---

### Documentation Can Regress After Code Is Green
Passing code and smoke checks did not prevent the operator manual from regressing later, so the phase needed exact `rg` doc gates and a restore pass.

**Context:** The first verifier pass caught a manual regression after the main plan work was already complete.
**Source:** 25-VERIFICATION.md

---

## Patterns

### Export Scope Checks From the Same Governor Contract Used by Acquire
Relay-facing code should call exported governor helpers instead of duplicating governed-model lists or local heuristics.

**When to use:** Any time another package needs to know whether a model belongs to the governed embedding path.
**Source:** 25-01-SUMMARY.md

---

### Keep Workload Classification Metadata-Only
The classifier should operate only on workload header, item count, and character count, without storing raw embedding input text.

**When to use:** Any queueing, routing, or safety logic that must distinguish `batch` from `interactive` without leaking request content.
**Source:** 25-01-SUMMARY.md, 25-VERIFICATION.md

---

### Stop Relay Tests at the Governor Hook
Relay tests for governor behavior should fail or pass at the governor acquisition hook or synthetic reject path rather than relying on upstream transport.

**When to use:** When validating metadata capture, cap enforcement, or reject behavior in a relay path that would otherwise depend on network I/O.
**Source:** 25-02-SUMMARY.md

---

### Smoke Helpers Should Validate Rows, Dimensions, and Redaction Together
Embedding smoke tooling should check response row count, vector dimension, and output redaction as one contract, while also keeping a no-token exit-before-network path.

**When to use:** Any client-facing embedding smoke that covers both single-input and array-input behavior.
**Source:** 25-03-SUMMARY.md, 25-VERIFICATION.md

---

## Surprises

### The Fallback Test Failed Because the Contract Was Stricter Than Assumed
The initial assumption about disabled auto-workload was wrong: once inference is off, configured `BatchModels` still wins.

**Impact:** The test had to be rewritten, and it clarified that “disable inference” is not the same as “force interactive”.
**Source:** 25-01-SUMMARY.md

---

### Relay Package Tests Hang Even in Narrow Modes
The relay test wrapper hung in this environment even for highly constrained runs, including `-run '^$'`.

**Impact:** Validation strategy had to switch from the standard package command to compiled test binaries, which is an operational quirk worth remembering for future relay work.
**Source:** 25-02-SUMMARY.md

---

### The Missing-Token Path Was Not the Only Validation Blocker
Phase execution started with the token absent, but closeout later revealed a separate bug in default dimension handling even after the safe no-token path had passed.

**Impact:** “Token missing” was not the only risk; authenticated smoke still surfaced contract drift that purely local checks had missed.
**Source:** 25-03-SUMMARY.md, 25-VERIFICATION.md

---

### Docs Regressed After the Main Implementation Was Done
The manual regressed after the primary Phase 25 implementation, and the verifier had to force a restore of the exact Phase 25 guidance.

**Impact:** It justified keeping documentation under the same quality gate as code and smoke helpers instead of treating it as low-risk follow-up text.
**Source:** 25-VERIFICATION.md
