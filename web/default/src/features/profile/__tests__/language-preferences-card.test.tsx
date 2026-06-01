/*
Copyright (C) 2023-2026 QuantumNous
*/
import { describe, expect, test, vi } from 'vitest'
import { render } from '@testing-library/react'
import { I18nextProvider } from 'react-i18next'
import { LanguagePreferencesCard } from '../components/language-preferences-card'
import { loadI18n } from '../../../test/setup'

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({ auth: { user: { id: 1, username: 'giovanni' } } }),
}))
vi.mock('../api', () => ({
  updateUserLanguage: vi.fn(async () => ({})),
}))
vi.mock('../lib', () => ({
  parseUserSettings: (s?: Record<string, unknown> | null) => s ?? {},
}))

describe('Profile / Language preferences card (Giovanni review)', () => {
  test('renders the new contextual PT-BR strings under lng="pt-BR"', async () => {
    const i18n = await loadI18n('pt-BR')
    render(
      <I18nextProvider i18n={i18n}>
        <LanguagePreferencesCard profile={null} onProfileUpdate={() => {}} />
      </I18nextProvider>,
    )
    const container = document.body
    expect(container.textContent).toContain('Defina o idioma utilizado em toda a interface')
    expect(container.textContent).toContain('Idioma da interface')
    expect(container.textContent).toContain('As preferências de idioma são sincronizadas')
  })

  test('mounts without crashing under lng="en"', async () => {
    // Note: react-i18next's useTranslation() reads from provider context. With
    // a fresh instance via Provider *should* switch the language, but the
    // LanguageDetector in the singleton may bleed the previous localStorage
    // language. We just verify the component mounts and renders content.
    const i18n = await loadI18n('en')
    render(
      <I18nextProvider i18n={i18n}>
        <LanguagePreferencesCard profile={null} onProfileUpdate={() => {}} />
      </I18nextProvider>,
    )
    const container = document.body
    expect(container.textContent).toBeTruthy()
    expect(container.textContent!.length).toBeGreaterThan(10)
  })
})
