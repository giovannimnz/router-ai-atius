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

import { ATIUS_LOCAL_ICON_KEY } from '@/components/atius-logo-key'

import { getLobeIcon } from './lobe-icon'

describe('getLobeIcon Atius branding', () => {
  test('resolves the compact model descriptor to the canonical logo', () => {
    const icon = getLobeIcon(
      `${ATIUS_LOCAL_ICON_KEY}.Avatar.type={'platform'}`,
      20
    )
    const markup = renderToStaticMarkup(icon)

    assert.match(markup, /class="text-background /)
    assert.match(markup, /fill="currentColor"/)
    assert.doesNotMatch(markup, />A<\/div>/)
  })
})
