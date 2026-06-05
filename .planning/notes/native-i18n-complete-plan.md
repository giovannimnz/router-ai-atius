# Plano Completo: Sistema i18n Nativo — Atius AI Router

**Data:** 2026-06-05
**Status:** Em revisão
**Objetivo:** O router-ai-atius DEVE usar 100% a infraestrutura i18n nativa do new-api upstream. Zero código customizado de tradução. O conteúdo PT-BR relevante fica nas documentações (Fumadocs).

---

## Princípios

1. **O new-api (upstream) tem i18n nativo** — go-i18n (backend) + i18next (frontend) + Fumadocs (docs)
2. **Nosso fork adiciona pt seguindo o MESMO padrão** dos locales existentes
3. **Nada de "solução" customizada** — sem scripts de locale loading próprios, sem configs de tradução fora do padrão
4. **Se o upstream remove pt, a gente reaplica no merge** — não cria workaround

---

## Stack completa (5 sistemas)

| # | Sistema | Framework i18n | O que é nativo? | Nosso pt | Ação |
|---|---|---|---|---|---|
| 1 | Backend Go | go-i18n (nicksnyder) | ✅ `i18n/i18n.go`, `middleware/i18n.go`, `Accept-Language`, `UserSetting.Language` | ✅ `pt.yaml` (228 keys) | MANTER |
| 2 | Frontend SPA (default) | i18next + react-i18next | ✅ `config.ts`, `languages.ts`, `LanguageSwitcher`, `language-preferences-card` | ✅ `pt.json` (4521 keys) | MANTER |
| 3 | Frontend Classic | i18next + react-i18next | ✅ `i18n.js`, `language.js` | ❌ Sem pt | DECIDIR |
| 4 | Docs (Fumadocs) | Fumadocs Core i18n | ✅ `defineI18n()` + URL prefix + MDX per locale | 🔄 Phase 02 | COMPLETAR |
| 5 | Electron | i18next? | ❌ Precisa verificar | ❌ | FORA DO ESCOPO |

---

## O que JÁ é nativo (preservar)

### Backend Go
```
i18n/                    ← Package go-i18n (upstream)
  i18n.go               ← Init(), T(), Translate(), GetLangFromContext()
  keys.go               ← 332 constantes tipadas
  locales/
    en.yaml             ← UPSTREAM
    zh-CN.yaml          ← UPSTREAM
    zh-TW.yaml          ← UPSTREAM
    pt.yaml             ← NOSSO (228 keys, mesmo formato)
middleware/
  i18n.go               ← detectLanguage() (UserSetting → Accept-Language → en)
```

**Nativo porque:** follow the exact upstream pattern. `pt.yaml` is a YAML file in `locales/`, loaded in `Init()`, normalized in `normalizeLang()`, and checked in `IsSupported()`. The `normalizeLang("pt-BR")` maps to `LangPt`. Everything else is upstream code.

### Frontend SPA (default)
```
web/default/src/i18n/
  config.ts             ← i18next init (upstream + nossa adição pt)
  languages.ts          ← INTERFACE_LANGUAGE_OPTIONS (upstream + nossa adição)
  locales/
    en.json, fr.json... ← UPSTREAM
    pt.json             ← NOSSO (4521 keys, mesmo formato)
components/
  language-switcher.tsx ← UPSTREAM (itera INTERFACE_LANGUAGE_OPTIONS)
features/profile/
  language-preferences-card.tsx ← UPSTREAM (Select com persistência)
```

**Nativo porque:** `pt` é adicionado a `resources`, `supportedLngs`, e `INTERFACE_LANGUAGE_OPTIONS` — exatamente como os outros 7 locales. O switcher e o profile card já fazem tudo.

---

## O que PRECISA de ação

### Phase 02 — Docs PT-BR (93% completo)

| Tarefa | Status |
|---|---|
| Register pt in i18n.ts + next.config.mjs | ✅ |
| Seed files (11) copied | ✅ |
| 294/293 files translated | ✅ |
| Frontmatter fix (37 files) | ✅ |
| Build | 🔄 Rodando |
| Rebuild Docker image | ⏳ |
| Restart container | ⏳ |
| Playwright validation | ⏳ |
| Commit | ⏳ |
| fork-sync mirror update | ⏳ |

### Classic Frontend (opcional)

`web/classic/` é o tema clássico do new-api. Verificar se está ativo em produção:

```bash
# Verificar se o classic frontend é servido
grep -r "classic" /etc/apache2/sites-enabled/router*
# Verificar docker-compose ou env
grep -r "CLASSIC\|classic" .env docker-compose*
```

**Se ativo:** Adicionar `pt` seguindo o MESMO padrão dos outros 7 locales.
**Se inativo (default SPA usado):** Pular — sem necessidade.

---

## Plano de manutenção

### Como o pt sobrevive a merges upstream

O **fork-sync** gerencia isso:

1. **router-ai-atius (new-api):** Os arquivos `pt.yaml` e `pt.json` são adicionados ao repo. Quando o upstream faz merge, a fork-sync pode ter `protected_globs` ou o arquivo simplesmente não conflita (não existe no upstream).

2. **new-api-docs-v1 (fork):** `sync.yaml` já tem `protected_globs: ["content/docs/pt/**"]`. Os 313+ arquivos PT estão protegidos de overwrite upstream.

### Como novas chaves de tradução são adicionadas

**Backend:** Quando o upstream adiciona novas chaves em `en.yaml`, a build não quebra — go-i18n faz fallback automático pra en. Periodicamente (ou em PR de merge), rodar sync: `diff en.yaml pt.yaml` e traduzir as novas chaves.

**Frontend:** O script `bun run i18n:sync` gera um report com `missing=65`. As chaves novas ficam como string vazia e i18next fallback para en. Pode-se rodar o script Python periodicamente pra traduzir.

**Docs:** No push pra main do fork, o CI (`translate.yml`) pode ser configurado pra traduzir automaticamente en→pt (adaptando o workflow).

---

## Risco: Manter pt.json no SPA vs remover

| Opção | Prós | Contras | Decisão |
|---|---|---|---|
| **Manter** (Phase 01) | App 100% em PT-BR. Consistência total. | 4521 chaves pra manter. Sync periódico necessário. | ✅ **Recomendado** |
| **Remover** (app só EN) | Zero manutenção de locale. App sempre atualizado. | Usuário PT-BR só vê docs traduzidas, app em EN. Experiência quebrada. | ❌ Não recomendado |

**Decisão autónoma:** Manter Phase 01. O esforço de sync é mínimo (script automatizado). A experiência do usuário fica completa — app + docs em PT-BR.
