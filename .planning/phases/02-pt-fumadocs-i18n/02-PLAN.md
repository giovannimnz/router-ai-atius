# Phase 02: pt Fumadocs i18n — PLAN.md

**Phase:** 02
**Status:** Ready
**Date:** 2026-06-05
**Branch:** feat/pt-fumadocs (NEW — separate branch in the FORK REPO, not router-ai-atius)

## Read First

- `02-CONTEXT.md` — arquitetura, decisões D-01..D-04
- `/home/ubuntu/fork-sync/projects/atius-router-docs/sync.yaml` — já tem `protected_globs: ["content/docs/pt/**"]`
- `/home/ubuntu/fork-sync/projects/atius-router-docs/pt-content/docs/pt/` — 11 PT files já existentes
- `/home/ubuntu/GitHub/refs/new-api-docs-v1/scripts/translate-docs.ts` — script a adaptar
- `/home/ubuntu/GitHub/refs/new-api-docs-v1/src/lib/i18n.ts` — locale config

---

## ⚠️ Critical: Work Location

Phase 02 happens in the **FORK REPO** `giovannimnz/new-api-docs-v1`, NOT in `router-ai-atius`. The router-ai-atius `integration/docs/` is just an extracted-types mirror.

**Work in this order:**
1. **First**: clone or sync the fork repo locally
2. **Edit**: i18n config + translate script + content
3. **Commit**: in the fork repo
4. **Build + deploy**: Docker image rebuild + container restart
5. **Update**: fork-sync `pt-content/` to mirror final state
6. **Commit**: in router-ai-atius (the fork-sync `pt-content` mirror)

---

## Tasks

### Task 01: Sync fork repo locally

Get a working copy of the Atius-branded fork (not the upstream `refs/new-api-docs-v1/`).

```bash
# Option A: Fresh clone
cd ~/GitHub
git clone https://github.com/giovannimnz/new-api-docs-v1.git atius-docs
cd atius-docs
git checkout -b feat/pt-fumadocs

# Option B: Update existing local clone (if any)
cd /path/to/existing/atius-docs
git checkout -b feat/pt-fumadocs
git pull origin main
```

**Validation:** `git log --oneline -3` shows Atius commits (Atius Router branding, etc.). `cat package.json` shows `"name": "new-api-docs-v1"`.

---

### Task 02: Register `pt` in i18n config

2 edits in the fork repo.

**Edit 1 — `src/lib/i18n.ts`:**
```ts
export const i18n = defineI18n({
  defaultLanguage: 'en',
  languages: ['en', 'zh', 'ja', 'pt'],  // ← add 'pt'
  parser: 'dir',
});
```

**Edit 2 — `next.config.mjs`:**
```js
// Line 30: change regex
source: '/:lang(en|zh|ja|pt)/:path*',  // ← add |pt
```

**Validation:** `git diff` shows only these 2 files modified. `grep -r "'pt'" src/` confirms.

---

### Task 03: Copy existing 11 PT files from fork-sync

The 11 PT files in `/home/ubuntu/fork-sync/projects/atius-router-docs/pt-content/docs/pt/` are proven seed content. Copy them to the fork repo.

```bash
# In the fork repo
mkdir -p content/docs/pt
cp -a /home/ubuntu/fork-sync/projects/atius-router-docs/pt-content/docs/pt/. content/docs/pt/

# Add per-section meta.json files (already exist in seed)
ls content/docs/pt/
# Should show: index.mdx meta.json guide/ installation/ api/
```

**Validation:** `find content/docs/pt -type f | wc -l` shows 11. `cat content/docs/pt/index.mdx | head -3` shows PT-BR content.

---

### Task 04: Adapt `translate-docs.ts` for en→pt

Currently the script translates FROM `zh` to `en` and `ja`. We need to translate FROM `en` to `pt` (source override).

**Edit 1 — `scripts/translate-docs.ts` LANGUAGES object:**
```ts
const LANGUAGES = {
  en: { name: 'English', nativeName: '英文', dir: 'en' },
  ja: { name: 'Japanese', nativeName: '日文', dir: 'ja' },
  pt: { name: 'Portuguese', nativeName: 'Português', dir: 'pt' },  // ← NEW
} as const;
```

**Edit 2 — `GLOSSARY`:**
Replace the Chinese→English glossary with an English→Portuguese one (preserve code patterns, brand terms, etc.).

**Edit 3 — Add `SOURCE_LANGUAGE` constant:**
```ts
const SOURCE_LANGUAGE = 'en';  // pt translates from en (not from zh)
```

**Edit 4 — Adjust the file scanning logic** to read from `content/docs/${SOURCE_LANGUAGE}/` and write to `content/docs/pt/`.

**Validation:** `bun run scripts/translate-docs.ts --help` (or `bun run translate --help`) shows new `pt` target. Dry run on 1 file works.

---

### Task 05: Translate index pages first (high-impact)

Translate the root `index.mdx` for guide, installation, api, apps, skills, support, business — these appear in sidebar navigation.

```bash
# In the fork repo
export OPENAI_API_KEY=...
export OPENAI_BASE_URL=https://api.openai.com/v1  # or Atius router
export OPENAI_MODEL=gemini-2.5-flash
export FORCE_TRANSLATE=true
export INCREMENTAL_TRANSLATE=false

# Translate only the index pages first
bun run translate -- --specific-path "guide/index.mdx" --lang pt
bun run translate -- --specific-path "installation/index.mdx" --lang pt
# ... repeat for each section
```

