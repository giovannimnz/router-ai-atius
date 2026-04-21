# CONCERNS.md - Technical Debt, Risks & Code Concerns

## Visão Geral

Este documento identifica preocupações técnicas, riscos e áreas de atenção no codebase NewAPI. As preocupações são organizadas por severidade: **HIGH** (risco imediato), **MEDIUM** (risco potencial), **LOW** (melhorias).

---

## HIGH — Riscos Críticos

### 1. Credenciais Expostas em `.env`

| Arquivo | `integration/.env` |
|---|---|
| **Problema** | API keys e tokens estão em texto puro no repositório |
| **Evidência** | `DEEPSEAK_API_KEY_1=sk-e80eaa8c55ef4eeb84488294f6d21724` e 2 chaves adicionais visíveis |
| **Evidência** | `NEWAPI_ADMIN_TOKEN=sk-vXqhMUmQEAzBOw64yOR8ViddrZBSrK8OrhoDwxHkLOEWYXpQ` exposto |
| **Risco** | Acesso não autorizado a APIs pagas e ao gateway admin |
| **Recomendação** | Usar Docker secrets ou vault externo; adicionar `.env` ao `.gitignore` |

### 2. Credenciais PostgreSQL Fracas

| Arquivo | `.env`, `integration/.env` |
|---|---|
| **Problema** | Usuário `admin` e senha `password123` são credenciais padrão |
| **Evidência** | `POSTGRES_USER=admin` / `POSTGRES_PASSWORD=password123` |
| **Risco** | Acesso direto ao banco via porta 8746 exposta |
| **Recomendação** | Alterar para credenciais fortes e únicas |

### 3. PostgreSQL Exposto na Porta 8746

| Arquivo | `docker-compose.yml` |
|---|---|
| **Problema** | Porta do banco exposta ao host (`8746:5432`) |
| **Evidência** | `ports: - "8746:5432"` |
| **Risco** | Ataque direto ao banco se a porta 8746 for acessível externamente |
| **Recomendação** | Remover mapeamento de porta se não necessário; usar rede Docker interna |

### 4. `sslmode=disable` no DSN

| Arquivo | `.env`, `integration/.env` |
|---|---|
| **Problema** | Conexão ao banco sem SSL |
| **Evidência** | `SQL_DSN=postgres://admin:password123@db-newapi:5432/newapi?sslmode=disable` |
| **Risco** | Dados trafegam em texto puro (mitigado por rede Docker interna, mas ainda vulnerável) |
| **Recomendação** | Habilitar SSL se o banco for acessível externamente |

### 5. Sem `.gitignore` para Arquivos Sensíveis

| Arquivo | Raiz do projeto |
|---|---|
| **Problema** | `.env` e `data/` podem ser commitados acidentalmente |
| **Evidência** | Nenhum `.gitignore` encontrado no projeto |
| **Risco** | Vazamento de credenciais via git |
| **Recomendação** | Criar `.gitignore` com `.env`, `data/`, `backups/`, `.planning/` |

### 6. Arquivo Legacy `one-api.db`

| Arquivo | `data/one-api.db` |
|---|---|
| **Problema** | SQLite legado coexistindo com PostgreSQL |
| **Evidência** | Arquivo presente em `data/one-api.db` |
| **Risco** | Confusão sobre qual banco é a fonte da verdade; dados des sincronizados |
| **Recomendação** | Verificar se ainda é usado; remover se for resíduo de migração |

---

## MEDIUM — Riscos Potenciais

### 7. Sem Healthcheck para NewAPI

| Arquivo | `docker-compose.yml` |
|---|---|
| **Problema** | PostgreSQL tem healthcheck, mas NewAPI não |
| **Evidência** | `healthcheck` definido apenas em `db-newapi` |
| **Risco** | NewAPI pode estar indisponível sem que o Docker detecte |
| **Recomendação** | Adicionar healthcheck com `curl -f http://localhost:3000/health` |

### 8. Imagem Docker sem Pin de Versão

| Arquivo | `docker-compose.yml` |
|---|---|
| **Problema** | Usa `calciumion/new-api:latest` sem versão específica |
| **Evidência** | `image: calciumion/new-api:latest` |
| **Risco** | Updates automáticos podem introduzir breaking changes |
| **Recomendação** | Pinar versão específica (ex: `calciumion/new-api:0.5.0`) |

### 9. Scripts sem Validação de Input

| Arquivo | `management.sh`, `backup-restore.sh` |
|---|---|
| **Problema** | Scripts aceitam input do usuário sem validação |
| **Evidência** | `read -p "Digite o nome completo do arquivo..." BACKUP_FILE` sem sanitização |
| **Risco** | Path traversal ou execução de comandos via input malicioso |
| **Recomendação** | Validar e sanitizar inputs do usuário |

