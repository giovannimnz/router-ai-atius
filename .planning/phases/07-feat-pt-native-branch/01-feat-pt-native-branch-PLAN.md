---
phase: 7
name: feat-pt-native-branch
wave: 1
depends_on: []
files_modified:
  - i18n/i18n.go
  - i18n/locales/pt.yaml
  - web/default/src/i18n/config.ts
  - web/default/src/i18n/languages.ts
  - web/default/src/i18n/locales/pt.json
autonomous: false
---

# Phase 7: feat/pt-native branch — minimal upstream PR

## Objective

Criar branch `feat/pt-native` no fork `giovannimnz/router-ai-atius` baseado em `upstream/main`, contendo APENAS os 5 arquivos nativos de tradução PT (mesmo padrão de zh/en), sem nenhuma contaminação do fork Atius. Push do branch não incluso nesta fase (Phase 2).

## Tasks

### Task 01: Fetch upstream/main e criar branch base

<read_first>
- `https://raw.githubusercontent.com/QuantumNous/new-api/main/i18n/i18n.go` (canonical Go pattern)
- `.planning/phases/v2.12-pt-native-upstream-sync/01-CONTEXT.md` (decisions)
</read_first>

<acceptance_criteria>
- `git fetch upstream main` exits 0
- `git rev-parse upstream/main` returns a valid SHA matching QuantumNous/new-api HEAD
- `git branch feat/pt-native upstream/main` creates the branch
- `git log feat/pt-native --oneline -1` shows the upstream commit (NOT the fork's main)
- Working tree on feat/pt-native has no uncommitted changes from fork's main
</acceptance_criteria>

<action>
Run in /home/ubuntu/docker/Atius/router-ai-atius:
1. `git fetch upstream main` — atualiza ref `upstream/main`
2. Verify SHA: `git rev-parse upstream/main` should match `curl -s https://api.github.com/repos/QuantumNous/new-api/commits/main | jq -r .sha`
3. `git branch feat/pt-native upstream/main` — cria branch novo baseado no upstream, NÃO no main do fork
4. `git checkout feat/pt-native`
5. `git status` deve estar clean (sem modificações herdadas do main do fork)

Se feat/pt-native já existir localmente, error: usuário precisa decidir (delete ou checkout).
</action>

### Task 02: Copy pt.yaml para i18n/locales/

<read_first>
- File: `i18n/locales/en.yaml` (target file to read first for structure parity)
- File: Source: `i18n/locales/pt.yaml` from current branch `main` (the 227-key validated translation)
</read_first>

<acceptance_criteria>
- File `i18n/locales/pt.yaml` exists on feat/pt-native branch
- File has exactly the same line count and key count as the source from main branch
- `head -1 i18n/locales/pt.yaml` is NOT a GSD-2 commit hash or fork-specific comment
- File structure follows the same indentation as `en.yaml` (2-space YAML)
</acceptance_criteria>

<action>
Run in /home/ubuntu/docker/Atius/router-ai-atius on branch feat/pt-native:
1. `git show main:i18n/locales/pt.yaml > i18n/locales/pt.yaml` — copy from fork's main to working tree
2. `git diff --stat i18n/locales/pt.yaml` should show new file (no prior version in upstream)
3. Verify file count: `grep -c '^\w' i18n/locales/pt.yaml` should match the 227 key count from STATE.md
4. Diff against en.yaml structure: `head -30 i18n/locales/pt.yaml` and `head -30 i18n/locales/en.yaml` should have parallel key ordering
</action>

### Task 03: Copy pt.json para web/default/src/i18n/locales/

<read_first>
- File: `web/default/src/i18n/locales/en.json` (target file for structure parity)
- File: Source: `web/default/src/i18n/locales/pt.json` from current branch `main` (3910-key validated translation)
</read_first>

<acceptance_criteria>
- File `web/default/src/i18n/locales/pt.json` exists on feat/pt-native branch
- File parses as valid JSON (`jq empty web/default/src/i18n/locales/pt.json` exits 0)
- File top-level has `translation` namespace key
- Key count matches source: `jq '.translation | keys | length' web/default/src/i18n/locales/pt.json` equals `jq '.translation | keys | length' web/default/src/i18n/locales/en.json`
</acceptance_criteria>

<action>
Run in /home/ubuntu/docker/Atius/router-ai-atius on branch feat/pt-native:
1. `git show main:web/default/src/i18n/locales/pt.json > web/default/src/i18n/locales/pt.json` — copy from fork's main
2. `jq empty web/default/src/i18n/locales/pt.json` — validate JSON
3. `jq '.translation | keys | length' web/default/src/i18n/locales/pt.json` should equal same command on en.json (3910 keys per STATE.md)
4. `git diff --stat web/default/src/i18n/locales/pt.json` should show new file
</action>

### Task 04: Patch i18n/i18n.go — adicionar LangPt

<read_first>
- File: `i18n/i18n.go` (target file, current state on feat/pt-native)
- File: Upstream canonical: `https://raw.githubusercontent.com/QuantumNous/new-api/main/i18n/i18n.go`
</read_first>

<acceptance_criteria>
- File `i18n/i18n.go` has `LangPt = "pt"` constant added
- `Init()` function's `files` slice includes `"locales/pt.yaml"`
- `Init()` function pre-creates `localizers[LangPt] = i18n.NewLocalizer(bundle, LangPt)`
- `normalizeLang()` function has `case strings.HasPrefix(lang, "pt"): return LangPt`
- `SupportedLanguages()` function's return slice includes `LangPt` (at end)
- No "i18n" mentioned in comments related to PT addition (use "Portuguese" or "pt" instead)
- `go build ./i18n/...` exits 0
</acceptance_criteria>

<action>
On feat/pt-native, edit `i18n/i18n.go` with these 4 hunks:

1. In the `const (` block (after `LangEn = "en"`), add:
   ```go
   LangPt = "pt"
   ```
   No trailing comment about "i18n" — keep minimal.

2. In `Init()` function, change:
   ```go
   files := []string{"locales/zh-CN.yaml", "locales/zh-TW.yaml", "locales/en.yaml"}
   ```
   to:
   ```go
   files := []string{"locales/zh-CN.yaml", "locales/zh-TW.yaml", "locales/en.yaml", "locales/pt.yaml"}
   ```

3. In `Init()` function's localizers block, after `localizers[LangEn] = i18n.NewLocalizer(bundle, LangEn)`, add:
   ```go
   localizers[LangPt] = i18n.NewLocalizer(bundle, LangPt)
   ```

4. In `normalizeLang()` function, after the zh-TW case and before the zh case (or after the zh case, alphabetical doesn't matter for HasPrefix), add:
   ```go
   case strings.HasPrefix(lang, "pt"):
       return LangPt
   ```

5. In `SupportedLanguages()` function, change:
   ```go
   return []string{LangZhCN, LangZhTW, LangEn}
   ```
   to:
   ```go
   return []string{LangZhCN, LangZhTW, LangEn, LangPt}
   ```

Verify with: `cd /home/ubuntu/docker/Atius/router-ai-atius && go build ./i18n/...`
</action>

### Task 05: Patch web/default/src/i18n/config.ts — adicionar pt

<read_first>
- File: `web/default/src/i18n/config.ts` (target file, current state on feat/pt-native)
- File: Upstream canonical: `https://raw.githubusercontent.com/QuantumNous/new-api/main/web/default/src/i18n/config.ts`
</read_first>

<acceptance_criteria>
- File imports `pt` from `./locales/pt.json`
- `resources` object has `pt` key
- `i18n.init({ ... supportedLngs: [...] })` includes `'pt'` in the supportedLngs array
- No "i18n" mentioned in comments related to PT addition
- File parses as valid TypeScript: `bun run typecheck` (from web/default/) passes
</acceptance_criteria>

<action>
On feat/pt-native, edit `web/default/src/i18n/config.ts` with 3 hunks:

1. In the imports block (after `import ja from './locales/ja.json'`), add:
   ```typescript
   import pt from './locales/pt.json'
   ```

2. In the `resources` object (after `ja,`), add:
   ```typescript
   pt,
   ```

3. In the `i18n.init()` call's `supportedLngs` array (after `'ja',`), add:
   ```typescript
   'pt',
   ```

Verify with: `cd /home/ubuntu/docker/Atius/router-ai-atius/web/default && bun install && bun run typecheck`
</action>

### Task 06: Patch web/default/src/i18n/languages.ts — adicionar pt option

<read_first>
- File: `web/default/src/i18n/languages.ts` (target file, current state on feat/pt-native)
- File: Upstream canonical: `https://raw.githubusercontent.com/QuantumNous/new-api/main/web/default/src/i18n/languages.ts`
</read_first>

<acceptance_criteria>
- `INTERFACE_LANGUAGE_OPTIONS` array has new entry `{ code: 'pt', label: 'Português' }` at the end
- `normalizeInterfaceLanguage()` function is UNCHANGED from upstream (no case-insensitive refactor — parity with upstream)
- `InterfaceLanguageCode` type includes `'pt'`
- File parses as valid TypeScript: `bun run typecheck` (from web/default/) passes
</acceptance_criteria>

<action>
On feat/pt-native, edit `web/default/src/i18n/languages.ts` with 1 hunk:

In the `INTERFACE_LANGUAGE_OPTIONS` array, after the last entry (`{ code: 'vi', label: 'Tiếng Việt' }`), add:
```typescript
  { code: 'pt', label: 'Português' },
```

DO NOT modify `normalizeInterfaceLanguage` — keep parity with upstream. The TypeScript type
`InterfaceLanguageCode` is derived from `typeof INTERFACE_LANGUAGE_OPTIONS[number]['code']`,
so it will auto-include `'pt'`.

Verify with: `cd /home/ubuntu/docker/Atius/router-ai-atius/web/default && bun run typecheck`
</action>

### Task 07: Verify clean scope (5 files only)

<read_first>
- File: `.planning/phases/v2.12-pt-native-upstream-sync/01-CONTEXT.md` (decisions about scope)
</read_first>

<acceptance_criteria>
- `git diff upstream/main --stat` shows exactly 5 files modified
- `git diff upstream/main --name-only` returns exactly: `i18n/i18n.go`, `i18n/locales/pt.yaml`, `web/default/src/i18n/config.ts`, `web/default/src/i18n/languages.ts`, `web/default/src/i18n/locales/pt.json`
- `git diff upstream/main --shortstat` shows reasonable line count (estimate: 200-450 lines added, 0-5 deleted, since most are file additions)
- No files from PR #5245 contamination list are present: not `docker-compose.yml`, not `podman-compose.yml`, not `podman/**`, not `integration/middleware/**`, not `.planning/**`, not `docs/**`, not `web/default/src/routes/(auth)/login.tsx`, not `web/default/src/routeTree.gen.ts`, not `web/default/vitest.config.ts`, not `web/default/src/test/setup.ts`, not `VERSION`, not `.dockerignore`, not `.gitignore`, not `web/default/package.json`
</acceptance_criteria>

<action>
Run in /home/ubuntu/docker/Atius/router-ai-atius on branch feat/pt-native:

1. `git diff upstream/main --stat` — must show exactly 5 files
2. `git diff upstream/main --name-only` — verify the 5-file list
3. If contamination found:
   - `git restore <contaminated-file>` for each unwanted file
   - If contamination is in a file we WANT modified (rare), edit the file to remove the contamination hunks
4. Re-verify with step 1
5. Generate diff for review: `git diff upstream/main > /tmp/feat-pt-native-clean.diff`
6. Sanity check diff size: `wc -l /tmp/feat-pt-native-clean.diff` should be < 5000 lines

If the working tree has any other uncommitted changes, abort and ask user.
</action>

### Task 08: Final validation — Go build + frontend typecheck + JSON validation

<read_first>
- File: All 5 modified files
</read_first>

<acceptance_criteria>
- `go build ./...` exits 0 (in /home/ubuntu/docker/Atius/router-ai-atius)
- `cd web/default && bun install && bun run typecheck` exits 0
- `cd web/default && bun run build` exits 0 (full production build, catches i18n bundle errors)
- `jq empty i18n/locales/pt.yaml` exits 0 — but pt.yaml is YAML, not JSON. Use `python3 -c "import yaml; yaml.safe_load(open('i18n/locales/pt.yaml'))"` exits 0 instead
- `jq empty web/default/src/i18n/locales/pt.json` exits 0
- `go test ./i18n/...` (if any tests exist) passes — but upstream has no i18n tests, so this is just a sanity check

</acceptance_criteria>

<action>
Run in /home/ubuntu/docker/Atius/router-ai-atius on branch feat/pt-native:

1. `go build ./...` — must exit 0
2. `cd web/default && bun install` — must exit 0
3. `cd web/default && bun run typecheck` — must exit 0
4. `cd web/default && bun run build` — must exit 0
5. `python3 -c "import yaml; yaml.safe_load(open('i18n/locales/pt.yaml'))"` — must exit 0
6. `jq empty web/default/src/i18n/locales/pt.json` — must exit 0
7. `git status` should show only the 5 expected files as modified/new

If any step fails, fix and re-run. Common failure modes:
- `go build`: missing import or const reference — check i18n.go syntax
- `bun run typecheck`: pt.json shape mismatch with en.json — diff against en.json
- `bun run build`: i18n bundle error — likely wrong JSON key count, fix pt.json
</action>

## Must Haves (Goal-Backward Verification)

- [ ] Branch `feat/pt-native` exists locally, based on `upstream/main` (NOT fork's main)
- [ ] Working tree on feat/pt-native has exactly 5 modified/new files
- [ ] The 5 files are: i18n/i18n.go, i18n/locales/pt.yaml, web/default/src/i18n/config.ts, web/default/src/i18n/languages.ts, web/default/src/i18n/locales/pt.json
- [ ] pt.json key count = en.json key count (parity)
- [ ] pt.yaml key count matches expected 227 keys (per STATE.md)
- [ ] `go build ./...` passes
- [ ] `bun run typecheck` passes
- [ ] `bun run build` passes
- [ ] No "i18n" mention in PT-related comments (Portuguese/pt only)
- [ ] No fork-Atius contamination (no podman, no model-detailed, no docker-compose rebrand, no .planning, no docs/, no vitest, no login.tsx, no routeTree.gen.ts changes)
- [ ] `normalizeInterfaceLanguage` is UNCHANGED (upstream parity)

## Files NOT in scope (anti-checklist)

These files MUST NOT appear in the diff:
- `docker-compose.yml`, `podman-compose.yml`, `VERSION`
- `podman/**`, `integration/middleware/**`
- `.planning/**`, `docs/**`
- `web/default/src/routes/(auth)/login.tsx`, `web/default/src/routeTree.gen.ts`
- `web/default/vitest.config.ts`, `web/default/src/test/setup.ts`
- `web/default/package.json` (no dep changes)
- `.dockerignore`, `.gitignore`

## Notes

- Phase 2 (push + close PR #5245 + open new PR) é fase separada, dependente desta.
- Não commitar nada — push e commit são na Phase 2 (workflow seguro, autorização explícita).
- Working tree changes em `feat/pt-native` são o entregável desta fase.
- Branch é local-only; `git push` não é executado aqui.