**Validation:** `ls content/docs/pt/*/index.mdx` shows all 7 sections translated.

---

### Task 06: Bulk translate all 313 files

Run the full batch with workers.

```bash
# In the fork repo
export MAX_WORKERS=3
export RETRY_DELAY=2
export RETRY_BACKOFF=2.0

# Run full batch (this may take 30-60 minutes for 302 files)
bun run translate -- --force-all --lang pt
```

**Validation:** `find content/docs/pt -type f | wc -l` shows 313+. Check translation log for any failures.

---

### Task 07: Typecheck + Build

```bash
# In the fork repo
bun run typecheck
bun run build
```

**Validation:** Both pass. `ls .next/server/app/\[lang\]/pt/docs/` shows the PT route is built.

---

### Task 08: Rebuild Docker image

```bash
cd /home/ubuntu/fork-sync/projects/atius-router-docs
./atius-router-docs-rebrand.sh  # Reapply Atius branding if needed
cd /path/to/atius-docs

podman build -f /home/ubuntu/fork-sync/projects/atius-router-docs/Dockerfile.template \
  -t localhost/router-ai-atius-docs:local .

# Or use the actual Dockerfile if it exists
podman build -t localhost/router-ai-atius-docs:local .
```

**Validation:** Image builds. `podman images | grep router-ai-atius-docs` shows new image.

---

### Task 09: Restart docs container

```bash
podman stop router-ai-atius-docs
podman rm router-ai-atius-docs
podman run -d --name router-ai-atius-docs \
  --network router-ai-atius_internal \
  localhost/router-ai-atius-docs:local
```

**Validation:** `podman ps | grep docs` shows running. `curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:3003/pt/docs/` returns 200.

---

### Task 10: Browser validation with Playwright/Chromium

```bash
# Use Playwright (Chromium) to validate
node /tmp/validate-pt-docs.cjs
```

Test cases:
- `/pt/docs/` returns 200, renders PT-BR content
- `/pt/docs/installation/` renders PT-BR
- `/pt/docs/skills/` returns 200, renders PT-BR
- `/pt/docs/api/` returns 200, renders PT-BR
- `rel="alternate" hrefLang="pt"` present in `<head>`

**Validation:** All 5 test cases pass. Screenshot confirms PT-BR rendering.

---

### Task 11: Update fork-sync pt-content mirror

Sync the translated content back to the fork-sync project so future upstream merges preserve it.

```bash
# Copy the final pt/ content to fork-sync
cp -a content/docs/pt/. /home/ubuntu/fork-sync/projects/atius-router-docs/pt-content/docs/pt/

# Verify
diff -r content/docs/pt/ /home/ubuntu/fork-sync/projects/atius-router-docs/pt-content/docs/pt/
```

**Validation:** `diff` shows no differences. `find /home/ubuntu/fork-sync/projects/atius-router-docs/pt-content -type f | wc -l` shows 313+.

---

### Task 12: Commit in both repos

**Fork repo** (the main change):
```bash
cd /path/to/atius-docs
git add -A
git commit -m "feat(i18n): add pt locale to Fumadocs docs site

- src/lib/i18n.ts: add 'pt' to languages list
- next.config.mjs: extend :lang regex to include pt
- scripts/translate-docs.ts: add pt to LANGUAGES, source = en
- content/docs/pt/: 313 MDX files translated from en (PT-BR)
- 11 files were pre-existing seed from fork-sync pt-content

Follows same pattern as en/ja locales. PT-BR translation covers
all sections: guide, installation, api, apps, skills, support, business."
```

**Router-ai-atius** (just the fork-sync mirror update):
```bash
cd /home/ubuntu/docker/Atius/router-ai-atius
git add -A
git commit -m "docs(fork-sync): sync atius-router-docs pt-content mirror

Updated pt-content/ to reflect translated state of fork repo's
content/docs/pt/. All 313+ files now in PT-BR."
```

---

## Acceptance Criteria

- [ ] `git diff` in fork repo shows: 2 i18n config edits + 1 translate script edit + 313 content files
- [ ] `bun run typecheck` passes
- [ ] `bun run build` succeeds
- [ ] Docker image `localhost/router-ai-atius-docs:local` rebuilt
- [ ] Container `router-ai-atius-docs` running
- [ ] `/pt/docs/` returns 200 (via curl)
- [ ] `/pt/docs/skills/` returns 200
- [ ] `rel="alternate" hrefLang="pt"` in `<head>` (via Playwright)
- [ ] PT-BR content visible in browser (via Playwright vision)
- [ ] fork-sync `pt-content/` mirror updated
- [ ] Commits in both repos

## Push Policy

| Operation | Authorization |
|---|---|
| Commit on fork (giovannimnz/new-api-docs-v1) | Auto — local only |
| `git push origin feat/pt-fumadocs` (fork) | Hard-gate — "pode push?" |
| PR to upstream QuantumNous/new-api-docs-v1 | Hard-gate per operation |
| `podman build` + `podman run` (production deploy) | Hard-gate — "pode deploy?" |
| `podman stop` + `podman rm` (production) | Hard-gate — "pode restart?" |
| Commit in router-ai-atius (fork-sync mirror) | Auto — local only |
