import { describe, expect, it } from 'bun:test'

import {
  CHANNEL_FORM_DEFAULT_VALUES,
  transformFormDataToUpdatePayload,
} from './channel-form'

describe('channel setting preservation', () => {
  it('keeps credential ownership and health metadata when editing known settings', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        proxy: 'http://proxy.example',
        setting: JSON.stringify({
          codex_credential_source: 'external_file',
          codex_credential_health: {
            last_probe_status: 'ok',
          },
        }),
      },
      5
    )

    const setting = JSON.parse(String(payload.setting))
    expect(setting).toMatchObject({
      proxy: 'http://proxy.example',
      codex_credential_source: 'external_file',
      codex_credential_health: {
        last_probe_status: 'ok',
      },
    })
  })
})
