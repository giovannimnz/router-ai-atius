# 2026-07-10 - GPT-5.6 Codex explore

## Pergunta

Como a chegada de `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna` impacta o `router-ai-atius`?

## Resposta curta

- Discovery dinamica do channel 5 deve conseguir ver novos slugs.
- O fork ainda esta preso em policy/docs/defaults de `gpt-5.5` e `gpt-5.4`.
- O gap principal e de contrato de API:
  - `reasoning.mode`
  - `reasoning.context`
  - `prompt_cache_options`
  - `prompt_cache_breakpoint`
  - `allowed_callers`
  - `output_schema`
  - tipos `program` / `program_output`

## Arquivo principal

- [docs/OPENAI-GPT-5.6-CODEX-RESEARCH-2026-07-10.md](/home/ubuntu/GitHub/containers/router-ai-atius/docs/OPENAI-GPT-5.6-CODEX-RESEARCH-2026-07-10.md)

## Proximo passo sugerido

- quando quiser transformar isso em execucao: abrir uma fase nova ou estender a trilha Codex atual com implementacao e validacao live dos `gpt-5.6-*`
