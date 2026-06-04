# Phase 7: feat/pt-native-branch - Context

**Gathered:** 2026-06-04
**Status:** Ready for planning
**Milestone:** v2.12 — pt-native upstream sync
**Source:** User intent (direct) + PR #5245 audit

<domain>
## Phase Boundary

Reabrir a submissão da tradução PT-BR para o upstream QuantumNous/new-api em um PR limpo, com escopo mínimo (idioma nativo, igual zh/en), sem inflar com código do fork Atius. Branch `feat/pt-native` baseado em `upstream/main` com apenas 5 arquivos modificados.

**Fora do escopo:** PR #2 (já mergeado em 2026-05-31), v2.11 rebrand, podman migration, model-detailed middleware, fork-specific infra.

</domain>

<decisions>
### Translation Scope (LOCKED)

- **Adicionar PT como idioma nativo, mesmo padrão de zh/en:**
  - 1 linha em `INTERFACE_LANGUAGE_OPTIONS` (languages.ts): `{ code: 'pt', label: 'Português' }`
  - 1 import + 1 entry em `resources` + 1 entry em `supportedLngs` (config.ts)
  - 1 const `LangPt = "pt"` + 1 entry em `localizers` + 1 case em `normalizeLang` + 1 entry em `SupportedLanguages()` (i18n.go)
  - 1 arquivo `i18n/locales/pt.yaml` (227 chaves, 100% de en.yaml)
  - 1 arquivo `web/default/src/i18n/locales/pt.json` (3910 chaves, 100% de en.json)

### Naming & Branching (LOCKED)

- **Branch name:** `feat/pt-native` (sem "i18n" no nome, igual upstream pratica)
- **Base:** `upstream/main` (QuantumNous/new-api main), NÃO `main` do fork
- **Título do PR:** `feat: add Brazilian Portuguese (pt) language`
- **Commit message:** `feat: add Portuguese (pt) language` (clean, sem prefixo GSD)

### Code Cleanliness (LOCKED)

- **Sem menção a "i18n" no código:** PT é idioma, não "i18n feature". Comentários em Go/TS descrevem "Portuguese" ou "pt", não "i18n".
- **Sem comportamento case-insensitive refactor:** upstream não tem, não introduzir no PR
- **Sem docs internas do fork** (TRANSLATION-PT-BR.md) — fica no fork only
- **Sem testes:** upstream não tem pra zh/en, paridade manda
- **Sem vitest/setup.ts/package.json deps:** upstream não tem, paridade manda
- **Sem login.tsx/routeTree.gen.ts/docker-compose rebrand/PODMAN/model-detailed:** tudo fork-specific

### Workflow (LOCKED)

- **Cherry-pick manual:** não `git cherry-pick` (conflitos), mas `cp` dos 5 arquivos com base `upstream/main`
- **Validação Go build:** `go build ./...` deve passar no diretório do fork
- **Validação frontend:** `bun install && bun run typecheck && bun run build` em web/default/
- **PR workflow:** push branch + fechar #5245 com comentário + abrir PR novo

### Claude's Discretion

- Estrutura exata dos commits (1 squash vs 5 separados) — recomendação: 1 squash limpo
- Conteúdo do comentário de fechamento do PR #5245 — referência ao novo PR + explicação do rebase
- Validação de chaves 100% cobertura — pode usar `bun run i18n:sync` se disponível, ou checagem manual

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Upstream patterns (source-of-truth for "add a language")

- `https://raw.githubusercontent.com/QuantumNous/new-api/main/i18n/i18n.go` — canonical Go pattern (zh-TW as reference)
- `https://raw.githubusercontent.com/QuantumNous/new-api/main/i18n/locales/en.yaml` — canonical YAML structure
- `https://raw.githubusercontent.com/QuantumNous/new-api/main/web/default/src/i18n/config.ts` — canonical frontend config
- `https://raw.githubusercontent.com/QuantumNous/new-api/main/web/default/src/i18n/languages.ts` — canonical language list
- `https://github.com/QuantumNous/new-api/pull/5245.diff` — current polluted PR (to identify and exclude contaminated files)

### Local reference (the source PT translations)

- `i18n/locales/pt.yaml` (current fork main, 227 chaves)
- `web/default/src/i18n/locales/pt.json` (current fork main, 3910 chaves)
- `.planning/STATE.md` — confirms v1.6 PT-BR closed and prod-validated

### Project rules

- `AGENTS.md` Rule 5 (protected project info: QuantumNous/new-api references must not be removed)
- `AGENTS.md` Rule 1 (JSON package: use common/json.go, not encoding/json directly)

</canonical_refs>

<specifics>
## Specific Ideas

- `INTERFACE_LANGUAGE_OPTIONS` no upstream tem 5 entries (en, zh, fr, ru, ja, vi). Adicionar pt mantém ordem alfabética no array se possível, ou ordem cronológica (último adicionado = final). Recomendação: ordem cronológica (final do array) — mais simples, não muda diffs antigos.
- `normalizeLang` no upstream Go: ordem dos cases importa? Não — são HasPrefix, qualquer ordem funciona. Adicionar `pt` em qualquer posição.
- `i18n.NewBundle(language.Chinese)` — bundle base é Chinese, não English. Não mexe nisso.
- `load: 'languageOnly'` em config.ts — converte `pt-BR` → `pt` automaticamente. Não mexer.
- Coverage check: 100% das chaves de en.yaml/en.json. Já validado em v1.6 (STATE.md).

</specifics>

<deferred>
## Deferred Ideas

- `docs/TRANSLATION-PT-BR.md` — fica no fork only, não vai pro upstream
- Testes de PT (vitest) — fica no fork only, paridade upstream manda
- `__tests__/pt-fallback.test.ts`, `__tests__/normalize-interface-language.test.ts` — fork only
- `vitest.config.ts` + `setup.ts` + testing-library deps — fork only
- `login.tsx` + `routeTree.gen.ts` — fork only (feature de login, não i18n)
- `docker-compose.yml` rebrand — fork only (não vai upstream nunca)

</deferred>

---
*Phase: v2.12.1-feat-pt-native-branch*
*Context gathered: 2026-06-04 via direct user request + PR #5245 audit*
