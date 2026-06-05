# Phase 01: pt Locale Registration — PLAN.md

**Phase:** 01
**Status:** Ready
**Date:** 2026-06-05
**Branch:** feat/pt-native

## Read First

- `01-CONTEXT.md` — todas as decisões (D-01 a D-04)
- `i18n/i18n.go` — padrão de registro backend
- `web/default/src/i18n/config.ts` — padrão de registro frontend
- `web/default/src/i18n/languages.ts` — INTERFACE_LANGUAGE_OPTIONS
- `i18n/locales/en.yaml` — source of truth (279 keys)

---

## Tasks

### Task 01: Recuperar pt.json do stash

Recuperar o arquivo `web/default/src/i18n/locales/pt.json` do stash `stash@{0}` (branch `portuguese-translation-clean`).

```
cd /home/ubuntu/docker/Atius/router-ai-atius
git stash show -p stash@{0} -- web/default/src/i18n/locales/pt.json > /tmp/pt.json.stash
# Extrair só o conteúdo do arquivo, aplicar em web/default/src/i18n/locales/pt.json
```

**Validation:** `cat web/default/src/i18n/locales/pt.json | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'Keys: {len(d)}')"` mostra ~3910 chaves.

---

### Task 02: Criar backend pt.yaml (279 keys)

Criar `i18n/locales/pt.yaml` com todas as 279 chaves de `i18n/locales/en.yaml`, traduzidas para PT-BR.

**Método:** Batch translate via LLM. Estrutura: chave = `common.invalid_params`, valor = tradução PT-BR contextual.

**Convenções (do i18n-contextual-translation skill):**
- "Sign in" → "Iniciar sessão" (não "Login")
- "Sign out" → "Sair" (não "Logout")
- Brand names em EN: QuantumNous, new-api, OpenAI, Anthropic, etc.
- Tech terms em EN: API, SDK, JWT, OAuth, WebAuthn, CLI, etc.

**Validation:** `python3 -c "import yaml; d=yaml.safe_load(open('i18n/locales/pt.yaml')); print(f'Keys: {len(d)}')"` mostra 279.

---

### Task 03: Registrar pt no backend Go

3 edições em `i18n/i18n.go`:

1. **Constante:** Adicionar `LangPt = "pt"` (após LangEn, seguindo padrão zh-CN/zh-TW)
2. **Init():** Adicionar `"locales/pt.yaml"` ao array `files` + `localizers[LangPt] = i18n.NewLocalizer(bundle, LangPt)`
3. **normalizeLang():** Adicionar case `strings.HasPrefix(lang, "pt")` → `return LangPt`
4. **SupportedLanguages():** Adicionar `LangPt` ao slice

**Validation:** `go build ./...` compila sem erros.

---

### Task 04: Registrar pt no frontend React

3 edições + 1 import:

1. **config.ts:** `import pt from './locales/pt.json'` + adicionar `pt` a `resources` + adicionar `'pt'` a `supportedLngs`
2. **languages.ts:** Adicionar `{ code: 'pt', label: 'Português' }` ao `INTERFACE_LANGUAGE_OPTIONS`

**Validation:**
- `cd web/default && bun run typecheck` sem erros
- `cd web/default && bun run build` gera dist corretamente
- Language switcher renderiza "Português" como opção

---

### Task 05: Sincronizar pt.json com en.json

Rodar o script de sync para garantir que pt.json tem exatamente as mesmas chaves que en.json:

```bash
cd web/default
bun run i18n:sync
```

**Validation:** O sync report mostra `pt: missing=0 extras=0`. Chaves extras (que existem em pt mas não em en) são removidas. Chaves missing (em en mas não em pt) são adicionadas com fallback "".

---

### Task 06: Typecheck + Build

```bash
go build ./...
cd web/default && bun run typecheck && bun run build
```

**Validation:** Ambos passam sem erros.

---

### Task 07: Browser validation com chrome-devtools

1. Deploy local (Docker ou `go run`)
2. chrome-devtools: navegar para a URL local
3. Clicar no language switcher (ícone de globo)
4. Selecionar "Português"
5. Tirar screenshot de visão: verificar que a UI renderiza em PT-BR
6. Verificar localStorage: `localStorage.getItem("i18nextLng")` → `"pt"`

**Validation:** Screenshot mostra UI 100% em Português. Sem strings em inglês residual (exceto brand/tech terms).

---

### Task 08: Commit

```bash
git add -A
git commit -m "feat(i18n): register pt locale in native i18n infrastructure

- Backend: pt.yaml (279 keys) + LangPt constant + normalizeLang + SupportedLanguages
- Frontend: pt.json + config.ts resources/supportedLngs + languages.ts option
- Follows exact same pattern as fr, ja, ru, vi locales
- Zero custom code — only registration points"
```

---

## Acceptance Criteria

- [ ] `go build ./...` compila sem erros
- [ ] `bun run typecheck` sem erros
- [ ] `bun run build` gera dist
- [ ] Language switcher mostra "Português" como opção
- [ ] Selecionar "Português" → UI renderiza em PT-BR (validado via browser)
- [ ] Backend responde mensagens de erro em PT-BR quando `Accept-Language: pt`
- [ ] `git diff --stat` mostra apenas modificações nos 6 pontos de registro

## Push Policy

| Operação | Autorização |
|---|---|
| `git commit` no branch `feat/pt-native` | Auto — local only |
| `git push origin feat/pt-native` | Hard-gate — "pode push?" |
| PR contra upstream QuantumNous/new-api | Fora do escopo desta phase |
