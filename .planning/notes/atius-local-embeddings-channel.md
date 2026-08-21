# Atius Local Embeddings channel

## Decision

Consolidate local GTE embeddings and reranking in one first-class router provider:

- channel `11`, type `59`, `Atius Local`
- `embedding-gte-v1` routes `/v1/embeddings` to the two-pod HA Service at `http://10.21.1.21:31115/v1/embeddings`
- `reranker-gte-v1` routes `/v1/rerank` to `http://10.21.1.21:31216/rerank`
- reranking uses `jina_rerank_to_tei_native`; both upstream routes use auth `none`
- local models remain hosted on `horistic-srv`
- embeddings use pinned model revision `9bbca17d9273fd0d03d5725c7a4b0f6b45142062`; redundancy is pod/process HA on the single required host

## Production state

- final image: `v2.17.3-atius-local-embeddings.3`
- canonical public Podman runtime and parallel k3s deployment are aligned
- legacy channels `9` and `10` were deleted only after both models passed with them disabled
- public embeddings return 768 dimensions
- public reranking returns ordered relevance results
- permanent assets: `web/default/public/images/logos/logo-atius.svg` and `logo-atius.png`
- channel `5` instance name is `ChatGPT - Codex`; its technical provider type remains `OpenAI - Codex`
