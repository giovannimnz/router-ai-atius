# Codex CI, auth e release - alinhamento oficial

Data de referência: 2026-07-08.

Este documento fixa o contrato oficial da trilha Codex para automação neste
fork. Ele existe para evitar drift entre a operação local do `router-ai-atius`
e a documentação oficial da OpenAI/Codex.

## Fonte de verdade

As referências autoritativas desta trilha são as páginas oficiais da OpenAI:

- `https://developers.openai.com/codex/noninteractive`
- `https://developers.openai.com/codex/github-action`
- `https://developers.openai.com/codex/auth/ci-cd-auth`
- `https://developers.openai.com/api/docs/guides/tools-connectors-mcp`
- `https://developers.openai.com/codex/sdk`

Repos comunitários, exemplos locais e notas históricas podem informar contexto,
mas não substituem essas referências quando o tema é:

- `codex exec`
- `openai/codex-action`
- auth em CI/CD
- Docs MCP / OpenAI docs
- uso do Codex SDK

## Regra prática

### GitHub Actions

Quando o Codex rodar dentro do GitHub Actions, o caminho preferido é:

- usar `openai/codex-action@v1`
- não instalar o CLI manualmente em shell step quando o job puder usar a action
- manter `safety-strategy: drop-sudo` em Linux/macOS

Motivo operacional:

- a action instala o CLI
- sobe o proxy da Responses API quando recebe API key
- encapsula melhor a exposição da credencial do que um shell step simples

### Outras automações

Fora do GitHub Actions, o caminho preferido é:

- `codex exec`

Exemplo:

```bash
CODEX_API_KEY="$OPENAI_API_KEY" \
codex exec --json --sandbox workspace-write "revise este diretório e gere um resumo"
```

## Auth em automação

### Default recomendado

A OpenAI recomenda API key como default para automação.

Neste fork, isso significa:

- em GitHub Actions, preferir `openai/codex-action` com `openai-api-key`
- fora do GitHub Actions, usar `CODEX_API_KEY` inline na invocação de
  `codex exec`

### O que não fazer

Não definir `OPENAI_API_KEY` ou `CODEX_API_KEY` como env de job amplo quando o
job executa código controlado pelo repositório, porque scripts, tests, hooks ou
ações comprometidas no mesmo job podem ler essas variáveis.

### ChatGPT-managed auth em CI/CD

`auth.json` com `auth_mode: chatgpt` é trilha avançada e restrita:

- só em runner privado/confiável
- só quando realmente for necessário executar como conta Codex/ChatGPT
- nunca como default para automação
- nunca para repositório público/open-source

Contrato oficial:

- sem refresh custom em job
- o padrão suportado é deixar o próprio Codex refrescar `auth.json`
- persistir o `auth.json` atualizado entre runs
- sem reescrever o arquivo original a cada job

Tratamento obrigatório:

- `~/.codex/auth.json` é segredo
- não commitar
- não copiar para docs
- não colar em issues/chat/logs

## Docs MCP / OpenAI docs

Quando o tema for OpenAI API, Responses API, Codex, SDK ou CI/auth do Codex:

- usar OpenAI Docs MCP / `openaiDeveloperDocs` como lookup primário
- usar páginas oficiais da OpenAI como fallback

Isso vale para:

- geração de docs locais
- revisão de workflows
- troubleshooting de auth/automation

## Mapeamento deste fork

### Workflow atual com Codex

Arquivo:

- `.github/workflows/sync.yml`

Uso atual:

- `openai/codex-action@v1` entra como helper de análise de falha
- usa prompt versionado em `.github/codex/prompts/fork-sync-conflict-review.md`
- grava `codex-sync-analysis.md` como artefato

Contrato desejado:

- inputs oficiais da action sempre que existirem (`model`, `effort`, `sandbox`,
  `output-file`, `safety-strategy`)
- `codex-args` apenas para flags realmente extras, não para substituir inputs
  oficiais

### Workflow de release

Arquivo:

- `.github/workflows/release.yml`

Contrato local:

- release notes do fork continuam PT-BR-first
- isso já está alinhado com a política local do fork
- a trilha de release não precisa usar Codex para ser considerada correta

## Checklist operacional

Antes de aprovar mudança em CI/auth/release envolvendo Codex:

1. Confirmar se a mudança é baseada em docs oficiais da OpenAI.
2. Confirmar se GitHub Actions usa `openai/codex-action` quando aplicável.
3. Confirmar se API key continua sendo o default.
4. Confirmar se qualquer menção a `auth.json` está limitada a runner privado e
   nunca a repo público/open-source.
5. Confirmar que a documentação publicada do fork continua em PT-BR.

## Validação local desta fase

Pontos mínimos:

```bash
python3 - <<'PY'
from pathlib import Path
import yaml
for rel in [
    ".github/workflows/sync.yml",
    ".github/workflows/release.yml",
]:
    yaml.safe_load(Path(rel).read_text(encoding="utf-8"))
print("workflow yaml ok")
PY

rg -n "codex exec|openai/codex-action|API key|auth.json|Docs MCP" \
  docs/CODEX-CI-AUTH-RELEASE.md \
  docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md \
  docs/CI-RELEASE-WATCHDOG.md
```
