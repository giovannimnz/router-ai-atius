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
import { renderToStaticMarkup } from 'react-dom/server'

import { CHANNEL_TYPE_ATIUS_LOCAL_EMBEDDINGS } from '../lib'
import { ChannelTypeLogo } from './channel-type-logo'

describe('ChannelTypeLogo Atius branding', () => {
  test('keeps the background theme-aware and preserves brand colors', () => {
    const markup = renderToStaticMarkup(
      <ChannelTypeLogo type={CHANNEL_TYPE_ATIUS_LOCAL_EMBEDDINGS} size={18} />
    )

    assert.match(markup, /class="text-background /)
    assert.match(markup, /fill="currentColor"/)
    assert.doesNotMatch(markup, /fill="#fff"/)
    assert.match(markup, /fill="#0f3b25"/)
    assert.match(markup, /fill="#d2aa2a"/)
  })
})
