# Contexto da Phase 29

## Objetivo

Preparar e validar em shadow a substituicao completa do runtime Podman pelo k3s
em `atius-srv-1`: PostgreSQL, Redis e `router-ai-atius`. A fase termina apenas
com restore real, smoke autenticado e decisao go/no-go documentada, sem alterar
o trafego publico.

## Decisoes Vinculantes

- Todos os Pods devem usar afinidade obrigatoria para um label dedicado presente
  somente em `atius-srv-1`; nao basta preferencia de scheduler ou hostname solto.
- O `DiskPressure` deve ser resolvido por limpeza segura. E proibido mascarar o
  problema com toleration, remover a taint manualmente ou alterar thresholds de
  eviction apenas para permitir o rollout.
- O gate exige pelo menos 20 GiB de recuperacao segura, alvo de 25% de espaco
  livre, `DiskPressure=False` e ausencia da taint por no minimo cinco minutos.
- A storage class deve ser explicitamente `local-path`. Os PVCs precisam ficar
  vinculados ao node correto e os PVs devem receber politica `Retain` depois do
  bind, antes de qualquer etapa destrutiva.
- O apply deve ser estagiado: namespace/config/secret/PostgreSQL; restore e
  validacao; Redis; router. O router nao pode iniciar contra banco vazio.
- A fonte canonica de `DBRouterAiAtius` e o PostgreSQL 17 nativo do host em
  `127.0.0.1:8745`, cluster `/var/lib/postgresql/17/main`, administrado por
  `postgresql@17-main`. O PgBouncer em `10.11.1.11:6432` serve somente para
  cross-check de identidade e invariantes; o PostgreSQL Podman esta vazio e
  nunca pode ser origem de backup ou restore.
- O backup deve usar `pg_dump` 17 diretamente contra o PostgreSQL 17 do host e
  o restore deve terminar em PostgreSQL 17 no k3s.
- O target k3s deve ser provadamente integralmente limpo antes da importacao. O
  restore e atomico, falha no primeiro erro e so admite retry apos um `no-go`
  explicito, preservando a evidencia anterior.
- Secrets reais sao criados fora do Git a partir das fontes operacionais atuais;
  nenhum valor pode aparecer em logs, evidencias, planos ou commits.
- A imagem deve ser imutavel e importada no containerd do k3s; e proibido usar
  `latest` flutuante como identidade do rollout validado.
- O shadow deve usar `ClusterIP`. A auditoria live provou que o Apache do host
  alcanca a rede de Services; nao usar Ingress, hostPort ou NodePort sem um novo
  bloqueio tecnico comprovado.
- O smoke e o go gate sao fail-closed: token ausente, endpoint nao autenticado,
  restore incompleto ou evidencia ausente resultam em no-go.
- A evidencia de cleanup e historica, mas deve pertencer ao cluster atual. Cada
  preflight revalida por cinco minutos o estado atual de espaco, DiskPressure e
  taint. A evidencia de bootstrap deve estar fresca, pertencer ao cluster atual
  e corresponder exatamente ao hash dos manifests usados na execucao.
- Antes de aposentar Podman, o CLIAnything deve possuir backend k3s validado para
  operacao do banco sem dependencia de `podman exec`.

## Evidencia Atual

- `atius-srv-1` esta Ready, mas apresenta `DiskPressure=True` e taint
  `node.kubernetes.io/disk-pressure:NoSchedule`.
- O Metrics API esta funcional; o bloqueio antigo correspondente esta obsoleto.
- Existe apenas `local-path`, suficiente para esta topologia single-node desde
  que pinning, Retain e backup/restore sejam tratados explicitamente.
- Nao existe IngressClass e ela nao e necessaria para o desenho escolhido.
- O trafego publico continua no Podman via Apache em `127.0.0.1:3000`.
- O backup antigo citado no planejamento nao e suficiente; a fase exige dump
  fresco e validado antes do restore.
- Operacoes pesadas permanecem limitadas a no maximo 20% da CPU do host; cada
  container dos Pods gerenciados deve pedir e limitar exatamente `500m` de CPU.
  Secrets continuam vindo do Vault sem valores em arquivos versionados ou
  evidencias.

## Definicao de Sucesso

- espaco e pressao do node estabilizados;
- stack completa restaurada e executando em shadow no k3s de `atius-srv-1`;
- dados, health, modelos e chamadas autenticadas validados;
- rollback Podman preservado e testavel;
- artefato explicito de go para a Phase 30, ou no-go objetivo sem tocar trafego
  publico.
