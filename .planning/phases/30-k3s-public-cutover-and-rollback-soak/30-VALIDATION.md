---
phase: 30
phase_slug: k3s-public-cutover-and-rollback-soak
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-13
---

# Phase 30 — Estratégia de Validação Nyquist

## Arquitetura de validação

A fase precisa provar continuamente seis superfícies:

1. GO da Phase 29 e envelope de backup íntegros antes de mutação;
2. somente `DBRouterAiAtius` muda no PgBouncer, da fonte PostgreSQL 17 host em `127.0.0.1:8745` para k3s, e somente upstreams `:3000` mudam no Apache;
3. contratos públicos health/models/non-stream/stream/Responses continuam válidos;
4. soak bloqueante dura pelo menos 30 minutos, com >=30 amostras e >=6 matrizes;
5. qualquer gate crítico executa rollback Apache para router Podman, PgBouncer para o PostgreSQL 17 host e smoke completo nessa topologia;
6. retirement trata router/Redis Podman e PostgreSQL 17 host separadamente, preserva tudo por >=7 dias e mantém edge/DB/CLIAnything no k3s.

## Infraestrutura e feedback

| Propriedade | Valor |
|---|---|
| Framework | Bash `set -euo pipefail`, fixtures/self-tests, Python HTTP contracts, kubectl, systemd, PgBouncer e Apache oficiais |
| Quick gate | `bash -n scripts/k3s-router-*.sh` e `--self-test` do script criado pela task |
| Gate live | modos `--verify-live`/`--verify-evidence`, consumindo diretório de evidência explícito e `SHA256SUMS` |
| CPU | comandos leves; qualquer dump/check pesado usa contenção do repo e nunca excede 20% da CPU total |
| Segredos | Vault→processo; token/DSN/senha/header nunca entram em stdout, Markdown ou evidência |
| Feedback máximo pré-live | menos de 60 segundos para syntax + fixtures |

## Wave 0 — scaffolds obrigatórios

Wave 0 acontece dentro da primeira task que cria cada script, antes de habilitar qualquer modo live. Nenhuma mutação de Apache, PgBouncer ou Podman é permitida até todos os self-tests da wave correspondente passarem.

| Wave 0 ID | Produzido por task | Scaffold/test | Gate que desbloqueia |
|---|---|---|---|
| W0-01 | 30-01-01/02 | fixtures de GO inválido, secret ausente, endpoint inválido, topologia/contagem 34/0 divergente, backup incompleto e checksum divergente | coleta live do envelope |
| W0-02 | 30-02-01 | fixtures PgBouncer com entrada ausente/duplicada, diff colateral e rollback isolado ao host PG17 | repoint DB live |
| W0-03 | 30-02-02 | fixtures Apache com target divergente, alteração em `:3003`, configtest/reload falho e rollback isolado | retarget Apache live |
| W0-04 | 30-03-01 | fixtures HTTP/JSON/SSE/Responses e classificação local/upstream | smoke público live |
| W0-05 | 30-03-02 | relógio fake, >=30/6, todos os gates críticos e ordem rollback Apache→PgBouncer host PG17→smoke router Podman | soak live de 30 minutos |
| W0-06 | 30-04-01 | fixtures soak inválido, allowlist Podman divergente, preserve-set divergente, PG17 com/sem dependências e auto-recuperação | retirement separado live |
| W0-07 | 30-04-02 | fixture de manifest JSON + SHA256SUMS para renderer/verifier de `30-OPERATION.md` | registro final |

## Mapeamento Nyquist por task

| Task ID | Wave | Requirements | Evidência automatizada | Prova live obrigatória | Status |
|---|---:|---|---|---|---|
| 30-01-01 | 1 | PHASE-22-CUTOVER-ROLLBACK | syntax + `--self-test` | GO/checksums/ClusterIPs/EndpointSlices/PV/Pod/node e topologia PG17 34/0 validados | pending |
| 30-01-02 | 1 | PHASE-22-CUTOVER-ROLLBACK | `--self-test-backup` | envelope 0700 e dump exclusivo da fonte host, configs/inventários e SHA256SUMS íntegros | pending |
| 30-01-03 | 1 | PHASE-22-CUTOVER-ROLLBACK | docs link check + fatos operacionais | nenhum; documentação deriva do contrato validado | pending |
| 30-02-01 | 2 | PHASE-22-CUTOVER-ROLLBACK | self-tests PgBouncer/rollback | diff/checksums allowlisted, demais DBs idênticas, SHOW DATABASES sanitizado, query nova via 6432 e rollback host PG17/34 tabelas | pending |
| 30-02-02 | 2 | PHASE-22-CUTOVER-ROLLBACK | self-tests Apache/rollback | configtest, target ClusterIP, região `:3003` idêntica e health/smoke | pending |
| 30-02-03 | 2 | PHASE-22-CUTOVER-ROLLBACK | `--verify-live` sobre manifest + SHA256SUMS | cutover serial completo ou rollback comprovado | pending |
| 30-03-01 | 3 | PHASE-20-GO-ONLY-V1-MODELS, PHASE-25-CLIENT-SMOKE-VALIDATION | fixtures dos contratos | matriz pública autenticada completa | pending |
| 30-03-02 | 3 | PHASE-22-CUTOVER-ROLLBACK, PHASE-25-CLIENT-SMOKE-VALIDATION | fixtures de relógio/gates/rollback | artifact JSON checksummed com duração >=1800s, >=30 amostras e >=6 matrizes | pending |
| 30-03-03 | 3 | todos da fase | `--verify-evidence` + docs links | PASS ou rollback Apache→PgBouncer host PG17→smoke router Podman | pending |
| 30-04-01 | 4 | PHASE-22-CUTOVER-ROLLBACK | fixtures retirement | router/Redis Podman ausentes; PG17 obrigatoriamente active/enabled com `KEEP_SERVICE`; preserve-set íntegro | pending |
| 30-04-02 | 4 | todos da fase | renderer + verifier estrutural/checksum | edge/DB k3s, smoke, CLIAnything e decisão PG17 aprovados | pending |
| 30-04-03 | 4 | todos da fase | docs links + facts gerados | runbooks coerentes com manifest final | pending |

