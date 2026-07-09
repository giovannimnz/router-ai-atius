# PLAN.md — phase-4-session-fix

## Meta
- **Phase**: 4
- **Slug**: session-fix
- **Objetivo**: Corrigir expiração rápida de sessão no browser
- **Roteiro**: v1.5 (nesta versão está sendo tratada a estabilidade do router)

## UAT (User Acceptance Criteria)

| # | Critério | Como validar |
|---|----------|--------------|
| 1 | `SESSION_SECRET` existe no `.env` com valor aleatório de 32+ chars | `grep SESSION_SECRET .env` retorna valor |
| 2 | `SESSION_SECRET` é passado ao container em `docker-compose.yml` | Variável `SESSION_SECRET` no environment do serviço new-api |
| 3 | Container `new-api` não reinicia com erro de `SESSION_SECRET` | `docker-compose up -d` executa sem erro |
| 4 | `Secure: true` no cookie de sessão em produção | Verificar `main.go:183` depois do rebuild |
| 5 | Após correção, sessão persiste após reinício do container | 1) Fazer login 2) `docker-compose restart new-api` 3) Sessão continua válida |
| 6 | Sessão persiste após fechar e abrir o navegador | Login → fechar navegador → abrir → sessão continua |

---

## Step 1: Adicionar SESSION_SECRET ao .env

### Ação
Gerar string aleatória de 64 chars (hex) e adicionar ao `.env`.

### Comando
```bash
openssl rand -hex 32
```

### Resultado
`.env` contém:
```
SESSION_SECRET=<valor-hex-64-chars>
```

### Tempo estimado: 2 minutos

---

## Step 2: Passar SESSION_SECRET ao container

### Ação
Adicionar `SESSION_SECRET` ao environment do serviço `new-api` em `docker-compose.yml`.

### Antes
```yaml
environment:
  - SQL_DSN=postgres://admin:***@db-newapi:5432/newapi?sslmode=disable
  - TZ=America/Sao_Paulo
  - LANG=pt_BR.UTF-8
```

### Depois
```yaml
environment:
  - SQL_DSN=postgres://admin:***@db-newapi:5432/newapi?sslmode=disable
  - TZ=America/Sao_Paulo
  - LANG=pt_BR.UTF-8
  - SESSION_SECRET=${SESSION_SECRET}
```

### Tempo estimado: 3 minutos

---

## Step 3: Rebuild da imagem Docker

### Ação
Reconstruir imagem com `docker build` e subir para GHCR.

### Comandos
```bash
# No diretório do projeto
cd /home/ubuntu/docker/Atius/router-ai-atius
docker build -t ghcr.io/giovannimnz/atius-ai-router:session-fix -f Dockerfile .
docker push ghcr.io/giovannimnz/atius-ai-router:session-fix
docker tag ghcr.io/giovannimnz/atius-ai-router:session-fix ghcr.io/giovannimnz/atius-ai-router:latest
docker push ghcr.io/giovannimnz/atius-ai-router:latest
```

### Resultado
Nova imagem disponível com `SESSION_SECRET` compilado no binário.

### Tempo estimado: 10-15 minutos (build + push)

---

## Step 4: Deploy no Oracle SRV-1

### Ação
Fazer pull da nova imagem e restartar o container.

### Comandos (via SSH para atius-srv-1)
```bash
ssh atius-srv-1
cd /home/ubuntu/docker/Atius/router-ai-atius
docker pull ghcr.io/giovannimnz/atius-ai-router:latest
docker-compose down
docker-compose up -d
docker ps | grep new-api
```

### Verificação
```bash
docker exec new-api env | grep SESSION_SECRET
```

### Tempo estimado: 5 minutos

---

## Step 5: Corrigir Secure:false no cookie (main.go)

### Ação
Mudar `Secure: false` para `Secure: true` em `main.go:183`.

### Antes
```go
store.Options(sessions.Options{
    Path:     "/",
    MaxAge:   2592000, // 30 dias
    HttpOnly: true,
    Secure:   false,   // ⚠️
    SameSite: http.SameSiteStrictMode,
})
```

### Depois
```go
// Em produção (via Apache HTTPS), Secure:true para enviar cookie corretamente
isProd := os.Getenv("GIN_MODE") == "release"
store.Options(sessions.Options{
    Path:     "/",
    MaxAge:   2592000, // 30 dias
    HttpOnly: true,
    Secure:   isProd,
    SameSite: http.SameSiteStrictMode,
})
```

### Observação
- Detectar automaticamente se está em produção via `GIN_MODE=release`
- Ou simplesmente hardcodar `true` já que o ambiente é sempre HTTPS via Apache
- Mesmo com `Secure: true`, o Apache proxy consegue repassar o cookie

### Tempo estimado: 5 minutos

---

## Step 6: Validar - Teste de Sessão Persistente

### Teste 1: Login → Refresh → Sessão Mantida
1. Abrir `https://router.atius.com.br`
2. Fazer login
3. Apertar F5 (refresh)
4. Verificar que **não** pede login novamente

### Teste 2: Login → Restart Container → Sessão Mantida
1. Fazer login
2. `docker-compose restart new-api` no srv-1
3. Refresh na página
4. Verificar que sessão continua

### Teste 3: Login → Fechar Navegador → Abrir → Sessão Mantida
1. Fazer login
2. **Importante**: fazer pelo menos 1 request depois do login (SPA precisa de activity)
3. Fechar navegador completamente
4. Abrir nova aba e ir para `router.atius.com.br`
5. Verificar que sessão continua

### Tempo estimado: 15 minutos

---

## Step 7: Commit e Tag

### Ação
Commit das mudanças e tag de versão.

```bash
git add -A
git commit -m "fix: add SESSION_SECRET and Secure cookie for session persistence

- Add SESSION_SECRET to .env (64-char hex from openssl rand -hex 32)
- Pass SESSION_SECRET to new-api container in docker-compose.yml
- Detect GIN_MODE=release to enable Secure cookie flag
- Fixes session expiration issues after container restart"

git tag v1.5.1-session-fix
git push origin main v1.5.1-session-fix
```

---

## Cron Job de Validação

Criar cron job para validar sessão a cada 30 minutos:

1. Fazer request autenticado (com cookie de sessão)
2. Verificar que retorna 200
3. Se 401, reportar "Sessão expirou inesperadamente"

---

## Timeline Total Estimada

| Step | Tempo |
|------|-------|
| Step 1 (SESSION_SECRET no .env) | 2 min |
| Step 2 (docker-compose.yml) | 3 min |
| Step 3 (Build Docker image) | 15 min |
| Step 4 (Deploy no Oracle) | 5 min |
| Step 5 (Fix Secure:true) + rebuild | 20 min |
| Step 6 (Testes de validação) | 15 min |
| Step 7 (Commit) | 3 min |
| **Total** | **~63 min** |

---

## Rollback Plan

Se algo der errado:

1. **Rollback docker-compose.yml**: remover `SESSION_SECRET` do environment
2. **Rollback main.go**: voltar `Secure: false`
3. **Restart container**: `docker-compose restart new-api`
4. **Rollback imagem**: `docker tag ghcr.io/giovannimnz/atius-ai-router:<previous-tag> ghcr.io/giovannimnz/atius-ai-router:latest`

---

## Dependencies

- Acesso SSH ao atius-srv-1
- GHCR credentials para push da imagem
- Acesso ao repositório git para commit
