# Router AI Atius + Antigravity

## Objetivo

Criar um novo channel no Atius capaz de utilizar o runtime oficial do Antigravity (`agy`) como backend, com suporte a múltiplas contas/runtimes isolados, sem reutilizar diretamente tokens internos nem chamar endpoints privados não documentados.

## Princípios

- O Atius atua como orquestrador.
- O `agy` oficial continua sendo a fronteira de autenticação e execução.
- Não extrair/reutilizar tokens do Antigravity fora do runtime oficial.
- Não depender de endpoint privado do Google.
- Manter cada conta isolada.
- Permitir múltiplos runtimes dentro de um único channel Antigravity.
- Não usar múltiplas contas para burlar limites; o objetivo é isolamento, continuidade operacional e uso legítimo de contas distintas.

## Arquitetura proposta

```text
Cliente
  |
  v
Atius
  |
  +--> Headroom / Context Optimization
  |
  v
Antigravity Channel
  |
  v
Runtime Account Pool
  |
  +--> Conta A -> container/runtime agy A
  +--> Conta B -> container/runtime agy B
  +--> Conta C -> container/runtime agy C
```

## Channel Antigravity

Criar uma abstração própria para runtimes, em vez de tentar representar cada conta como uma simples API key.

Exemplo conceitual:

```ts
interface RuntimeAccount {
  id: string;
  status: "ready" | "busy" | "unhealthy" | "offline";
  runtimeType: "agy";
  transport: "stdio" | "socket";
  containerId?: string;
  lastUsedAt?: Date;
  healthScore?: number;
}

interface RuntimePool {
  acquire(request: RuntimeRequest): Promise<RuntimeAccount>;
  release(accountId: string): Promise<void>;
  healthCheck(accountId: string): Promise<void>;
}
```

## Seleção de conta/runtime

O pool pode considerar:

- saúde do processo;
- disponibilidade;
- projeto;
- sessão;
- afinidade de conversa;
- capacidade do runtime;
- limites administrativos definidos pelo usuário;
- preferência manual;
- fallback em caso de falha.

Evitar modelar isso como rotação cega de credenciais.

## Podman

Abordagem preferencial:

- uma única imagem base;
- múltiplos containers rootless;
- um container por conta;
- cada conta com `HOME`, configuração, credenciais e storage próprios;
- imagem read-only compartilhada entre containers;
- volumes separados por conta;
- comunicação local por Unix socket quando possível;
- sem `systemd` dentro do container, salvo necessidade real.

Exemplo:

```text
antigravity-base:latest
  |
  +--> agy-account-a
  |      /home/agy -> volume_a
  |
  +--> agy-account-b
  |      /home/agy -> volume_b
  |
  +--> agy-account-c
         /home/agy -> volume_c
```

## Estimativa de footprint

Valores apenas de ordem de grandeza até haver benchmark real:

- imagem compartilhada: algumas centenas de MB;
- writable layer por container: pequeno, desde que caches sejam controlados;
- volume por conta: depende de histórico/configuração/cache;
- RAM por runtime ativo: provavelmente dezenas a poucas centenas de MB, dependendo da implementação do CLI e do workload.

O custo de disco não cresce linearmente com o número de containers porque as camadas da imagem são compartilhadas.

## Continuidade de conversa

Não depender do SQLite interno do `agy` como memória canônica.

Motivos:

- sessão pode estar vinculada à conta;
- compartilhar SQLite entre contas pode ser instável;
- formato interno pode mudar;
- cria acoplamento excessivo.

Direção preferida:

```text
Atius -> histórico canônico -> Headroom -> contexto otimizado -> agy
```

Quando houver troca de runtime/conta, o Atius reconstrói o contexto necessário para a nova sessão.

## Headroom

O Headroom fica como camada de otimização de contexto antes do channel.

Responsabilidades:

- compressão;
- redução de contexto redundante;
- reaproveitamento de informação;
- preparação de contexto para troca de runtime.

Ele não deve manipular credenciais do Antigravity.

```text
Request
  -> Atius
  -> Headroom
  -> Antigravity Channel
  -> Runtime Pool
  -> agy
```

## Segurança e risco de conta

Abordagem mais conservadora:

- somente binário/runtime oficial;
- autenticação feita pelo mecanismo oficial;
- sem scraping de tokens;
- sem reprodução de endpoints internos;
- sem tentativa de contornar quotas;
- isolamento por conta;
- auditoria de requisições.

Como houve histórico prévio de suspensão ao usar uma conta Google com ferramenta de terceiros, esta arquitetura deve privilegiar interfaces explicitamente suportadas e evitar qualquer mecanismo de autenticação improvisado.

## Próximos passos

1. Validar o modo headless oficial atual do `agy`.
2. Validar transporte disponível: stdin/stdout, socket ou outro.
3. Criar `RuntimeAdapter`.
4. Criar `AntigravityRuntimeAdapter`.
5. Criar `RuntimeAccountPool`.
6. Implementar health check.
7. Implementar isolamento via Podman.
8. Adicionar Headroom como middleware.
9. Criar benchmark de RAM, disco e latência.
10. Definir política explícita de seleção/fallback.
