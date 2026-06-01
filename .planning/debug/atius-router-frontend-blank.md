---
name: atius-router-frontend-blank
slug: atius-router-frontend-blank
status: fixing
trigger: "Atius Router dashboard returns '{}' (blank page) — frontend not loading, only /docs/ works"
trigger_source: user-reported
created: 2026-05-31
updated: 2026-05-31T04:02:08-03:00
symptoms:
  expected: "Frontend React app renders at https://router.atius.com.br/ with login page or dashboard"
  actual: "Browser receives HTML with only body content '{}' — blank page, no React mount"
  error_messages:
    - "curl / → '{}' (3 bytes HTML)"
    - "curl /dashboard → '{}'"
    - "curl /sign-in → '{}'"
    - "curl /docs/ → 427KB Scalar API reference (WORKS via Apache proxy to port 3399)"
  timeline:
    - "Image built: 2026-05-28 06:01 (2 days ago)"
    - "Last known working frontend: unknown (likely before May 12)"
    - "Version local: 1.0.0-rc.2.1"
    - "X-New-Api-Version header: v0.0.0 (ldflags NOT applied)"
  reproduction:
    - "curl http://127.0.0.1:3301/ → '{}'"
    - "python playwright → blank page screenshot"
    - "Playwright page.content() → '<html><head></head><body>{}</body></html>'"
hypothesis: "go:embed web/default/dist in the GHCR binary embedded an EMPTY directory — bun run build failed silently in GH Actions, or working directory mismatch caused go build to embed nothing"
next_action: "delegate gsd-debugger to investigate GH Actions build logs and docker-build workflow"
reasoning_checkpoint: ""
files_changed: []
evidence:
  - timestamp: 2026-05-31T05:24Z
    type: curl
    finding: "curl http://127.0.0.1:3301/ → '{}' (3 bytes)"
  - timestamp: 2026-05-31T05:25Z
    type: curl
    finding: "curl http://127.0.0.1:3301/docs/ → 427KB (WORKS — Apache proxy to model-detailed on port 3399)"
  - timestamp: 2026-05-31T05:26Z
    type: header
    finding: "X-New-Api-Version: v0.0.0 (ldflags version NOT applied to binary)"
  - timestamp: 2026-05-31T05:27Z
    type: filesystem
    finding: "Container /new-api is the binary; /app does not exist; web/default/dist exists LOCALLY but not in container"
  - timestamp: 2026-05-31T05:28Z
    type: image_inspection
    finding: "ghcr.io/giovannimnz/router-ai-atius:latest created 2026-05-28, derived from debian:bookworm-slim"
  - timestamp: 2026-05-31T05:29Z
    type: workflow_history
    finding: "GH Actions Release workflow failing since May 13 (sync.yml failures) — tag conflict with upstream"
  - timestamp: 2026-05-31T05:30Z
    type: local_build_check
    finding: "docker build locally fails: bun install → Illegal instruction (core dumped) — CPU arch mismatch"
  - timestamp: 2026-05-31T03:49:09-03:00
    type: knowledge_base
    finding: "knowledge-base.md not found (no known-pattern candidates available)"
  - timestamp: 2026-05-31T03:51:21-03:00
    type: filesystem
    finding: "web/default/dist/index.html e web/classic/dist/index.html no repo contém apenas '{}' (placeholder); dist default só tem assets estáticos (logo/scalar), sem bundle do frontend"
  - timestamp: 2026-05-31T03:55:31-03:00
    type: docker-build
    finding: "docker build --target builder completou; /build/dist contém index.html (~943 bytes) + assets (favicon/static), indicando bun build gera dist válido no builder stage"
  - timestamp: 2026-05-31T03:57:39-03:00
    type: docker-build
    finding: "docker build --target builder2 completou; /build/web/default/dist contém index.html (~943 bytes) e assets, confirmando COPY do dist funciona antes do go build"
  - timestamp: 2026-05-31T03:58:06-03:00
    type: git
    finding: "Dockerfile no commit a42b39760 (Apr 28) já contém bun build + COPY do dist; sem diferença relevante vs HEAD"
  - timestamp: 2026-05-31T03:59:32-03:00
    type: runtime
    finding: "Executar /build/new-api via docker falha por falta de DB (hostname db-newapi não resolve), então não foi possível verificar HTTP / localmente"
  - timestamp: 2026-05-31T04:00:13-03:00
    type: git
    finding: "Tags locais incluem versões rc (v1.0.0-rc.2.1 etc); nenhuma evidência imediata de tag duplicada sem comparar com upstream"
  - timestamp: 2026-05-31T04:01:51-03:00
    type: workflow
    finding: "docker-publish.yml determina TAG via git describe ANTES do checkout; em workflow_run isso roda fora de um repo e gera TAG vazia → exit 1 ('No tag found')"
eliminated: []
root_cause: "docker-publish.yml calcula TAG via `git describe` antes do checkout em workflow_run; isso resulta em TAG vazia e aborta o build, deixando a GHCR sem nova imagem (a UI segue com dist placeholder e responde '{}')."
fix: "Reordenar docker-publish.yml para fazer checkout antes do step 'Determine tag', permitindo `git describe` rodar com tags disponíveis."
verification: "Pendente: executar workflow docker-publish (workflow_run ou manual) e validar que `/` retorna HTML completo em vez de '{}'."
files_changed:
  - ".github/workflows/docker-publish.yml"
tags:
  - docker
  - go-embed
  - frontend
  - github-actions
  - build
severity: critical
impact: "Frontend completely broken — users cannot login or access dashboard"
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "docker-publish.yml falha em workflow_run porque o step que usa `git describe` roda antes do checkout, retornando TAG vazia e abortando o build; sem build, a imagem GHCR permanece antiga com dist placeholder e a UI fica em branco."
  confirming_evidence:
    - "docker-publish.yml mostra 'Determine tag' antes de 'Checkout' e usa `git describe` no branch workflow_run."
    - "Sem repo checado, `git describe` retorna vazio → o script faz `exit 1` ('No tag found')."
  falsification_test: "Após mover o checkout antes do `git describe`, um workflow_run deve concluir o build da imagem; se a UI continuar retornando '{}' mesmo com imagem nova, a hipótese é falsa."
  fix_rationale: "Reordenar os steps garante que `git describe` rode dentro de um repo com tags, evitando TAG vazia e permitindo o build/redeploy da imagem."
  blind_spots: "Não tenho acesso aos logs do GH Actions nem posso verificar o deploy real; preciso de confirmação do usuário após rebuild."

---

## Investigation Plan (delegate to gsd-debugger)

1. **GH Actions logs:** Buscar último workflow run bem-sucedido do docker-build.yml (antes de May 28)
2. **Verificar bun install crash:** GH Actions usa `oven-sh/setup-bun` — arch detection pode falhar em ARM64 runners
3. **Dockerfile path:** Verificar se `COPY --from=builder /build/dist ./web/default/dist` está no Dockerfile correto
4. **Embed verification:** Extrair o binary do container e verificar strings relacionadas ao frontend
5. **Alternative fix:** Se GH Actions bun sempre falha, considerar buildar frontend localmente e copiar para image

---

## Resolution Log

*(to be filled after root cause confirmed)*
