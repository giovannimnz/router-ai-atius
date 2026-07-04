\set ON_ERROR_STOP 1

\if :{?codex_channel_key_json}
\else
\set codex_channel_key_json __SET_FROM_SECURE_SOURCE__
\endif

-- Phase 24 catalog transform for candidate DBRouterAiAtius.
-- This file is meant for the candidate DB only, never for blind replay over live newapi.
-- Secure use example:
--   sudo -u postgres psql -h 127.0.0.1 -p 8745 -d DBRouterAiAtius \
--     -v codex_channel_key_json="$(cat /secure/path/codex-channel-key.json)" \
--     -f scripts/phase24-catalog-transform.sql

BEGIN;

DO $guard$
BEGIN
  IF :'codex_channel_key_json' = '__SET_FROM_SECURE_SOURCE__' THEN
    RAISE EXCEPTION 'codex_channel_key_json must be passed securely with -v codex_channel_key_json=...';
  END IF;
END
$guard$;

-- Remove forbidden aliases and disabled Codex embeddings rows before rebuild.
DELETE FROM public.abilities
WHERE (channel_id = 5 AND (model LIKE 'gpt-5._-1m' OR model LIKE 'text-embedding-3-%'))
   OR model LIKE 'gpt-5._-1m'
   OR model LIKE 'text-embedding-3-%';

DELETE FROM public.models
WHERE model_name LIKE 'gpt-5._-1m'
   OR model_name LIKE 'text-embedding-3-%';

-- Channel 1: consolidated MiniMax restored but disabled in the final state.
UPDATE public.channels
SET type = 35,
    status = 2,
    name = 'MiniMax',
    test_model = '',
    base_url = 'https://api.minimax.io',
    models = 'MiniMax-M3,MiniMax-M2.7-highspeed,MiniMax-M2.7',
    model_mapping = '',
    remark = '2026-07-04: Phase 24 restored consolidated MiniMax catalog on candidate DB; final state remains disabled.'
WHERE id = 1;

-- Channel 2: consolidated DeepSeek stays active.
UPDATE public.channels
SET type = 43,
    status = 1,
    name = 'DeepSeek',
    base_url = 'https://api.deepseek.com',
    models = 'deepseek-v4-pro,deepseek-v4-flash',
    model_mapping = '',
    remark = '2026-07-04: Phase 24 reconciled consolidated DeepSeek catalog on candidate DB.'
WHERE id = 2;

-- Channel 5: OpenAI - Codex restored without long-context alias rows or Codex embedding rows.
INSERT INTO public.channels (
  id, type, key, open_ai_organization, test_model, status, name, weight, created_time,
  test_time, response_time, base_url, other, balance, balance_updated_time, models, "group",
  used_quota, model_mapping, status_code_mapping, priority, auto_ban, other_info, tag, setting,
  param_override, header_override, remark, channel_info, settings
)
VALUES (
  5, 57, :'codex_channel_key_json', NULL, 'gpt-5.5', 1, 'OpenAI - Codex', 0, EXTRACT(EPOCH FROM NOW())::bigint,
  0, 0, '', NULL, NULL, NULL, 'gpt-5.5,gpt-5.4,gpt-5.4-mini,gpt-5.3-codex-spark', 'default',
  0, '', '', 0, 1, NULL, NULL, NULL, NULL, NULL,
  '2026-07-04: Phase 24 restored OpenAI - Codex on candidate DB using secure credential injection; -1m aliases intentionally excluded.',
  NULL, NULL
)
ON CONFLICT (id) DO UPDATE
SET type = EXCLUDED.type,
    key = EXCLUDED.key,
    test_model = EXCLUDED.test_model,
    status = EXCLUDED.status,
    name = EXCLUDED.name,
    models = EXCLUDED.models,
    model_mapping = EXCLUDED.model_mapping,
    remark = EXCLUDED.remark;

-- Channel 9: preserve Go-governed TEI embeddings path.
UPDATE public.channels
SET type = 1,
    status = 1,
    name = 'Local TEI - GTE Embeddings',
    test_model = 'embedding-gte-v1',
    models = 'embedding-gte-v1',
    model_mapping = '{}',
    remark = 'Local TEI embeddings channel for embedding-gte-v1; preserved by Phase 24 candidate catalog restore.'
WHERE id = 9;

