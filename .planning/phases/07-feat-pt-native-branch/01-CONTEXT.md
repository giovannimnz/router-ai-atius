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

### Working Tree Strategy (LOCKED — discussed 2026-06-04)

- **Pre-flight:** `git stash push -u` antes de criar o branch, pra limpar `podman-compose.yml` modified e `integration/docs/` untracked. Aplicar `git stash pop` DEPOIS de validar Phase 7, antes de Phase 8.
- **Razão:** o working tree sujo pode poluir `git status` durante a validação. O stash preserva o trabalho do fork e isola Phase 7.
- **Exceção:** se o stash conflitar com feat/pt-native working tree, abortar e pedir guidance.

### Conflict Resolution Strategy (LOCKED — discussed 2026-06-04)

- **Método:** para cada arquivo, usar `git show main:<file> > <file>` (do fork, fonte da tradução) OU `git show upstream/main:<file> > <file>` (do upstream, baseline).
- **i18n.go:** patchar com `patch` tool — não `cp`. O upstream `i18n.go` é base, os 4 hunks vão aplicados por cima. Se o patch falhar (rejeição por offset), abrir o arquivo e editar manualmente linha por linha.
- **i18n/locales/pt.yaml + pt.json:** `git show main:web/default/src/i18n/locales/pt.json > web/default/src/i18n/locales/pt.json` — copy pura, sem patch.
- **config.ts + languages.ts:** patch com `patch` tool usando hunks conhecidos (já especificados no PLAN.md).

### Coverage Validation (LOCKED — discussed 2026-06-04)

- **Método:** `jq '.translation | keys | length' pt.json` deve igualar mesmo comando em `en.json` (3910 chaves).
- **Mesma checagem em pt.yaml:** `python3 -c "import yaml; d=yaml.safe_load(open('pt.yaml')); print(len(d))"` deve igualar en.yaml.
- **Sem tooling upstream:** `bun run i18n:sync` não é usado (script interno do fork, não confiável cross-fork).

### Commit Strategy (LOCKED — discussed 2026-06-04)

- **1 squash final:** 5 arquivos em 1 commit único. Mensagem: `feat: add Portuguese (pt) language`.
- **Razão:** PR mais limpo, atomic change, mais fácil de revisar upstream.
- **Implementação:** `git add i18n/ web/default/src/i18n/ && git commit -m "feat: add Portuguese (pt) language"` (1 só commit, no final da Phase 7).
- **Nota:** Phase 7 = "ready to commit" (working tree). Phase 8 = "commit + push + PR". Pra Phase 7 autônoma, NÃO commitar — usuário decide quando commitar.

### Claude's Discretion

- Estrutura exata do commit message (sufixo body, refs a PR #2) — recomendado: minimal, sem body
- Conteúdo do comentário de fechamento do PR #5245 — Phase 8, não Phase 7
- Ordem de aplicação dos hunks em i18n.go — bottom-up pra evitar offset shifts

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
