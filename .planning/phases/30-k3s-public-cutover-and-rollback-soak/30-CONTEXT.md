# Contexto da Phase 30

## Objetivo

Transferir o trafego publico do runtime Podman para a stack k3s validada e fixa
em `atius-srv-1`, observar o comportamento durante o soak e, se todos os gates
passarem, aposentar definitivamente o runtime Podman de producao preservando
artefatos de rollback.

## Gate Obrigatorio

Nao iniciar enquanto a Phase 29 nao entregar:

- `DiskPressure=False` e taint ausente de forma estavel;
- backup fresco e restore real validados;
- PostgreSQL, Redis e router pinados em `atius-srv-1`;
- smoke shadow autenticado fail-closed;
- imagem imutavel, PVC/PV protegidos e backend k3s do CLIAnything validado;
- decisao formal `GO` com hashes, endpoints e procedimentos de rollback.

## Decisoes Vinculantes

- O Apache deve trocar apenas os upstreams do router atualmente em
  `127.0.0.1:3000`; rotas de docs/assets em `127.0.0.1:3003` permanecem intactas.
- O destino sera o `ClusterIP` persistente do Service validado na Phase 29. O IP
  exato e o checksum da configuracao devem entrar na evidencia de cutover.
- Antes da troca, criar backups do vhost, banco, estado k3s, estado Podman e
  configuracoes auxiliares afetadas; validar sintaxe Apache antes e depois.
- Smoke publico deve cobrir health, catalogo de modelos e chamadas autenticadas
  non-stream/stream nos contratos relevantes, distinguindo falha interna de
  falha do upstream.
- O soak deve ter checks periodicos, criterio objetivo de rollback e registro de
  disponibilidade, Pods, restarts, eventos, armazenamento e resposta publica.
- Qualquer gate critico falho reverte imediatamente o Apache para Podman e exige
  smoke de rollback antes de encerrar a tentativa.
- Com soak aprovado, Podman deixa de ser runtime de producao: units/containers
  sao desabilitados e removidos de forma controlada. Imagens, volumes, dumps e
  checksums necessarios ao rollback permanecem preservados pelo periodo definido.
- Nao implementar nem configurar Headroom nesta fase.

## Definicao de Sucesso

- trafego publico servido pelo k3s fixo em `atius-srv-1`;
- soak completo sem regressao critica;
- Podman aposentado como runtime ativo, sem eliminar a capacidade documentada de
  rollback;
- operacao, monitoramento, backup/restore e CLIAnything funcionando no k3s;
- evidencias, documentacao em portugues, commits e push concluidos.
