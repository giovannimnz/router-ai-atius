# Router DB/Catalog Recovery

## Estado atual

- O runtime live do router usa `SQL_DSN` via host PgBouncer para o banco `DBRouterAiAtius`.
- O unit file ativo aponta para `10.1.1.1:6432/DBRouterAiAtius`.
- O banco live atual continua sendo a fonte de verdade para dados operacionais recentes.
- `users, tokens e logs permanecem vindo do banco live`.
- A recuperacao da Fase 24 nao pode sobrescrever esses dados com dumps antigos.
- O alias legado `newapi` foi removido do PgBouncer em `2026-07-04` depois da validacao do cutover final.

## Fontes de restauracao

### Ranking de autoridade

1. Backup fresco gerado no inicio da janela da Fase 24 a partir do banco live atual.
2. `catalogo 2026-07-01` em `backups/clianything/20260701_184735_channels.sql`, `backups/clianything/20260701_184735_models.sql` e `backups/clianything/20260701_184735_abilities.sql`.
3. Dumps locais anteriores usados apenas para diff, auditoria e rollback de emergencia.

### Uso de cada fonte

- O `catalogo 2026-07-01` e a melhor fonte para restaurar `OpenAI - Codex` e as linhas GPT/Codex ausentes no catalogo atual.
- O banco `DBRouterAiAtius` live atual permanece a fonte de verdade para `users`, `tokens`, `logs`, configuracoes recentes e o estado operacional ja corrigido de `embedding-gte-v1`.
- O dump `/home/ubuntu/.backups/router-ai-atius-incident-20260703T231027-0300/newapi-before.fix.dump` fica reservado para rollback e comparacao, nao para replay cego sobre o banco live.

## Transformacoes obrigatorias

- Restaurar `OpenAI - Codex` e as linhas GPT/Codex permitidas do snapshot de 2026-07-01.
- Restaurar `deepseek-v4-flash` e `deepseek-v4-pro`.
- Restaurar MiniMax de forma consolidada, mas com canal e modelos finais desabilitados.
- Preservar `embedding-gte-v1` como alias publico governado.
- Nao reintroduzir:
  - `gpt-5.4-1m`
  - `gpt-5.5-1m`
  - `text-embedding-3-small`
  - `text-embedding-3-large`
- Nao restaurar `channels.model_mapping` do channel 5 para aliases `-1m`.

## Banco final canonico

- A recuperacao da Fase 24 trabalha com o objetivo de voltar para a identidade canonica `DBRouterAiAtius` no host, sempre via PgBouncer.
- O banco legado `newapi` foi preservado apenas durante a janela de migracao e nao participa mais do runtime live.
- O runtime final fica exclusivamente em `DBRouterAiAtius` via PgBouncer.
- O contrato desta fase e copiar ou restaurar para um destino candidato, nunca mutar cegamente o banco live sem backup fresco validado.

## Backups obrigatorios

Antes de qualquer mutacao:

1. Gerar um dump completo fresco do banco live atual.
2. Gerar um backup catalog-only fresco de `channels`, `models` e `abilities`.
3. Validar os artefatos com `pg_restore -l` quando o formato for archive/custom.
4. Registrar checksums, tamanho e horario dos backups da Fase 24.

Sem esses artefatos, nenhuma restauracao, rename ou repoint de runtime pode comecar.

## Mutacao segura

1. Congelar a verdade operacional:
   - confirmar unit file atual;
   - confirmar contagens atuais de `channels`, `models`, `abilities` e `tokens`;
   - confirmar bancos disponiveis no host.
2. Criar o candidato a banco canonico a partir de backup fresco do live atual.
3. Aplicar reconciliacao de catalogo usando o snapshot de 2026-07-01 com as transformacoes obrigatorias.
4. Validar o candidato antes de qualquer repoint do router:
   - `bin/clianything status --strict`
   - inventario de providers
   - contagens de catalogo
   - verificacao de `embedding-gte-v1`
5. So depois disso repointar PgBouncer e runtime para o banco canonico.
6. Depois da validacao final, remover o alias legado `newapi` do PgBouncer para deixar somente `DBRouterAiAtius` como rota ativa do router.

## Rollback

- Se qualquer gate falhar, o rollback deve usar o backup fresco da Fase 24 como fonte primaria de restauracao.
- O rollback agora e manual e excepcional:
  - reintroduzir temporariamente o alias `newapi` no PgBouncer ou restaurar o backup em `DBRouterAiAtius`;
  - recolocar a unit do router no target anterior somente se houver falha real no banco canonico;
  - repetir as verificacoes de status e catalogo.
- O dump `newapi-before.fix.dump` permanece como referencia secundaria de emergencia, nao substitui o backup fresco da Fase 24.

## Validacao final

- `OpenAI - Codex` volta a existir no catalogo ativo.
- `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini` e `gpt-5.3-codex-spark` aparecem novamente.
- `gpt-5.4-1m` e `gpt-5.5-1m` seguem ausentes.
- `text-embedding-3-small` e `text-embedding-3-large` seguem ausentes.
- DeepSeek permanece ativo de forma consolidada.
- MiniMax permanece restaurado, mas desabilitado.
- `embedding-gte-v1` continua sendo o unico alias publico governado.
- O runtime final continua no host via PgBouncer e alinhado exclusivamente ao banco canonico `DBRouterAiAtius`.
