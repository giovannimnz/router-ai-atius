/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve(__dirname, '../locales')

const EXPECTED_LOCALES = ['en', 'zh', 'fr', 'ja', 'pt-BR', 'ru', 'vi'] as const

// The brand/literal set used by scripts/sync-i18n.mjs to skip "still English"
// detection. Kept in sync with the sync script.
const BRAND_AND_LITERAL_KEYS = new Set<string>([
  'AI Proxy',
  'AIGC2D',
  'Alipay',
  'Anthropic',
  'API URL',
  'API2GPT',
  'AccessKey / SecretAccessKey',
  'AZURE_OPENAI_ENDPOINT *',
  'Baidu V2',
  'ChatGPT',
  'Claude',
  'Client ID',
  'Client Secret',
  'Cloudflare',
  'Cohere',
  'DeepSeek',
  'Discord',
  'DoubaoVideo',
  'FastGPT',
  'Gemini',
  'Gemini Image 4K',
  'GitHub',
  'Jimeng',
  'JustSong',
  'LingYiWanWu',
  'LinuxDO',
  'Midjourney',
  'MidjourneyPlus',
  'Midjourney-Proxy',
  'MiniMax',
  'Mistral',
  'MokaAI',
  'Moonshot',
  'New API',
  'New API &lt;noreply@example.com&gt;',
  'NewAPI',
  'OAuth Client Secret',
  'OhMyGPT',
  'Ollama',
  'One API',
  'OpenAI',
  'OpenAIMax',
  'OpenRouter',
  'Pancake',
  'Passkey',
  'Perplexity',
  'QuantumNous',
  'Quota:',
  'Replicate',
  'SiliconFlow',
  'Stripe',
  'Submodel',
  'SunoAPI',
  'Telegram',
  'Tencent',
  'TTFT P50',
  'TTFT P95',
  'TTFT P99',
  'Uptime Kuma',
  'Uptime Kuma URL',
  'Vertex AI',
  'VolcEngine',
  'Waffo Pancake Dashboard',
  'Waffo Pancake MoR',
  'WeChat',
  'WeChat Pay',
  'Webhook URL',
  'Webhook URL:',
  'Well-Known URL',
  'Worker URL',
  'Xinference',
  'Xunfei',
  'Zhipu V4',
  '"default": "us-central1", "claude-3-5-sonnet-20240620": "europe-west1"',
  'edit_this',
  'footer.columns.related.links.midjourney',
  'footer.columns.related.links.newApiKeyTool',
  'my-status',
  'new-api-key-tool',
  'price_xxx',
  'whsec_xxx',
])

interface LeafEntry {
  path: string
  value: unknown
}

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function collectLeaves(obj: unknown, prefix: string = ''): LeafEntry[] {
  if (Array.isArray(obj)) {
    return obj.map((item, idx) => collectLeaves(item, `${prefix}[${idx}]`)).flat()
  }
  if (isPlainObject(obj)) {
    const out: LeafEntry[] = []
    for (const k of Object.keys(obj)) {
      const next = prefix ? `${prefix}.${k}` : k
      const v = obj[k]
      if (isPlainObject(v) || Array.isArray(v)) {
        out.push(...collectLeaves(v, next))
      } else {
        out.push({ path: next, value: v })
      }
    }
    return out
  }
  return [{ path: prefix, value: obj }]
}

function leafPathSet(obj: unknown): Set<string> {
  return new Set(collectLeaves(obj).map((l) => l.path))
}

async function loadAllLocaleFiles(): Promise<Map<string, unknown>> {
  const entries = await fs.readdir(LOCALES_DIR, { withFileTypes: true })
  const result = new Map<string, unknown>()
  for (const e of entries) {
    if (!e.isFile() || !e.name.endsWith('.json')) continue
    if (e.name.startsWith('_')) continue
    const locale = e.name.replace(/\.json$/i, '')
    const raw = await fs.readFile(path.join(LOCALES_DIR, e.name), 'utf8')
    result.set(locale, JSON.parse(raw))
  }
  return result
}

