# Decisão: i18n pt Nativo — Infraestrutura Upstream

**Data:** 2026-06-05
**Contexto:** Adicionar locale `pt` (Português do Brasil) ao ecossistema Atius Router seguindo ESTRITAMENTE o padrão nativo do new-api. Zero código customizado. Zero scripts de locale loading próprios.

## Problema

O new-api (upstream) já tem i18n nativo em 3 sistemas separados:
1. **Backend Go** — go-i18n (nicksnyder) com YAML
2. **Frontend SPA** — i18next + react-i18next com JSON
3. **Docs** — Fumadocs Core com URL prefix + MDX per locale

A abordagem anterior (branch `feat/portuguese-translation-clean`) tinha um `pt.json` com 420 chaves extras que não existiam em `en.json`, contaminava outros locales durante o sync, e usava um formato de arquivo diferente do padrão.

## Decisão

1. **Phase 01 — App (Go + React SPA):** Registrar `pt` nos 5 pontos de registro nativos. O `pt.json` deve ter EXATAMENTE as mesmas chaves que `en.json` (4521). O `pt.yaml` deve ter EXATAMENTE as mesmas chaves que `en.yaml` (228). ✅ COMPLETO

2. **Phase 02 — Docs (Fumadocs):** Adicionar `pt` ao `defineI18n()` + `next.config.mjs` + traduzir 313 arquivos via script wrapper usando `mmx` CLI. 🔄 EM ANDAMENTO

3. **Manutenção:** Novo conteúdo upstream (novas chaves EN) cai como fallback EN no pt — sem quebra. Sync periódico via script.

## Contexto técnico

**Fumadocs i18n (`defineI18n`):**
```ts
export const i18n = defineI18n({
  defaultLanguage: 'en',
  languages: ['en', 'zh', 'ja', 'pt'],  // pt adicionado
  parser: 'dir',  // /en/docs/, /pt/docs/
});
```

**Next.js headers:**
```js
source: '/:lang(en|zh|ja|pt)/:path*',  // regex extendido
```

**Middleware:**
```ts
export default createI18nMiddleware(i18n);  // automático
```

**Protected content (via fork-sync):**
```yaml
protected_globs:
  - "content/docs/pt/**"  # PT files protegidos de overwrite upstream
```

## Impacto

- **Positivo:** Arquitetura 100% nativa. Qualquer dev familiarizado com new-api entende. Merge upstream é trivial.
- **Negativo:** Dependência do formato upstream — se mudar, temos que adaptar.
- **Risco:** 4521 chaves de frontend pra manter. Mas o sync é automatizado.

## Alternativas consideradas

| Alternativa | Rejeitada porque |
|---|---|
| Usar `pt-BR` como código | Quebra com `load: 'languageOnly'` no i18next. `pt` segue fr/ja/ru/vi |
| Traduzir só docs, app em EN | Experiência quebrada — usuário vê switch "Português" mas app fica EN |
| Script de tradução próprio (Node) | `mmx` CLI já tem keys configuradas, mais simples |
| Manter `pt-BR.json` antigo | 420 chaves extra, contaminava sync, formato inconsistente |

## Links

- [[v212-pt-native-i18n]] — projeto no vault
- [[2026-06-05-phase02-pt-fumadocs-autonomous]] — log de sessão
- `~/.planning/notes/native-i18n-complete-plan.md` — plano completo
- `~/.planning/notes/native-i18n-migration-plan.md` — plano de migração
