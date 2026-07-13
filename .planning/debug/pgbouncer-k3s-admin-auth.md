---
name: pgbouncer-k3s-admin-auth
slug: pgbouncer-k3s-admin-auth
status: resolved_live_verified
trigger: "[$gsd-debug] consulte docs e o q mudou e resolva"
trigger_source: user-reported
created: 2026-07-13
updated: 2026-07-13T16:35:00-03:00
symptoms:
  expected: "O cutover da Phase 30 deve repointar apenas DBRouterAiAtius no PgBouncer de 127.0.0.1:8745 para o PostgreSQL k3s 10.43.179.157:5432, mantendo autenticacao valida do usuario admin e liberando a etapa Apache."
  actual: "A linha de DBRouterAiAtius no pgbouncer.ini muda para o ClusterIP k3s, mas novas conexoes via 127.0.0.1:6432 falham com password authentication failed for user admin; o rollback para 127.0.0.1:8745 ja foi comprovado."
  error_messages:
    - "server login failed: FATAL password authentication failed for user \"admin\""
    - "password authentication failed for user \"admin\""
    - "Connection matched file \"/var/lib/postgresql/data/pg_hba.conf\" line 128: \"host all all all scram-sha-256\""
  timeline:
    - "Phase 29 terminou com shadow/apply/smoke verdes e no-go apenas por live-stability."
    - "Phase 30 preflight live gerou manifest READY_WITH_PHASE29_OVERRIDE em 2026-07-13."
    - "Primeiro cutover live de PgBouncer falhou por reload sem permissao de leitura do pgbouncer.ini."
    - "Depois de alinhar owner/mode e recarregar, o backend k3s passou a falhar em auth do usuario admin."
  reproduction:
    - "Executar PHASE30_EXECUTE=1 ./scripts/podman-admin.sh profile-run -- ./scripts/k3s-router-cutover.sh --live --stage pgbouncer --evidence-dir /home/ubuntu/.local/state/router-ai-atius/phase30/run-20260713T160941Z"
hypothesis: "O role admin restaurado no PostgreSQL k3s nao aceita o mesmo segredo SCRAM/fluxo que o PgBouncer usa para autenticar no backend, mesmo com o cliente Vault ainda autenticando no host e diretamente no k3s."
next_action: "Seguir para a etapa Apache da Phase 30; o blocker de autenticacao PgBouncer -> PostgreSQL k3s foi resolvido."
reasoning_checkpoint: "Root cause confirmada por docs oficiais + hashes sanitizados + replay live do cutover do PgBouncer."
files_changed:
  - scripts/k3s-router-cutover-preflight.sh
  - scripts/k3s-router-cutover.sh
  - scripts/k3s-router-rollback.sh
  - docs/K3S-MIGRATION.md
  - .planning/phases/30-k3s-public-cutover-and-rollback-soak/30-01-SUMMARY.md
evidence:
  - timestamp: 2026-07-13T16:09:48-03:00
    type: runtime
    finding: "manifest.json da Phase 30 ficou READY_WITH_PHASE29_OVERRIDE, com host PG17 em 127.0.0.1:8745 (34 tabelas), k3s PG em 10.43.179.157:5432 (35 tabelas) e Podman postgres com DBRouterAiAtius vazio (0 tabelas)."
  - timestamp: 2026-07-13T16:15:28-03:00
    type: journal
    finding: "PgBouncer recebeu HUP mas falhou em reler /etc/pgbouncer/pgbouncer.ini por Permission denied; a configuracao nova nao entrou em vigor."
  - timestamp: 2026-07-13T16:17:46-03:00
    type: journal
    finding: "Apos alinhar owner/mode e recarregar, o PgBouncer abriu conexao server para 10.43.179.157:5432 e recebeu password authentication failed for user admin."
  - timestamp: 2026-07-13T16:18:00-03:00
    type: runtime
    finding: "Rollback de PgBouncer restaurou DBRouterAiAtius = host=127.0.0.1 port=8745 dbname=DBRouterAiAtius e 127.0.0.1:6432 voltou a responder 34 tabelas."
  - timestamp: 2026-07-13T16:28:00-03:00
    type: docs
    finding: "PgBouncer config/auth docs exigem que segredos SCRAM no auth_file e no backend sejam identicos; apenas a mesma senha em texto nao basta. PostgreSQL CREATE/ALTER ROLE armazena SCRAM apresentado como-is, permitindo recarregar o segredo exato."
  - timestamp: 2026-07-13T16:31:00-03:00
    type: sanitized-hash
    finding: "O hash sanitizado do SCRAM do admin no PgBouncer/userlist e no host coincidia, mas o do PostgreSQL k3s divergia."
  - timestamp: 2026-07-13T16:34:22-03:00
    type: runtime
    finding: "Depois de sincronizar o SCRAM exato do host para o admin no k3s e repetir o repoint, o PgBouncer passou a apontar para 10.43.179.157:5432 e 127.0.0.1:6432 passou a responder 35 tabelas."
eliminated:
  - hypothesis: "O problema era apenas owner/mode do /etc/pgbouncer/pgbouncer.ini."
    why_not: "Corrigir permissao permitiu o reload, mas o backend k3s continuou falhando com password authentication failed para admin."
  - hypothesis: "A senha do Vault estava errada para o k3s."
    why_not: "Conexao direta ao PostgreSQL k3s com admin + POSTGRES_PASSWORD funcionava; o drift estava no segredo SCRAM armazenado do role admin."
root_cause: "A restauracao Phase 29 recriava o password do role admin no PostgreSQL k3s a partir da senha em texto do Vault. Como PostgreSQL com scram-sha-256 gera um novo segredo SCRAM com novo salt/keys, o rolpassword do k3s ficou diferente do SCRAM do admin no userlist do PgBouncer, embora a senha em texto fosse a mesma. A documentacao oficial do PgBouncer exige segredo SCRAM identico entre auth_file e backend para login server-side via SCRAM."
fix: "O cutover da Phase 30 agora sincroniza o SCRAM exato do admin do host para o PostgreSQL k3s antes do repoint do PgBouncer e valida o backend via 127.0.0.1:6432. A restauracao Phase 29 tambem passou a reaplicar o SCRAM exato do host, em vez de regenerar um segredo novo a partir da senha em texto."
verification: "Docs oficiais consultadas; hashes sanitizados confirmaram drift host/userlist vs k3s; self-tests do cutover continuaram PASS; cutover live do PgBouncer ficou verde; pgbouncer.ini live aponta DBRouterAiAtius para 10.43.179.157:5432; consulta via 127.0.0.1:6432 retorna 35 tabelas."
tags:
  - pgbouncer
  - postgresql
  - k3s
  - phase-30
severity: high
impact: "Sem resolver a autenticacao do admin no backend k3s, a Phase 30 nao pode passar da etapa PgBouncer e o Apache nao deve ser movido."
---

## Current Focus

hypothesis: "Resolvida: o drift estava no segredo SCRAM do role admin entre host/userlist e PostgreSQL k3s."
test: "Sincronizar o SCRAM exato do host para o k3s, repetir o repoint e validar via 127.0.0.1:6432."
expecting: "O PgBouncer deve autenticar no backend k3s e expor 35 tabelas via DBRouterAiAtius."
next_action: "Avancar para a etapa Apache da Phase 30."

## Evidence

## Eliminated

## Resolution