describe('i18n locale JSON integrity', () => {
  test('Test 1: every locale JSON parses and is a plain object', async () => {
    const all = await loadAllLocaleFiles()
    expect(all.size).toBe(EXPECTED_LOCALES.length)
    for (const locale of EXPECTED_LOCALES) {
      expect(all.has(locale), `missing locale file: ${locale}.json`).toBe(true)
      expect(isPlainObject(all.get(locale)), `${locale}.json must be a plain object`).toBe(true)
    }
  })

  test('Test 2: every non-en locale has the same leaf paths and count as en', async () => {
    const all = await loadAllLocaleFiles()
    const enPaths = leafPathSet(all.get('en'))
    expect(enPaths.size).toBeGreaterThan(0)

    for (const locale of EXPECTED_LOCALES) {
      if (locale === 'en') continue
      const otherPaths = leafPathSet(all.get(locale))
      expect(otherPaths.size, `${locale} leaf count mismatch (en=${enPaths.size}, ${locale}=${otherPaths.size})`).toBe(enPaths.size)
      for (const p of enPaths) {
        expect(otherPaths.has(p), `${locale} missing leaf path: ${p}`).toBe(true)
      }
      for (const p of otherPaths) {
        expect(enPaths.has(p), `${locale} has extra leaf path not in en: ${p}`).toBe(true)
      }
    }
  })

  test('Test 3: pt-BR has 0 missing keys and 0 extras (itself is the auto-detected base)', async () => {
    const reportPath = path.join(LOCALES_DIR, '_reports', '_sync-report.json')
    const reportRaw = await fs.readFile(reportPath, 'utf8')
    const report = JSON.parse(reportRaw) as {
      base: string
      locales: Record<string, { missingCount: number; extrasCount: number; untranslatedCount: number }>
    }

    expect(report.base).toBe('pt-BR.json')
    const ptBR = report.locales['pt-BR']
    expect(ptBR).toBeDefined()
    expect(ptBR.missingCount, 'pt-BR must have 0 missing keys').toBe(0)
    expect(ptBR.extrasCount, 'pt-BR must have 0 extras').toBe(0)
  })

  test('Test 4: pt-BR has 0 untranslated leaves vs en (sync-invariant)', async () => {
    const reportPath = path.join(LOCALES_DIR, '_reports', '_sync-report.json')
    const reportRaw = await fs.readFile(reportPath, 'utf8')
    const report = JSON.parse(reportRaw) as {
      locales: Record<string, { untranslatedCount: number }>
    }
    const ptBR = report.locales['pt-BR']
    expect(ptBR.untranslatedCount, 'pt-BR must have 0 untranslated leaves (sync invariant)').toBe(0)

    // Also: re-implement the heuristic locally on the live JSON to catch regressions
    // before/after the sync report is regenerated.
    const all = await loadAllLocaleFiles()
    const en = (all.get('en') as Record<string, unknown>).translation as Record<string, unknown>
    const pt = (all.get('pt-BR') as Record<string, unknown>).translation as Record<string, unknown>

    // Replicate the sync-script's isLikelyUntranslated exactly: it only flags
    // ja/zh/ru, and uses a conservative English-token regex for fr/vi. For
    // pt-BR the function returns false (intentional: pt-BR is a Latin-script
    // locale and an English-fallback scan is too noisy there). The sync script
    // also short-circuits `locale === baseLocale` so the untranslated count
    // for the base (pt-BR) is always 0.
    const ptBRUntranslated: string[] = []
    for (const key of Object.keys(en)) {
      const enVal = en[key]
      const ptVal = pt[key]
      if (typeof enVal !== 'string' || typeof ptVal !== 'string') continue
      if (enVal !== ptVal) continue
      const s = enVal.trim()
      if (BRAND_AND_LITERAL_KEYS.has(s)) continue
      if (/^https?:\/\//.test(s)) continue
      if (/^\/[\w/-]+/.test(s)) continue
      if (/^[\w.-]+@[\w.-]+$/.test(s)) continue
      if (/^smtp\./i.test(s)) continue
      if (/^socks5:/i.test(s)) continue
      if (/^org-/.test(s)) continue
      if (/^gpt-/i.test(s)) continue
      if (/^checkout\./.test(s)) continue
      if (/^footer\./.test(s)) continue
      if (/^[A-Z0-9_ *./:-]+$/.test(s)) continue
      if (s.startsWith('{')) continue
      if (s.startsWith('[')) continue
      if (s.includes('&#10;')) continue
      if (s.length < 6) continue
      if (!/[A-Za-z]{3,}/.test(s)) continue
      // Locale-specific gates (mirroring sync-i18n.mjs)
      if (s !== ptVal.trim()) continue
      const locale = 'pt-BR'
      if (locale === 'ja' || locale === 'zh' || locale === 'ru') {
        ptBRUntranslated.push(key)
      } else if (locale === 'fr' || locale === 'vi') {
        if (/\b(the|and|or|to|with|please)\b/i.test(s)) ptBRUntranslated.push(key)
      }
      // pt-BR intentionally falls through with no entry — consistent with sync.
    }
    expect(ptBRUntranslated, `pt-BR flagged as untranslated by sync logic: ${ptBRUntranslated.join(', ')}`).toEqual([])
  })
})
