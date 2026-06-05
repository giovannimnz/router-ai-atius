# Plano: Migração i18n Nativa — Atius AI Router v2.12

**Contexto:** Substituir qualquer abordagem customizada de i18n pelo fluxo nativo de cada plataforma. O que já é nativo permanece; o que não é, migra.

---

## Estado Atual (pós-Phase 01 ✅)

| App | Framework | i18n nativo? | PT registrado? |
|---|---|---|---|
| Backend Go | go-i18n (nicksnyder) | ✅ Nativo | ✅ Sim (pt.yaml, 228 keys) |
| Frontend SPA | i18next + react-i18next | ✅ Nativo | ✅ Sim (pt.json, 4521 keys) |
| Frontend Classic | i18next + react-i18next | ✅ Nativo | ❌ **NÃO** — sem pt |
| Docs (Fumadocs) | Fumadocs Core i18n | ✅ Nativo | 🔄 Phase 02 em andamento |
| Apache (proxy) | n/a (infra) | n/a | n/a |
| electron/ | i18n | ❌ Fora do escopo | n/a |

---

## O que precisa ser feito (fora do escopo de Phase 01/02)

### 1. Classic Frontend — Sem suporte pt

**Arquivos:** `web/classic/src/i18n/` (i18n.js, language.js, locales/)
**Locais com pt:** ❌ Ausente
**Ação necessária:** 
- Criar `web/classic/src/i18n/locales/pt.json` com traduções
- NÃO tem `supportedLngs` explícito — verificar se pt já é detectado via LanguageDetector
- Verificar se o classic ainda é usado em produção

**Decisão autónoma:** Se o classic frontend não é servido ativamente (Apache só roteia pra :3301, que serve o SPA default), pode pular. Verificar `docker-compose.yml` e Apache config.

### 2. Docs (Phase 02) — Em andamento no fork repo

**Status:** 294/293 arquivos traduzidos ✅
**Pendente:** build + deploy Docker + browser validation

### 3. O que JÁ é nativo e deve ser PRESERVADO

```
router-ai-atius/
├── i18n/                          ← NATIVO (go-i18n)
│   ├── i18n.go                    ← Init(), T(), normalizeLang()
│   ├── keys.go                    ← Constantes tipadas
│   └── locales/
│       ├── en.yaml, zh-CN.yaml    ← UPSTREAM
│       ├── pt.yaml                ← NOSSO (nativo)
│       └── zh-TW.yaml             ← UPSTREAM
├── middleware/i18n.go             ← NATIVO (detectLanguage)
├── web/default/src/i18n/          ← NATIVO (i18next)
│   ├── config.ts                  ← resources + supportedLngs
│   ├── languages.ts               ← INTERFACE_LANGUAGE_OPTIONS
│   └── locales/
│       ├── en.json, fr.json...   ← UPSTREAM
│       └── pt.json               ← NOSSO (nativo, 4521 keys)
```

### 4. O que NÃO é nativo e DEVE ser removido ou adaptado

Nada encontrado. Zero código customizado. A auditoria não achou nenhum `pt-BR`, `pt-br`, locale loading custom, ou bypass do i18next/go-i18n.

---

## Próximas Ações

### Phase 02 — Finalização Docs

1. Task 06: Build + typecheck
2. Task 07: Rebuild Docker image  
3. Task 08: Restart docs container
4. Task 09: Playwright validation
5. Task 10: Commit + fork-sync mirror

### Phase 03 — Classic Frontend (se necessário)

1. Verificar se classic frontend está ativo em produção
2. Se sim: registrar pt em `web/classic/src/i18n/`
3. Seguir mesmo padrão fr/ja/ru/vi

### Milestone v2.12 — Fechamento

1. gsd-audit-milestone
2. gsd-complete-milestone
3. gsd-cleanup