INSERT INTO public.models (
  id, model_name, description, icon, tags, vendor_id, endpoints, status,
  sync_official, created_time, updated_time, deleted_at, name_rule
)
VALUES
  (1, 'MiniMax-M2.7', 'MiniMax M2.7', '', 'OpenAI,Anthropic', 0, '["anthropic","openai"]', 0, 1, 1777505227, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (2, 'MiniMax-M2.7-highspeed', 'MiniMax M2.7 Highspeed', '', 'OpenAI,Anthropic', 0, '{"openai":{"path":"/v1/chat/completions","method":"POST"},"anthropic":{"path":"/v1/messages","method":"POST"}}', 0, 1, 1777505227, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (5, 'deepseek-v4-flash', 'DeepSeek V4 Flash', '@DeepSeek', 'OpenAI,Anthropic', 0, '{"openai":{"path":"/v1/chat/completions","method":"POST"},"anthropic":{"path":"/v1/messages","method":"POST"}}', 1, 1, 1777505227, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (6, 'deepseek-v4-pro', 'DeepSeek V4 Pro', '', 'OpenAI,Anthropic', 0, '{"openai":{"path":"/v1/chat/completions","method":"POST"},"anthropic":{"path":"/v1/messages","method":"POST"}}', 1, 1, 1777505227, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (13, 'MiniMax-M3', 'MiniMax-M3', '', 'OpenAI,Anthropic', 0, '["openai","anthropic"]', 0, 1, 1780273324, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (14, 'gpt-5.5', 'OpenAI Codex GPT-5.5', '', 'Codex,OpenAI', 0, '["openai"]', 1, 0, 1781388789, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (15, 'gpt-5.4', 'OpenAI Codex GPT-5.4', '', 'Codex,OpenAI,Long Context', 0, '["openai"]', 1, 0, 1781388789, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (16, 'gpt-5.4-mini', 'OpenAI Codex GPT-5.4 Mini', '', 'Codex,OpenAI', 0, '["openai"]', 1, 0, 1781388789, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (17, 'gpt-5.3-codex-spark', 'OpenAI Codex Spark', '', 'Codex,OpenAI', 0, '["openai"]', 1, 0, 1781388789, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0),
  (21, 'embedding-gte-v1', 'Local TEI GTE embeddings (governed)', '', 'Embeddings,Local TEI,Governor', 0, '["embeddings"]', 1, 0, 1782513928, EXTRACT(EPOCH FROM NOW())::bigint, NULL, 0)
ON CONFLICT (id) DO UPDATE
SET model_name = EXCLUDED.model_name,
    description = EXCLUDED.description,
    icon = EXCLUDED.icon,
    tags = EXCLUDED.tags,
    vendor_id = EXCLUDED.vendor_id,
    endpoints = EXCLUDED.endpoints,
    status = EXCLUDED.status,
    sync_official = EXCLUDED.sync_official,
    updated_time = EXCLUDED.updated_time,
    deleted_at = EXCLUDED.deleted_at,
    name_rule = EXCLUDED.name_rule;

-- Rebuild only the intended routing abilities.
DELETE FROM public.abilities
WHERE (channel_id = 1 AND model IN ('MiniMax-M3', 'MiniMax-M2.7-highspeed', 'MiniMax-M2.7'))
   OR (channel_id = 2 AND model IN ('deepseek-v4-pro', 'deepseek-v4-flash'))
   OR (channel_id = 5 AND model IN ('gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.3-codex-spark'))
   OR (channel_id = 9 AND model = 'embedding-gte-v1');

INSERT INTO public.abilities ("group", model, channel_id, enabled, priority, weight, tag)
VALUES
  ('default', 'MiniMax-M3', 1, false, 0, 0, ''),
  ('default', 'MiniMax-M2.7-highspeed', 1, false, 0, 0, ''),
  ('default', 'MiniMax-M2.7', 1, false, 0, 0, ''),
  ('default', 'deepseek-v4-pro', 2, true, 0, 0, ''),
  ('default', 'deepseek-v4-flash', 2, true, 0, 0, ''),
  ('default', 'gpt-5.5', 5, true, 0, 0, NULL),
  ('default', 'gpt-5.4', 5, true, 0, 0, NULL),
  ('default', 'gpt-5.4-mini', 5, true, 0, 0, NULL),
  ('default', 'gpt-5.3-codex-spark', 5, true, 0, 0, NULL),
  ('default', 'embedding-gte-v1', 9, true, 0, 0, 'local-tei');

SELECT pg_catalog.setval('public.channels_id_seq', GREATEST((SELECT COALESCE(MAX(id), 1) FROM public.channels), 9), true);
SELECT pg_catalog.setval('public.models_id_seq', GREATEST((SELECT COALESCE(MAX(id), 1) FROM public.models), 21), true);

COMMIT;
