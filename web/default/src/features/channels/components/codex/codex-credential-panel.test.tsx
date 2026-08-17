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

import ptTranslations from '@/i18n/locales/pt.json'

import type { CodexCredentialMetadata } from '../../types'
import {
  CodexCredentialPanel,
  isCodexChannelType,
  shouldWarnAboutBaseUrl,
} from './codex-credential-panel'

const translations = ptTranslations.translation as Record<string, string>

const t = (key: string) => translations[key] ?? key

const baseMetadata: CodexCredentialMetadata = {
  channel_id: 5,
  channel_type: 57,
  channel_name: 'OpenAI - Codex',
  authenticated: true,
  has_refresh_token: true,
  expires_at: '2026-07-17T11:04:04Z',
  last_refresh: '2026-07-10T11:04:04Z',
  account_id: 'account-public-id',
  email: 'operator@example.com',
  last_probe_at: '2026-07-12T10:00:00Z',
  last_probe_status: 'ok',
  last_upstream_status: 200,
  last_upstream_auth_error: '',
  requires_regeneration: false,
  regeneration_reason: '',
}

function renderPanel(metadata: CodexCredentialMetadata) {
  return renderToStaticMarkup(
    <CodexCredentialPanel
      metadata={metadata}
      canManage
      hasSavedChannel
      isLoading={false}
      isRefreshing={false}
      isProbing={false}
      onRefresh={() => undefined}
      onProbe={() => undefined}
      onRegenerate={() => undefined}
      t={t}
    />
  )
}

function assertGenericCredentialControlsAbsent(markup: string) {
  for (const forbidden of [
    'Base URL',
    'API Key *',
    'Current key',
    'Reveal key',
    '>Copy<',
  ]) {
    assert.equal(markup.includes(forbidden), false, forbidden)
  }
}

describe('CodexCredentialPanel', () => {
  test('uses the type 57 boundary that gates generic credential fields', () => {
    assert.equal(isCodexChannelType(57), true)
    assert.equal(isCodexChannelType(1), false)
    assert.equal(shouldWarnAboutBaseUrl(57, 'https://example.com/v1'), false)
    assert.equal(shouldWarnAboutBaseUrl(1, 'https://example.com/v1'), true)
    assert.equal(shouldWarnAboutBaseUrl(1, 'https://example.com'), false)
  })

  test('renders healthy sanitized metadata and distinct lifecycle actions', () => {
    const markup = renderPanel(baseMetadata)

    assert.match(markup, /Credencial OAuth Codex/)
    assert.match(markup, /Autenticado/)
    assert.match(markup, /Atualizar credencial/)
    assert.match(markup, /Regenerar credencial/)
    assert.match(markup, /operator@example\.com/)
    assert.match(markup, /Probe OK/)
    assertGenericCredentialControlsAbsent(markup)
  })

  test('represents a missing refresh token as regeneration-required', () => {
    const markup = renderPanel({
      ...baseMetadata,
      has_refresh_token: false,
      requires_regeneration: true,
      regeneration_reason: 'refresh_token_missing',
    })

    assert.match(markup, /Sem refresh_token/)
    assert.match(markup, /Precisa regenerar/)
    assert.match(markup, /correção definitiva/)
    assertGenericCredentialControlsAbsent(markup)
  })

  test('warns when upstream auth fails despite future local expiration', () => {
    const markup = renderPanel({
      ...baseMetadata,
      last_probe_status: 'auth_failed',
      last_upstream_status: 401,
      last_upstream_auth_error: 'token_invalidated',
      requires_regeneration: true,
      regeneration_reason: 'token_invalidated',
    })

    assert.match(markup, /Erro upstream/)
    assert.match(markup, /token_invalidated/)
    assert.match(markup, /expiração local ainda está no futuro/)
    assertGenericCredentialControlsAbsent(markup)
  })
})
