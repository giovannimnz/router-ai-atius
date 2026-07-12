---
phase: 32
phase_name: "codex-oauth-lifecycle-and-upstream-auth-diagnostics"
project: "router-ai-atius"
generated: "2026-07-12"
counts:
  decisions: 4
  lessons: 4
  patterns: 4
  surprises: 4
missing_artifacts:
  - "32-UAT.md (validacao live registrada em 32-VERIFICATION.md e 32-04-SUMMARY.md)"
---

# Phase 32 Learnings: codex-oauth-lifecycle-and-upstream-auth-diagnostics

## Decisions

### Separar auth interna de auth upstream
Erros da API key do Router permanecem internos; falhas OAuth Codex usam codigos, mensagens e metadata upstream sanitizados.

**Rationale:** Permite diagnostico rapido sem atribuir `token_invalidated` ao consumidor errado.
**Source:** 32-01-SUMMARY.md

### Tratar Codex como upstream sempre streaming
O relay type 57 envia `stream=true` e `Accept: text/event-stream`; clientes non-stream recebem resposta bufferizada.

**Rationale:** O endpoint Codex rejeita requests non-stream, embora o Router precise preservar ambos os contratos publicos.
**Source:** 32-VERIFICATION.md

### Reconciliar discovery com candidatos oficiais curados
O catalogo avalia tanto os slugs dinamicos quanto a lista oficial curada e promove somente quem passa no probe live.

**Rationale:** O endpoint `/models` do tenant omitiu modelos que o endpoint `/responses` aceitou diretamente.
**Source:** 32-04-PARTIAL.md

### Nao fechar sem refresh token Router-owned
A fase so foi fechada depois que a credencial live deixou de ser access-token-only e o refresh manual passou.

**Rationale:** Expiracao futura nao substitui renovacao automatica nem prova a conclusao do fluxo OAuth definitivo.
**Source:** 32-04-SUMMARY.md

## Lessons

### SSE pode omitir Content-Type
O upstream retornou linhas `event:`/`data:` validas sem header de media type.

**Context:** A tentativa de decodificar o corpo inteiro como JSON causava `invalid character 'e'`.
**Source:** 32-04-PARTIAL.md

### Input string nao e aceito pelo Codex upstream
O contrato publico Responses aceita string, mas o backend Codex exige uma lista de mensagens/content parts.

**Context:** O adaptor precisa normalizar string para lista sem alterar a API publica.
**Source:** 32-01-SUMMARY.md

### Snapshot precisa versionar o validador
Assinar apenas modelos e policy reutiliza rejeicoes antigas quando a semantica do gate muda.

**Context:** Sol continuou rejeitado apos aceitar `Ok.` ate a versao do contrato entrar na assinatura.
**Source:** 32-VERIFICATION.md

### Expiracao local futura pode coexistir com regeneracao obrigatoria
O hotfix tinha expiracao futura e probe OK, mas continuava corretamente marcado para regeneracao por falta de refresh token.

**Context:** Health deve combinar campos locais com capacidade real de renovacao e probe upstream.
**Source:** 32-04-SUMMARY.md

## Patterns

### Streaming upstream, contrato cliente preservado
Force streaming apenas no request upstream e escolha forwarding ou buffering conforme o request original do cliente.

**When to use:** Providers que oferecem somente SSE, mas precisam permanecer OpenAI-compatible para clientes stream e non-stream.
**Source:** 32-01-SUMMARY.md

### Curated candidates plus live promotion
Una discovery e candidatos oficiais, aplique denylist local e use probe deterministico antes de criar abilities.

**When to use:** Rollouts de modelos onde discovery e disponibilidade efetiva divergem por tenant.
**Source:** 32-04-SUMMARY.md

### Build limitado com caches persistentes
Use wrapper `cpus=0.8`, compilador serial e mounts persistentes de `GOMODCACHE`/`GOCACHE`.

**When to use:** Builds Go em host de 4 vCPU com teto total de 20% e camadas Podman que nao preservam cache suficiente.
**Source:** 32-04-SUMMARY.md

### Credencial administrativa efemera com cleanup garantido
Crie token aleatorio apenas para a chamada administrativa, execute em `try/finally` e valide no banco que foi apagado.

**When to use:** Probes live administrativos quando a sessao de navegador nao esta disponivel.
**Source:** 32-04-SUMMARY.md

## Surprises

### Discovery omitiu modelos funcionais
`gpt-5.5`, `gpt-5.6-sol` e `gpt-5.6-terra` responderam 200 diretamente mesmo ausentes no discovery dinamico.

**Impact:** O catalogo puramente autoritativo por discovery removeria modelos validos.
**Source:** 32-04-PARTIAL.md

### Luna nao estava liberada neste tenant
`gpt-5.6-luna` retornou 404 enquanto Sol e Terra funcionaram.

**Impact:** Metadata oficial pode ser preservada, mas Luna nao deve ser exposta ate passar no probe.
**Source:** 32-VERIFICATION.md

### Resposta de validacao variou apenas na pontuacao
Sol respondeu `Ok.` quando o gate esperava `Ok`.

**Impact:** Comparacao textual estrita gerou falso negativo e exigiu normalizacao terminal minima.
**Source:** 32-VERIFICATION.md

### O ultimo gate exigiu handoff humano, mas tornou-se verificavel pela API
O callback precisou do browser autenticado; depois do handoff, metadata, probe e refresh comprovaram a credencial Router-owned sem expor segredos.

**Impact:** O fechamento deve esperar o callback real, mas toda a verificacao posterior pode ser automatizada com metadata sanitizada.
**Source:** 32-04-SUMMARY.md

### CLI local e runtime podem apontar para bancos homonimos diferentes
O `clianything` host default consultou um PostgreSQL local, enquanto o runtime
usava `10.11.1.11:6432/DBRouterAiAtius`.

**Impact:** Validacao operacional precisa conferir host, porta e database do
`SQL_DSN` live antes de qualquer leitura ou escrita administrativa.
**Source:** 32-04-SUMMARY.md

### Campos ocultos nao bastam se efeitos globais continuam ativos
O type 57 ocultava Base URL, mas o efeito global de warning ainda observava um
valor legado terminado em `/v1`.

**Impact:** Boundaries de UI especificas por provider devem governar tambem
efeitos, validacoes e toasts, nao apenas markup visivel.
**Source:** 32-04-SUMMARY.md
