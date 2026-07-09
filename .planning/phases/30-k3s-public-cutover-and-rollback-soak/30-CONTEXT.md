# Phase 30 Context

## Objective

Perform the public cutover only after Phase 29 has produced a real go decision.

This phase owns:

- Apache retarget to the k3s backend
- public smoke validation
- rollback decision and execution if needed
- soak window with Podman still available as rollback

## Hard Gate

Do not start unless Phase 29 has:

- real shadow rollout evidence
- real restore rehearsal evidence
- real smoke evidence
- explicit go/no-go decision

## Success Definition

Either:

- public traffic moves to k3s and survives the defined soak checks

or:

- Apache is restored to Podman and rollback checks pass

with full evidence captured.
