---
phase: 29-k3s-shadow-restore-and-go-no-go
plan: "01"
status: complete
completed_at: 2026-07-13T04:58:00Z
---

# Phase 29 Plan 01: Bootstrap seguro no atius-srv-1

## Resultado

A capacidade e os invariantes de bootstrap do shadow foram validados ao vivo.
Nenhum trafego publico foi alterado.

- backup root-only criado em
  `/var/backups/router-ai-atius-k3s-phase29-20260713T033225Z`;
- 27,4 GB de reclaim conservador registrados, com 25% livre;
- `DiskPressure=False` e taint ausente por cinco minutos continuos;
- label `atius.com.br/router-ai-atius-node=true` exclusivo no `atius-srv-1`;
- Secret criado a partir do Vault, validado somente pelos nomes das tres chaves;
- imagem arm64 live importada no namespace containerd `k8s.io` e tagueada pela
  referencia digest exata usada no manifest;
- quota da operacao comprovada como `80000 100000`;
- Podman e Apache permaneceram como runtime/edge publicos.

## Evidencias

Diretorio sanitizado da execucao:

`~/.local/state/router-ai-atius/phase29/run-20260713T040715Z`

Arquivos principais:

- `cleanup.json`: status `go`, reclaim, margem, cluster UID e estabilidade;
- `cleanup-items.jsonl`: allowlist efetivamente processada;
- `bootstrap.json`: label, nomes de chaves, digest, image ref, manifest hash e
  `cpu.max`.

Valores de Secrets nao foram gravados nesses artefatos.

## Validacao

- `tests/phase29-k3s-router-selftest.sh`: PASS sob `profile-run`;
- ShellCheck e `bash -n`: PASS;
- review independente da Wave 29-01: PASS;
- imagem live e imagem de rollback Podman preservadas;
- manifests estruturais: PASS.

## Proxima Wave

Executar `29-02`: dump fresco do PostgreSQL Podman, apply apenas do target
PostgreSQL, patch `Retain`, restore real e verificacao de invariantes antes de
subir Redis/router.
