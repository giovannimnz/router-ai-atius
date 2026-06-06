# Decisão: Apache Proxy para Next.js Docs — Lições da Phase 04

**Data:** 2026-06-06
**Contexto:** Phase 04 do v2.12 (Atius Router) — bugs de produção em
docs.atius.com.br detectados por user feedback ("site is broken, never tested").

## Problema

Next.js docs site (port 3003) é proxyado via Apache. A config original tinha
3 proxypasses para prefixos de locale (`/en/`, `/zh/`, `/ja/`) mas faltava
`/pt/` e — mais grave — **faltava `/_next/`** para os assets estáticos.

Resultado em produção:
- `/pt/docs/` retornava HTML do Go SPA (catch-all) com "New API" title
- `/_next/static/chunks/*.css` retornava `text/html` (Go SPA) em vez de CSS
- `/assets/atius-logo.svg` retornava 404
- Página renderizava 2 regras CSS, Times New Roman fallback, layout 1995

## Decisão

Adicionar 3 ProxyPass/Alias no vhost SSL `router.atius.com.br-le-ssl.conf`:

```
# D-02: PT locale (estava faltando — adicionado em Phase 04)
ProxyPass /pt/ http://127.0.0.1:3003/pt/
ProxyPassReverse /pt/ http://127.0.0.1:3003/pt/

# D-03: Logo do header Fumadocs
Alias /assets/atius-logo.svg /var/www/atius/atius-logo.svg
Alias /assets/atius-logo.png /var/www/atius/atius-logo.png
ProxyPass /assets/atius-logo.svg !
ProxyPass /assets/atius-logo.png !

# D-03b: Next.js chunks (CSS/JS) — root cause do CSS quebrado
ProxyPass /_next/ http://127.0.0.1:3003/_next/
ProxyPassReverse /_next/ http://127.0.0.1:3003/_next/
```

## Por que essa ordem (D-04b)

Apache ProxyPass é **first-match-wins**. As 3 regras DEVEM estar ANTES do
catch-all `ProxyPass / http://127.0.0.1:3030/` (linha 174 do vhost). Senão
qualquer request a `/pt/`, `/_next/`, `/assets/...` cai no Go SPA.

Padrão de validação pós-patch:
1. `apache2ctl configtest` → "Syntax OK"
2. `systemctl reload apache2`
3. `curl -sIk` em cada path crítico (com `--resolve host:443:127.0.0.1` para
   bypassar CF e ir direto no Apache origin)
4. Chromium headless + CDP raw WS para visual (mmx vision API)

## Pendente (não automatizado)

- **Cloudflare cache** ainda serve 404 antigo para `/pt/docs/` (age 4h) e
  `/assets/atius-logo.svg` (HIT). Origin está correto, novos requests
  funcionam. User precisa purgar manualmente no dashboard CF, ou esperar
  TTL (~24h).
- **Automação:** Phase 05 (M099-style) deveria ter um script `purge-cf.sh`
  que aceita paths como arg e chama CF API. Token CF não está no shell.

## Lições (não repetir)

1. **Next.js docs em Apache = 3 proxypasses, não 1.** Locale prefix
   (`/{lang}/`) + asset root (`/_next/`) + public assets (`/assets/`, `/og/`,
   `/api/search/`, etc). Cada um precisa de regra explícita.

2. **Visual validation é MANDATÓRIA antes de declarar done.** Esta Phase 04
   só foi pega porque user navegou e viu "quebrado". Prazo de detecção
   poderia ter sido 0 se validação visual fosse parte do execute-phase,
   não do verify-work do user.

3. **CSS rule count é métrica útil.** `<2 rules CSS = quebrado`,
   `>100 rules = provavelmente ok`. Rápido de checar via
   `document.styleSheets[0].cssRules.length` no headless browser.

4. **Curl direto no origin ≠ curl via Cloudflare.** CF cache HIT pode
   mascarar bugs já corrigidos no origin. Use `--resolve host:443:127.0.0.1`
   pra bypassar CF e testar a config Apache real.

5. **chrome-devtools MCP falha em SPAs grandes.** Bundle de 2.4MB JS no
   cold cache → MCP timeout 60s. Workaround: raw WS via Python
   `websocket-client` (skill `chrome-devtools-mcp-raw-websocket-bypass`).

## Cross-refs

- Vault: `60-LOGS/2026-06-06-atius-router-phase-04-prod-docs-bugfixes.md`
- Plan: `.planning/phases/04-prod-docs-bugfixes/04-CONTEXT.md`
- Result: `.planning/phases/04-prod-docs-bugfixes/04-SUMMARY.md`
- Skill: `chrome-devtools-mcp-raw-websocket-bypass`
- Skill: `webapp-visual-validation` (candidato a criar — pipeline formal)
