# RESEARCH.md — Session Fix Investigation

## Problema
Sessão expira rapidamente no browser ao usar o dashboard do Atius Router (router.atius.com.br).

## Arquitetura Atual

### Fluxo de Autenticação
```
Browser → Apache (SSL) → new-api:3000 (porta 3301) → Go server
                                    ↑
                           gin-contrib/sessions (cookie store)
```

### Sessão (Backend Go)
- **MaxAge**: 2592000 (30 dias) — configurado em `main.go:181`
- **Cookie**: `session`, HttpOnly, Secure:false, SameSite:Strict
- **Storage**: cookie store (dados criptografados no cookie do browser)
- **Renovação**: passiva a cada request válido

### Autenticação Middleware (`middleware/auth.go`)
1. Tenta session cookie primeiro (`sessions.Default(c)`)
2. Se session não tem `username`, verifica `Authorization: Bearer <token>`
3. Valida `New-Api-User` header contra session `id`
4. Se qualquer falha → 401 Unauthorized

### Frontend
- `localStorage.uid` — user ID persistente
- `localStorage.user` — dados do usuário (sem senha)
- `api.withCredentials: true` — inclui cookies em todas requisições
- Interceptor 401: limpa auth store + toast "Session expired!"

## Causas Identificadas

### 🔴 CRÍTICO: SESSION_SECRET não configurado
- **Arquivo**: `.env` não tem `SESSION_SECRET`
- **docker-compose.yml**: não passa `SESSION_SECRET` para o container
- **Fallback**: `SessionSecret = uuid.New().String()` — **muda a cada reinício do container**
- **Impacto**: Todos os cookies de sessão são invalidados ao reiniciar
- **Não é a causa** da expiração rápida (isso causaria expiração pós-reinício, não expiração em minutos)

### 🟡 ALTO: Cookie Secure=false em HTTPS
- **Arquivo**: `main.go:183`
- **Situação**: Apache serve via HTTPS (SSL), mas `Secure: false`
- **Impacto**: Browsers modernos podem rejeitar/rejeitar cookies non-Secure em contexto HTTPS
- **Mesmo não rejeitando**, browsers não enviam cookies Secure em requisições cross-origin

### 🟡 ALTO: SameSite=Strict em ambiente com navegação SPA
- **Arquivo**: `main.go:184`
- **Impacto**: Cookie não enviado em navegações cross-site (mas isso é desejável para CSRF)
- **Problema**: Se há redirects ou requisições cross-origin, cookie não é incluído

### 🟢 MÉDIO: Sem endpoint de renovação ativa
- Não há `/api/user/ping` ou similar para manter sessão viva
- Sessão só renova passivamente com requests ao backend

## Como o cookie é enviado atualmente

1. **Mesma origem** (mesmo domínio, mesma porta): sempre enviado
2. **Via Apache proxy**: Request headers forwards intactos se `ProxyPreserveHost On`
3. **Cookie path**: `/` — enviado para todas rotas

## Teste de Diagnóstico

O problema "sessão expira rapidamente" tipicamente significa:
- **Em minutos**: problema de configuração de cookie ou validação de sessão
- **Após reinício**: problema de SESSION_SECRET
- **Sempre após fechar navegador**: comportamento esperado (MaxAge não é set, ou browser fecha)

## Soluções Propostas

| # | Solução | Impacto | Risco |
|---|---------|---------|-------|
| 1 | Adicionar SESSION_SECRET ao .env + docker-compose | **Elimina invalidação pós-reinício** | Zero |
| 2 | Mudar Secure:true em production | **Cookie enviado corretamente em HTTPS** | Baixo |
| 3 | Adicionar endpoint /api/user/ping | **Mantém sessão ativa** | Zero |

## Prioridade de Implementação

1. **Primeiro**: SESSION_SECRET (elimina causa de perda de sessão)
2. **Segundo**: Secure:true (se SESSION_SECRET não resolver)
3. **Terceiro**: Endpoint ping (se ainda expirar)
