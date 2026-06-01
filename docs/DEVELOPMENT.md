# Atius AI Router — Development

## 1. Environment Setup

### 1.1 Build Go Backend

```bash
# Entrar no container (ou máquina com Go 1.22+)
docker exec -it new-api bash

# Build localmente
cd /app
go build -o new-api .
```

### 1.2 Frontend Dev

```bash
cd web/default

# Instalar dependências (Bun)
bun install

# Dev server
bun run dev

# Typecheck
bun run typecheck

# Build produção
bun run build
```

### 1.3 Python Middleware Dev

```bash
cd integration/middleware

# Ambiente virtual
python3 -m venv venv
source venv/bin/activate

# Instalar deps
pip install -r requirements.txt  # se existir

# Rodar FastAPI dev
uvicorn model_detailed_fastapi:app --reload --port 3001
```

## 2. Project Structure

```
router-ai-atius/
├── controller/          # Handlers (HTTP → Service)
├── service/             # Business logic
├── model/               # GORM models
├── relay/
│   ├── channel/        # Provider adapters
│   │   ├── minimax/
│   │   ├── deepseek/
│   │   └── ...
│   ├── relay_adaptor.go
│   └── api_request.go
├── middleware/          # Gin middleware
├── router/             # Route registration
├── web/
│   └── default/        # React 19 frontend
├── integration/
│   ├── middleware/     # Python FastAPI
│   │   └── model_detailed_fastapi.py
│   └── bruno-tests/   # API tests
└── agent-harness/     # CLI tool
```

## 3. Adding a New Provider Channel

### Step 1: Create Adapter

```bash
# 1. Criar diretório
mkdir -p relay/channel/myprovider

# 2. Criar relay channel struct
# relay/channel/myprovider/adaptor.go
```

### Step 3: Implement Interface

```go
// relay/channel/myprovider/adaptor.go
type Adaptor struct {
    *relay_adaptor.RelayAdaptor
}

func (a *Adaptor) GetRequestURL(model string) string {
    return "https://api.myprovider.com/v1/chat"
}

func (a *Adaptor) ConvertRequest() error {
    // Convert OpenAI format → provider format
}

func (a *Adaptor) DoRequest() error {
    // HTTP call to upstream
}

func (a *Adaptor) DoResponse() error {
    // Parse upstream response
}

func (a *Adaptor) ConvertResponse() ([]byte, error) {
    // Convert provider format → OpenAI format
}
```

### Step 4: Register in relay

```go
// relay/relay.go — adiciona case no switch
case "myprovider":
    return channel_myprovider.NewAdaptor(...)
```

### Step 5: Add StreamOptions Support (se provider suportar streaming)

```go
// Se provider suporta StreamOptions, adicionar ao mapa
// em relay/constant/stream.go
```

## 4. Modifying Middleware

### Python FastAPI (model_detailed_fastapi.py)

```bash
# Editar arquivo
nano integration/middleware/model_detailed_fastapi.py

# Rebuild container
cd integration/middleware
docker build -f Dockerfile.fastapi -t router-ai-atius-model-detailed .

# Restart
docker compose up -d --build model-detailed
```

### Key Endpoints in Middleware

```python
# enrichment.go ou similar
async def enrich_model(model_data: dict) -> dict:
    """Add context_length, pricing, etc to model entry."""
    model_id = model_data.get("id", "")
    
    # Look up from DB or static config
    metadata = MODEL_METADATA.get(model_id, {})
    
    model_data["context_length"] = metadata.get("context_length", 0)
    model_data["max_output_tokens"] = metadata.get("max_output_tokens", 0)
    model_data["pricing"] = metadata.get("pricing", {})
    
    return model_data
```

## 5. Database Changes

### Run Migrations

```bash
# Via docker exec
docker exec -it new-api /app/new-api migrate

# Ou acessar PostgreSQL direto
docker exec -it db-newapi psql -U admin -d newapi
```

### Create New Model

```go
// model/myfeature.go
type MyFeature struct {
    ID        uint      `gorm:"primaryKey"`
    Name      string    `gorm:"size:255;not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (MyFeature) TableName() string {
    return "my_features"
}
```

### Register Migration

```go
// model/main.go — adicionar no array de auto-migrate
db.AutoMigrate(&MyFeature{})
```

## 6. Testing Changes

```bash
# Run all bruno tests
./scripts/run-bruno-tests.sh

# Run specific test
bruno integration/bruno-tests/atius-router-tests/minimax-m27.bru \
  --env .env

# Go tests
docker exec new-api go test ./...

# Frontend tests
cd web/default
bun run test
```

## 7. Build and Deploy

### Build Go binary

```bash
# Cross-compile
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/new-api

# Com version
VERSION=$(git describe --tags)
go build -ldflags="-X main.version=$VERSION" -o dist/new-api
```

### Build Docker image

```bash
# Local
docker build -t ghcr.io/giovannimnz/router-ai-atius:local .

# Push
docker push ghcr.io/giovannimnz/router-ai-atius:local

# Via script
./scripts/deploy-ghcr.sh
```

## 8. Working with Protected Files

### Protected files (never overwritten by upstream sync)

```bash
# Restore if accidentally overwritten
git checkout HEAD -- integration/middleware/model_detailed.py
git checkout HEAD -- docker-compose.yml
```

### Fork-only files (must exist post-merge)

```
.planning/
agent-harness/
integration/bruno-tests/
scripts/run-bruno-tests.sh
.github/workflows/sync.yml
.github/workflows/release.yml
```

## 9. Sync with Upstream

```bash
# Dry-run first
./scripts/sync-fork.sh --dry-run

# Full sync (uses 'theirs' strategy)
./scripts/sync-fork.sh

# With 'ours' strategy (prefer fork changes on conflict)
./scripts/sync-fork.sh --strategy ours

# After sync, restore protected files
git checkout HEAD -- integration/middleware/model_detailed.py
git checkout HEAD -- docker-compose.yml
```

## 10. Key Development Conventions

### JSON Handling

Always use `common/json.go` wrappers:

```go
import "common"

data, err := common.Marshal(v)
err = common.Unmarshal(data, &v)
```

### Database Compatibility

- Use GORM abstractions (no raw `AUTO_INCREMENT`)
- Use `commonGroupCol`, `commonKeyCol` for reserved words
- Use `commonTrueVal`/`commonFalseVal` for booleans
- Test on SQLite, MySQL, PostgreSQL

### Zero Values in Request DTOs

For fields that re-marshal to upstream:

```go
// Use pointer types with omitempty
MaxTokens *int  `json:"max_tokens,omitempty"`
Temperature *float64 `json:"temperature,omitempty"`

// NOT:
// MaxTokens int  // zero values silently dropped
```

---

_Last updated: 2026-05-31_
