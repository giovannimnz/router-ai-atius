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
import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// Mock the auth store and api BEFORE importing the component under test.
const apiPutMock = vi.fn()
const useAuthStoreMock = vi.fn()

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: (selector: (s: { auth: { user: unknown } }) => unknown) =>
    useAuthStoreMock(selector),
}))
vi.mock('@/lib/api', () => ({
  api: {
    put: (...args: unknown[]) => apiPutMock(...args),
  },
}))

import { LanguageSwitcher } from '@/components/language-switcher'

describe('LanguageSwitcher', () => {
  beforeEach(() => {
    // Default: logged out
    useAuthStoreMock.mockImplementation((selector) =>
      selector({ auth: { user: null } })
    )
    apiPutMock.mockReset()
  })

  afterEach(() => {
    cleanup()
  })

  test('renders the trigger and includes pt-BR (Português) in the menu', async () => {
    const user = userEvent.setup()
    render(<LanguageSwitcher />)

    // The trigger button is the outer dropdown trigger (it carries the
    // accessible "Change language" name via the sr-only span). Pick it by
    // the exact text inside the sr-only span to avoid the nested icon button.
    const trigger = screen.getByText('Change language').closest('button')
    expect(trigger).toBeInTheDocument()

    await user.click(trigger as HTMLElement)

    // The menu should now include the Portuguese label
    await waitFor(() => {
      expect(screen.getByText('Português')).toBeInTheDocument()
    })
  })

  test('clicking pt-BR calls i18n.changeLanguage("pt-BR") and skips api.put when logged out', async () => {
    const user = userEvent.setup()
    // Use the real i18n singleton already initialised by the project.
    // Ensure we start from "en" so we can observe the change.
    const { default: i18n } = await import('@/i18n/config')
    await i18n.changeLanguage('en')

    const changeSpy = vi.spyOn(i18n, 'changeLanguage')

    render(<LanguageSwitcher />)

    const trigger = screen.getByText('Change language').closest('button')
    await user.click(trigger as HTMLElement)

    const ptItem = await screen.findByText('Português')
    fireEvent.click(ptItem)

    await waitFor(() => {
      expect(changeSpy).toHaveBeenCalledWith('pt-BR')
    })
    expect(i18n.language).toBe('pt-BR')
    expect(apiPutMock).not.toHaveBeenCalled()
  })
})
