# Atius AI Router — Configuration

## 1. Environment Variables

### 1.1 .env File

```bash
# Banco de dados
POSTGRES_USER=admin
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=newapi

# Security
SESSION_SECRET=your_32_char_secret

# Proxy (se atrás de reverse proxy)
TRUST_PROXY=true

# Timezone
TZ=America/Sao_Paulo
LANG=pt_BR.UTF-8
```

### 1.2 Docker Compose Variables

```bash
# docker-compose.yml overrides
SQL_DSN=postgres://admin:${POSTGRES_PASSWORD}@db-newapi:5432/newapi?sslmode=disable
```

## 2. Database Configuration

### 2.1 PostgreSQL Connection

```bash
# Connection string format
SQL_DSN=postgres://USER:PASSWORD@HOST:PORT/DATABASE?sslmode=DISABLE

# Production
SQL_DSN=postgres://admin:password@db-newapi:5432/newapi?sslmode=disable
```

### 2.2 Database Tables

| Table | Config Via | Description |
|-------|-----------|-----------|
| `channels` | Admin UI / API | Provider configs |
| `models` | Admin UI / API | Model definitions |
| `abilities` | Admin UI / API | Channel ↔ Model mapping |
| `tokens` | Admin UI / API | User API tokens |
| `options` | Admin UI / API | `api_info` JSON, settings |
| `channel_group` | Admin UI | Channel grouping |

## 3. Channel Configuration

### 3.1 Via SQL

```sql
-- Insert MiniMax channel
INSERT INTO channels (id, name, group_name, base_url, key, status)
VALUES (3, 'MiniMax - Anthropic Compatible', '', 'https://api.minimax.io', 'your_key_here', 1);

-- Insert DeepSeek channel
INSERT INTO channels (id, name, group_name, base_url, key, status)
VALUES (2, 'DeepSeek API', '', 'https://api.deepseek.com', 'your_key_here', 1);

-- Verify
SELECT id, name, base_url FROM channels;
```

### 3.2 Via Admin UI

```
https://router.atius.com.br
→ Channels → Add Channel
→ Fill: name, base_url, key, abilities
```

### 3.3 Ability Mapping

```sql
-- Map model to channel
INSERT INTO abilities (channel_id, ability)
VALUES (3, 'MiniMax-M2.7'), (3, 'MiniMax-M2.5');
```

## 4. Model Configuration

### 4.1 Model Metadata (via DB)

```sql
-- Update model context length
UPDATE models
SET context_length = 245760
WHERE name = 'MiniMax-M2.7';
```

### 4.2 Model Mapping (aliases)

Model aliases stored in `channels.model_mapping` column as JSON:

```json
{
  "MiniMax-M2.7-hs": "MiniMax-M2.7-highspeed",
  "MiniMax-M2.5-hs": "MiniMax-M2.5-highspeed"
}
```

### 4.3 Middleware Enrichment

Model enrichment configured in `integration/middleware/model_detailed_fastapi.py`:

```python
MODEL_METADATA = {
    "MiniMax-M2.7": {
        "context_length": 245760,
        "max_output_tokens": 50000,
        "pricing": {
            "prompt": "0.30",
            "completion": "1.20",
        }
    }
}
```

## 5. Rate Limiting

### 5.1 Configure Global Limits

```bash
# Via setting/rate_limit.go or DB
# RPM = requests per minute
# TPM = tokens per minute
```

### 5.2 Per-User Quotas

Managed via `service/quota.go` and `service/channel_select.go`.

### 5.3 Check Current Usage

```bash
docker exec db-newapi psql -U admin -d newapi -c \
  "SELECT * FROM user_usage LIMIT 5;"
```

## 6. Middleware Configuration

### 6.1 model-detailed (Python FastAPI)

```bash
# environment variables in docker-compose
MIDDLEWARE_PORT=3001
NEWAPI_BACKEND_URL=http://new-api:3000
BACKEND_TIMEOUT=60
```

### 6.2 Modify Enrichment Logic

```bash
# Edit source
nano integration/middleware/model_detailed_fastapi.py

# Rebuild
docker compose up -d --build model-detailed
```

## 7. Docker Compose Customization

### 7.1 CPU Limits

```yaml
# docker-compose.yml
services:
  new-api:
    deploy:
      resources:
        limits:
          cpus: '0.5'
  model-detailed:
    deploy:
      resources:
        limits:
          cpus: '0.1'
  db-newapi:
    deploy:
      resources:
        limits:
          cpus: '0.5'
```

### 7.2 Port Mapping

```yaml
services:
  new-api:
    ports:
      - "3301:3000"    # host:container
  model-detailed:
    ports:
      - "3300:3001"
  db-newapi:
    ports:
      - "8746:5432"
```

### 7.3 Volume Persistence

```yaml
volumes:
  ./data:/data           # new-api data
  ./db-data:/var/lib/postgresql/data   # PostgreSQL data
```

## 8. Network Configuration

### 8.1 Networks

```yaml
networks:
  atius-shared:    # 192.168.0.0/20
  newapi-internal: # 172.20.0.0/16
```

### 8.2 DNS Resolution

```bash
# From new-api container
curl http://model-detailed:3001/v1/models
curl http://db-newapi:5432
```

## 9. Billing Configuration

### 9.1 Ratio Settings

```bash
# setting/ratio_setting/ — model pricing ratios
# Loaded from DB options table
```

### 9.2 Expression-Based Billing

See `pkg/billingexpr/expr.md` for advanced expression-based pricing.

## 10. Auth Configuration

### 10.1 JWT Settings

```go
# common/jwt.go or config
- SESSION_SECRET env var
- JWT expiry configured in setting/
```

### 10.2 OAuth Providers

OAuth configs in `oauth/` directory:
- GitHub
- Discord
- OIDC
- Linux.do

## 11. API Info (Dashboard)

The `api_info` shown in `/api/status` is stored in DB `options` table:

```sql
-- View current api_info
SELECT key, value FROM options WHERE key = 'api_info';
```

Customize via Admin UI → System Settings → API Info.

## 12. Version File

```bash
cat VERSION  # 0.12.14.2
```

Update via:

```bash
./scripts/version-bump.sh
```

---

_Last updated: 2026-05-31_