## Contrato dos artefatos estruturados

Cada estágio live produz diretório `0700` com:

- `manifest.json`: `schema_version`, `stage`, `status`, `started_at`, `finished_at`, `duration_seconds`, targets não sensíveis, contagens, checksums e referências aos arquivos;
- `SHA256SUMS`: checksum de cada payload allowlisted, excluindo o próprio arquivo `SHA256SUMS` para evitar ciclo;
- JSON/JSONL sanitizado por domínio: configs diff, samples, matrices, retirement e preserve-set;
- nenhum body completo, token, cookie, senha, DSN credenciado, Secret YAML ou header Authorization.

Verificadores devem parsear JSON, validar schema/status/timestamps/contagens e executar `sha256sum -c`; `rg` em Markdown não constitui prova operacional.

## Gates bloqueantes

- Nenhuma mutação live sem Wave 0 verde e envelope READY.
- Nenhum Apache cutover antes da prova live PgBouncer.
- Nenhum PASS de soak com duração menor que 1800 segundos, menos de 30 amostras ou menos de 6 matrizes.
- Qualquer gate crítico exige rollback Apache para router Podman, PgBouncer para PostgreSQL 17 host em `127.0.0.1:8745`, prova de 34 tabelas e smoke público checksummed.
- Nenhum retirement sem soak PASS íntegro.
- Nenhuma conclusão sem router/Redis Podman disabled/inactive e ausentes, preserve-set íntegro, edge/DB k3s, smoke público e CLIAnything k3s aprovados, além de inventário completo e decisão segura para `postgresql@17-main`.
- Nunca aceitar a database homônima do container, com 0 tabelas, como fonte ou rollback; nunca apagar data dir, database, dumps, imagens, volumes ou units.
- Headroom não é criado, configurado ou alterado.

## Auditoria de cobertura das fontes

| SOURCE | ID | Feature/Requirement | Plan | Status | Notes |
|---|---|---|---|---|---|
| GO | — | Cutover público após GO da Phase 29, smoke, soak e rollback ativo | 30-01..30-04 | COVERED | O rollback separa router Podman de DB host PG17 conforme correção live. |
| REQ | PHASE-22-CUTOVER-ROLLBACK | Cutover e rollback testáveis | 30-01..30-04 | COVERED | Backups, trocas atômicas, rollback, soak e retirement. |
| REQ | PHASE-20-GO-ONLY-V1-MODELS | `/v1/models` permanece Go-owned | 30-03, 30-04 | COVERED | Smoke e verificação final preservam shape/campos públicos. |
| REQ | PHASE-25-CLIENT-SMOKE-VALIDATION | Smokes reais de cliente | 30-03, 30-04 | COVERED | Non-stream, stream, Responses, models e health. |
| RESEARCH | — | PgBouncer primeiro, Apache depois, com artifacts checksummed | 30-01, 30-02 | COVERED | Fonte e rollback DB são o PostgreSQL 17 host. |
| RESEARCH | — | Soak de 30 minutos, 30 amostras e 6 matrizes | 30-03 | COVERED | Gate objetivo e rollback automático. |
| RESEARCH | — | Retirement separado e retenção >=7 dias | 30-04 | COVERED | Podman router/Redis e backend PG17 possuem gates próprios. |
| CONTEXT | D-01 | Alterar somente Apache `:3000`; preservar `:3003` | 30-02 | COVERED | Diff allowlisted e configtest. |
| CONTEXT | D-02 | Usar ClusterIPs aprovados e registrar checksums | 30-01, 30-02 | COVERED | Preflight e manifest live. |
| CONTEXT | D-03 | Backups completos antes de mutação | 30-01 | COVERED | Dump exclusivo da fonte host PG17. |
| CONTEXT | D-04 | Smoke público completo e classificação de falhas | 30-03 | COVERED | Matriz autenticada fail-closed. |
| CONTEXT | D-05 | Soak periódico com critérios objetivos | 30-03 | COVERED | Sampling e gates estruturados. |
| CONTEXT | D-06 | Rollback imediato para router Podman + DB host PG17 | 30-02, 30-03 | COVERED | Ordem Apache→PgBouncer→smoke. |
| CONTEXT | D-07 | Retirement Podman e preservação obrigatória PG17 compartilhado | 30-01, 30-04 | COVERED | Somente `KEEP_SERVICE`; stop/disable é proibido. |
| CONTEXT | D-08 | Headroom fora do escopo | 30-01..30-04 | COVERED | Proibição explícita em todos os waves. |
| CONTEXT | D-09 | CPU <=20%, Pods 500m e segredos Vault | 30-01..30-04 | COVERED | Gates de preflight, execução e soak. |
| CONTEXT | D-10 | Fonte host 34 tabelas; container 0 e inelegível | 30-01..30-04 | COVERED | Topologia provada em preflight, rollback e retirement. |

Nenhum item obrigatório ficou sem plano; Headroom permanece exclusão explícita, não gap.

## Sign-off

- [x] Todos os tasks possuem `<automated>`.
- [x] Wave 0 mapeia cada scaffold faltante à task que o cria.
- [x] Gates live validam comportamento e evidência estruturada, não prosa.
- [x] Soak e retirement são autônomos, sem checkpoint humano.
- [x] CPU <=20%, segredos fail-closed e Headroom fora do escopo.

**Approval:** pending execution
