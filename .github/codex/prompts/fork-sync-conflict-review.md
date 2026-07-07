Analise esta falha de fork-sync / sync-release em Portugues do Brasil.

Objetivo:
- identificar a causa raiz com base nos arquivos do repo e nos artefatos locais gerados pelo workflow;
- priorizar conflitos de merge, paths protegidos, regressao de build/test e drift com o upstream;
- propor a menor correcao segura para restaurar a automacao.

Contexto obrigatorio:
- o fork e `giovannimnz/router-ai-atius`;
- o upstream e `QuantumNous/new-api`;
- este fork preserva customizacoes locais e nao pode perder seus protected paths;
- quando houver escolha de endpoint para modelos Codex, `Responses` e o padrao.

Instrucoes:
1. Leia `sync-dry-run.json`, `sync-apply.json`, `release-preflight.json` e quaisquer logs gerados no workspace.
2. Verifique os arquivos tocados pelo sync e os arquivos protegidos.
3. Se houver conflitos ou drift, explique quais paths deveriam estar protegidos ou re-portados.
4. Se houver erro de build, cite o arquivo, a linha e a menor correcao defensavel.
5. Responda em PT-BR, de forma objetiva.
6. Nao altere branding protegido do projeto.
