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
import {
  INTERFACE_LANGUAGE_OPTIONS,
  type InterfaceLanguageCode,
} from '@/i18n/languages'
import { resources } from '@/i18n/config'

const EXPECTED_CODES: InterfaceLanguageCode[] = [
  'en',
  'zh',
  'fr',
  'ja',
  'pt-BR',
  'ru',
  'vi',
]

const CONFIG_PATH = path.resolve(__dirname, '../config.ts')
const SWITCHER_PATH = path.resolve(__dirname, '../../components/language-switcher.tsx')

describe('i18n languages config wiring', () => {
  test('Test 5: INTERFACE_LANGUAGE_OPTIONS contains exactly the 7 expected codes (order-independent)', () => {
    const codes = INTERFACE_LANGUAGE_OPTIONS.map((opt) => opt.code).sort()
    const expected = [...EXPECTED_CODES].sort()
    expect(codes).toEqual(expected)
  })

  test('Test 6: language-switcher.tsx contains the pt-BR entry as a literal code', async () => {
    const src = await fs.readFile(SWITCHER_PATH, 'utf8')
    expect(src).toContain("'pt-BR'")
    expect(src).toContain("code: 'pt-BR'")
    // The switcher renders the Portuguese label
    expect(src).toContain('Português')
  })

  test('Test 7: config.ts wires pt-BR into resources and supportedLngs', async () => {
    const src = await fs.readFile(CONFIG_PATH, 'utf8')
    expect(src).toContain("'pt-BR': ptBR")
    expect(src).toMatch(/supportedLngs:\s*\[[^\]]*'pt-BR'/)

    // Runtime: resources['pt-BR'] is defined and shares the key set with en
    const ptBR = (resources as Record<string, Record<string, unknown>>)['pt-BR']
    const en = (resources as Record<string, Record<string, unknown>>).en
    expect(ptBR).toBeDefined()
    expect(en).toBeDefined()

    const enKeys = new Set(Object.keys(en))
    const ptKeys = new Set(Object.keys(ptBR))
    expect(ptKeys.size).toBe(enKeys.size)
    for (const k of enKeys) {
      expect(ptKeys.has(k), `resources.pt-BR missing top-level key: ${k}`).toBe(true)
    }
  })
})
