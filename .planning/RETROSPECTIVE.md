# Retrospectiva do Projeto

## Milestone: v2.17 — Codex OAuth lifecycle and upstream auth diagnostics

**Shipped:** 2026-07-12
**Phases:** 1 | **Plans:** 4

### What Was Built

- Ciclo OAuth Router-owned com Authorization Code + PKCE, refresh token próprio, probe e renovação.
- Diagnóstico explícito de auth upstream Codex separado da API key interna do Router.
- UI type 57 dedicada, sem Base URL/API Key genéricos e com recuperação operacional clara.
- Proteção de fork-sync, documentação PT-BR e validação local/pública do catálogo e inferência.

### What Worked

- A validação em camadas, código, DB canônico, runtime e endpoint público, isolou rapidamente o banco homônimo errado.
- Testes determinísticos cobriram o negativo upstream sem corromper a credencial live recém-regenerada.
- O wrapper de recursos manteve build e suites pesadas dentro do limite de 20% da CPU total.

### What Was Inefficient

- Cache Go compartilhado e wrappers globais de compilador causaram falhas ambientais; `GOCACHE` isolado e `CGO_ENABLED=0` resolveram o gate.
- Graphify possui efeito autorreferente: commitar o grafo muda o HEAD e torna o snapshot recém-commitado tecnicamente anterior ao commit.
- O archiver inferiu `current_phase: 17` de `v2.17` e não extraiu accomplishments; ambos exigiram correção manual.

### Patterns Established

- Validar credenciais contra o DSN efetivo do container, nunca contra defaults de CLI host.
- Tratar import de access token do Codex CLI apenas como break-glass, sem compartilhar refresh token.
- Não induzir falha destrutiva em produção quando testes determinísticos cobrem a taxonomia negativa.

### Key Lessons

- Expiração futura não prova saúde de OAuth; probe upstream e capacidade de refresh fazem parte do estado válido.
- Artefatos históricos também precisam de secret scanning antes de archive e milestone close.
- UI de provider especializado deve esconder configurações genéricas que não fazem parte do contrato real.

### Cost Observations

- Model mix e custo de sessões não foram medidos por este workflow.
- A maior espera foi Graphify sob CPU limitada; as validações estreitas evitaram rebuilds desnecessários.

## Cross-Milestone Trends

| Milestone | Quality | Operational pattern |
|-----------|---------|---------------------|
| v2.17 | 6/6 requisitos, Nyquist 6/6, security 14/14 | Validar código, DB canônico, runtime e público separadamente |
