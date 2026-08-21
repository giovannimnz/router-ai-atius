---
phase: 26
phase_name: codex-dynamic-discovery-and-curated-catalog
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 3, lessons: 3, patterns: 2, surprises: 2 }
missing_artifacts: ["26-VERIFICATION.md", "26-UAT.md"]
---

# Phase 26 Learnings: Codex dynamic discovery

## Decisions

### Request path usa catalogo Go local
Discovery upstream roda assíncrona e nunca vira dependência direta de `/v1/models` ou do relay.

**Source:** 26-01-SUMMARY.md

### Discovery nao promove modelo sem probe real
Slugs novos entram como candidates; somente resposta upstream válida permite promoção e sync de channel.

**Source:** 26-01-SUMMARY.md

### Default e ordering continuam política do fork
O upstream pode anunciar modelos, mas não decide automaticamente o default público do Atius Router.

**Source:** 26-01-SUMMARY.md

## Lessons

### Discovery, metadata e promoção são estados diferentes
Persistir snapshots e candidates permite auditar o que upstream anunciou sem expor imediatamente algo não validado.

**Source:** 26-01-SUMMARY.md

### Metadata pública precisa manter o contrato sanitizado
Context window pode ser enriquecido, mas campos internos de pricing/provenance continuam fora do payload público.

**Source:** 26-01-SUMMARY.md

### O wrapper de CPU precisa do toolchain real no PATH
Wrappers globais aninhados de `go`/`gcc` quebraram cwd, cgo e caches; o padrão estável foi `podman-admin profile-run` mais PATH e GOCACHE explícitos.

**Source:** 26-01-SUMMARY.md

## Patterns

### Discover, merge, probe, promote, publish
Usar pipeline persistente em cinco estágios para qualquer catálogo upstream instável.

**Source:** 26-01-SUMMARY.md

### Scheduler fora do request path
Executar sync diário e servir sempre do estado local conhecido reduz latência e blast radius upstream.

**Source:** 26-01-SUMMARY.md

## Surprises

### A primeira implementação criou import cycle
A integração entre model/service/controller precisou ser redesenhada antes dos testes passarem.

**Source:** 26-01-SUMMARY.md

### Slug dinâmico nao significa suporte completo a API nova
A pesquisa GPT-5.6 mostrou que discovery pode encontrar `gpt-5.6-*`, enquanto DTOs e conversões ainda não suportam todos os novos campos.

**Source:** docs/OPENAI-GPT-5.6-CODEX-RESEARCH-2026-07-10.md
