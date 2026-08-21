# Contexto da Phase 30

## Objetivo

Transferir o trafego publico do router/Redis Podman e a database logica
`DBRouterAiAtius` do PostgreSQL 17 host para a stack k3s validada e fixa em
`atius-srv-1`, observar o comportamento durante o soak e, se todos os gates
passarem, aposentar separadamente os backends antigos sem destruir os artefatos
de rollback.

## Topologia Canonica Descoberta Live

- O PgBouncer host em `10.11.1.11:6432` mapeia `DBRouterAiAtius` para
  `127.0.0.1:8745`.
- `127.0.0.1:8745` e o PostgreSQL 17 host, cluster
  `/var/lib/postgresql/17/main`, administrado por `postgresql@17-main`; essa e a
  fonte canonica de `DBRouterAiAtius`, com 34 tabelas.
- O PostgreSQL em container Podman possui uma `DBRouterAiAtius` vazia, com 0
  tabelas. Ele nao participa do cutover de dados nem do rollback de dados.
- O cutover de DB altera somente o mapping `DBRouterAiAtius`, do PostgreSQL 17
  host para o Service PostgreSQL 17 k3s. O rollback restaura esse mapping para
  `127.0.0.1:8745`.

## Gate Obrigatorio

Nao iniciar enquanto a Phase 29 nao entregar:

- `DiskPressure=False` e taint ausente de forma estavel;
- backup fresco e restore real validados;
- PostgreSQL, Redis e router pinados em `atius-srv-1`;
- smoke shadow autenticado fail-closed;
- imagem imutavel, PVC/PV protegidos e backend k3s do CLIAnything validado;
- decisao formal `GO` com hashes, endpoints e procedimentos de rollback.

## Decisoes Vinculantes

- **D-01:** O Apache deve trocar apenas os upstreams do router atualmente em
  `127.0.0.1:3000`; rotas de docs/assets em `127.0.0.1:3003` permanecem intactas.
- **D-02:** O destino sera o `ClusterIP` persistente do Service validado na Phase 29. O IP
  exato e o checksum da configuracao devem entrar na evidencia de cutover.
- **D-03:** Antes da troca, criar backups do vhost, da fonte canonica
  `DBRouterAiAtius` no PostgreSQL 17 host, dos metadados do cluster PG17 host,
  do estado k3s, do estado Podman e das configuracoes auxiliares afetadas;
  validar sintaxe Apache antes e depois.
- **D-04:** Smoke publico deve cobrir health, catalogo de modelos e chamadas autenticadas
  non-stream/stream nos contratos relevantes, distinguindo falha interna de
  falha do upstream.
- **D-05:** O soak deve ter checks periodicos, criterio objetivo de rollback e registro de
  disponibilidade, Pods, restarts, eventos, armazenamento e resposta publica.
- **D-06:** Qualquer gate critico falho reverte imediatamente o Apache para o
  router Podman e o mapping `DBRouterAiAtius` para o PostgreSQL 17 host em
  `127.0.0.1:8745`; o smoke de rollback usa essa combinacao antes de encerrar a
  tentativa.
- **D-07:** Com soak aprovado, aposentar as units/containers Podman de
  router/Redis e retirar `DBRouterAiAtius` como source no PostgreSQL 17 host.
  A auditoria live confirmou que `postgresql@17-main` e compartilhado por ATS,
  Horistic, GBrain, Omni Fleet e outras databases; portanto o service deve
  permanecer active/enabled nesta fase. Nunca apagar data dir, databases, dumps,
  imagens, volumes ou units; reter tudo por no minimo 7 dias.
- **D-08:** Nao implementar nem configurar Headroom nesta fase.
- **D-09:** Toda operacao pesada respeita CPU total <=20%; cada Pod normal tem
  requests/limits de 500m; segredos sao carregados exclusivamente do Vault e
  nunca entram em logs ou evidencias.
- **D-10:** O container PostgreSQL Podman vazio nunca e fonte, destino ou
  rollback de dados; qualquer evidencia que nao prove 34 tabelas na fonte host
  e 0 tabelas no container invalida o preflight.

## Definicao de Sucesso

- trafego publico servido pelo k3s fixo em `atius-srv-1`;
- soak completo sem regressao critica;
- router/Redis Podman aposentados como runtime ativo e PostgreSQL 17 host
  preservado active/enabled por ser compartilhado, sem eliminar a capacidade
  documentada de rollback;
- operacao, monitoramento, backup/restore e CLIAnything funcionando no k3s;
- evidencias, documentacao em portugues, commits e push concluidos.
