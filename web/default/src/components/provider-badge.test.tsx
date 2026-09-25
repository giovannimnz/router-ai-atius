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

import { ATIUS_LOCAL_ICON_KEY } from './atius-logo-key'
import { ProviderBadge } from './provider-badge'

describe('ProviderBadge Atius branding', () => {
  test('renders the canonical monochromatic logo for Atius Local', () => {
    const markup = renderToStaticMarkup(
      <ProviderBadge iconKey={ATIUS_LOCAL_ICON_KEY} label='Atius Local' />
    )

    assert.match(markup, /fill="#000000"/)
    assert.match(markup, /fill="#FFFFFF"/)
    assert.match(markup, /fill="#D1D5DB"/)
    assert.doesNotMatch(markup, /fill="#0f3b25"/)
    assert.doesNotMatch(markup, /fill="#d2aa2a"/)
    assert.match(markup, />Atius Local</)
  })
})
