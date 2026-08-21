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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { CHANNEL_TYPES } from '../constants'
import {
  CHANNEL_TYPE_ADVANCED_CUSTOM,
  CHANNEL_TYPE_ATIUS_LOCAL_EMBEDDINGS,
  createAtiusLocalEmbeddingsConfig,
  isAdvancedCustomChannelType,
  stringifyAdvancedCustomConfig,
  validateAdvancedCustomConfig,
} from './advanced-custom'

describe('Atius Local channel preset', () => {
  test('keeps the production Codex channel name canonical', () => {
    assert.equal(CHANNEL_TYPES[57], 'ChatGPT - Codex')
  })

  test('contains the validated embeddings and reranker routes', () => {
    const config = createAtiusLocalEmbeddingsConfig()

    assert.equal(validateAdvancedCustomConfig(config), null)
    assert.deepEqual(config.advanced_routes, [
      {
        incoming_path: '/v1/embeddings',
        upstream_path: 'http://10.21.1.21:31115/v1/embeddings',
        converter: 'none',
        auth: { type: 'none' },
      },
      {
        incoming_path: '/v1/rerank',
        upstream_path: 'http://10.21.1.21:31216/rerank',
        converter: 'jina_rerank_to_tei_native',
        auth: { type: 'none' },
      },
    ])
    assert.doesNotThrow(() => JSON.parse(stringifyAdvancedCustomConfig(config)))
  })

  test('shares Advanced Custom behavior without changing the generic type', () => {
    assert.equal(
      isAdvancedCustomChannelType(CHANNEL_TYPE_ADVANCED_CUSTOM),
      true
    )
    assert.equal(
      isAdvancedCustomChannelType(CHANNEL_TYPE_ATIUS_LOCAL_EMBEDDINGS),
      true
    )
    assert.equal(isAdvancedCustomChannelType(1), false)
  })
})
