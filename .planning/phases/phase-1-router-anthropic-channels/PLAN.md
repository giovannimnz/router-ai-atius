# Phase 1 Plan — Router Anthropic Channels + Session Fix

## Tasks

### Task 1: Aumentar session timeout para 12h
**File:** `main.go`
**Change:** `MaxAge: 2592000` → `MaxAge: 43200` (linha ~181)

```go
// Antes
MaxAge:   2592000, // 30 days

// Depois
MaxAge:   43200, // 12 hours
```

**Verification:**
```bash
grep -n 'MaxAge' /home/ubuntu/docker/Atius/router-ai-atius/main.go
```

### Task 2: Adicionar canais Anthropic (type=14) para MiniMax
**Canal 3:** MiniMax - Anthropic Compatible (modelos MiniMax-M2.7, MiniMax-M2.5)
**Canal 4:** MiniMax-Highspeed - Anthropic Compatible (modelos MiniMax-M2.7-highspeed, MiniMax-M2.5-highspeed)

**SQL:**
```sql
-- Canal 3: MiniMax Anthropic
INSERT INTO channels (name, type, base_url, key, models, status)
VALUES ('MiniMax - Anthropic Compatible', 14, 'https://api.minimax.io', 'YOUR_KEY', 'MiniMax-M2.7,MiniMax-M2.5', 1)
ON CONFLICT DO NOTHING;

-- Canal 4: MiniMax Highspeed Anthropic
INSERT INTO channels (name, type, base_url, key, models, status)
VALUES ('MiniMax-Highspeed - Anthropic Compatible', 14, 'https://api.minimax.io', 'YOUR_KEY', 'MiniMax-M2.7-highspeed,MiniMax-M2.5-highspeed', 1)
ON CONFLICT DO NOTHING;
```

**SQL - Habilidades:**
```sql
-- Habilidades para canal 3
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
VALUES ('default', 'MiniMax-M2.7', 3, true, 0, 0), ('default', 'MiniMax-M2.5', 3, true, 0, 0)
ON CONFLICT ("group", model, channel_id) DO UPDATE SET enabled=true;

-- Habilidades para canal 4
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
VALUES ('default', 'MiniMax-M2.7-highspeed', 4, true, 0, 0), ('default', 'MiniMax-M2.5-highspeed', 4, true, 0, 0)
ON CONFLICT ("group", model, channel_id) DO UPDATE SET enabled=true;
```

**Verification:**
```bash
docker exec db-newapi psql -U admin -d newapi -c "SELECT id, name, type, base_url, models FROM channels ORDER BY id;"
docker exec db-newapi psql -U admin -d newapi -c "SELECT \"group\", model, channel_id, enabled FROM abilities WHERE channel_id IN (3,4);"
```

### Task 3: Rebuild e deploy
```bash
cd /home/ubuntu/docker/Atius/router-ai-atius
docker build --no-cache -t ghcr.io/giovannimnz/router-ai-atius:latest .
docker push ghcr.io/giovannimnz/router-ai-atius:latest
# Deploy
ssh -i ~/.ssh/id_oracle ubuntu@10.1.1.1
sudo docker stop new-api && sudo docker rm new-api
sudo docker compose -f /home/ubuntu/docker/Atius/router-ai-atius/docker-compose.yml up -d
```

### Task 4: Verificar /docs/ após deploy
```bash
curl -s -o /dev/null -w '%{http_code}' https://router.atius.com.br/docs/
```

## Order
1. Task 1 (main.go)
2. Commit
3. Task 2 (SQL - pode ser antes do rebuild)
4. Task 3 (build + deploy)
5. Task 4 (verificação)

## Risks
- Session timeout baixo pode afectar utilizadores activos
- Canal Anthropic type=14 pode não routear correctamente para MiniMax (precisa model_mapping)