### 10. Dependência de Rede Externa sem Fallback

| Arquivo | `integration/docker-compose.yml` |
|---|---|
| **Problema** | Rede `atius-shared` é externa e pode não existir |
| **Evidência** | `atius-shared: external: true` |
| **Risco** | Falha no deploy se a rede externa não estiver configurada |
| **Recomendação** | Documentar pré-requisitos ou criar rede automaticamente |

### 11. Sem Rotação Automática de API Keys

| Arquivo | `integration/.env` |
|---|---|
| **Problema** | 3 chaves DeepSeek manuais sem automação de rotação |
| **Evidência** | `DEEPSEAK_API_KEY_1/2/3` fixas no `.env` |
| **Risco** | Chaves expiram (uma já expira em 2026-04-13) e serviço para |
| **Recomendação** | Automatizar rotação ou alertar antes da expiração |

### 12. Comentários Indicando Chave Expirada

| Arquivo | `integration/.env` |
|---|---|
| **Problema** | Comentário `giovannimunizds - Expires on 2026-04-13` |
| **Evidência** | Próximo às chaves DeepSeek |
| **Risco** | Chave pode expirar e causar interrupção |
| **Recomendação** | Renovar antes da expiração; configurar alertas |

---

## LOW — Melhorias e Boas Práticas

### 13. Sem Testes Automatizados

| Arquivo | Todo o projeto |
|---|---|
| **Problema** | Nenhum teste automatizado (unitário, integração, E2E) |
| **Evidência** | Apenas scripts manuais de verificação |
| **Risco** | Regressões não detectadas |
| **Recomendação** | Adicionar `test_all_models.sh` como cron job ou CI step |

### 14. Sem CI/CD Pipeline

| Arquivo | Todo o projeto |
|---|---|
| **Problema** | Deploys são manuais via scripts |
| **Evidência** | Nenhum `.github/workflows/`, `Jenkinsfile`, etc. |
| **Risco** | Inconsistência entre ambientes; deploys esquecem steps |
| **Recomendação** | Pipeline simples para validar health pós-deploy |

### 15. Duplicação entre `docker-compose.yml`

| Arquivo | Raiz vs `integration/` |
|---|---|
| **Problema** | Dois arquivos `docker-compose.yml` com definições similares |
| **Evidência** | Ambos definem `new-api` e `db-newapi` com configs quase idênticas |
| **Risco** | Divergência entre configs causa comportamento inconsistente |
| **Recomendação** | Unificar em um único arquivo ou usar overrides |

### 16. Scripts em Português e Inglês Misturados

| Arquivo | Vários scripts |
|---|---|
| **Problema** | Mensagens em português, variáveis em inglês |
| **Evidência** | `echo "Aplicação NewAPI iniciada com sucesso!"` vs `set -e`, `SCRIPT_DIR` |
| **Risco** | Confusão para colaboradores não lusófonos |
| **Recomendação** | Padronizar idioma (sugestão: inglês para código, português para UX) |

### 17. Sem Logging Estruturado

| Arquivo | `data/logs/` |
|---|---|
| **Problema** | Logs da aplicação não são estruturados (JSON) |
| **Evidência** | Diretório `data/logs/` existe mas sem formato definido |
| **Risco** | Dificuldade de análise e monitoramento |
| **Recomendação** | Configurar logging em JSON se o NewAPI suportar |

### 18. Sem Backup Automatizado

| Arquivo | `backup-restore.sh` |
|---|---|
| **Problema** | Backup é manual via script interativo |
| **Evidência** | Menu interativo sem opção de agendamento |
| **Risco** | Perda de dados se backup for esquecido |
| **Recomendação** | Cron job para backup diário do PostgreSQL |

---

## Resumo por Categoria

| Categoria | HIGH | MEDIUM | LOW |
|---|---|---|---|
| **Segurança** | 4 | 0 | 0 |
| **Infraestrutura** | 1 | 3 | 2 |
| **Código** | 0 | 2 | 3 |
| **Operacional** | 0 | 2 | 3 |
| **Total** | **5** | **7** | **8** |

## Ações Imediatas Recomendadas

1. **Criar `.gitignore`** para proteger `.env`, `data/`, `backups/`
2. **Alterar credenciais PostgreSQL** para senhas fortes
3. **Remover/verificar `data/one-api.db`** legado
4. **Adicionar healthcheck** ao serviço NewAPI
5. **Pinar versão** da imagem `calciumion/new-api`
6. **Configurar alerta** para expiração de chaves DeepSeek (2026-04-13)
