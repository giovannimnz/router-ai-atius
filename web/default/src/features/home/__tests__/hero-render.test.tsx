/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { describe, expect, test, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { I18nextProvider } from 'react-i18next'
import { Hero } from '../components/sections/hero'
import { loadI18n } from '../../test/setup'

// Mock the useStatus hook so Hero doesn't make network calls.
vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: undefined, isLoading: false }),
}))

describe('Hero renders translated text under pt-BR (Brief Test 8)', () => {
  test('every t() call in hero.tsx renders the Portuguese value when lng="pt-BR"', async () => {
    const i18n = await loadI18n('pt-BR')

    render(
      <I18nextProvider i18n={i18n}>
        <Hero isAuthenticated={false} />
      </I18nextProvider>,
    )

    // h1 — split across two lines in the source
    expect(
      screen.getByText(/API Gateway unificado para/),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Ampla Variedade de Modelos de IA/),
    ).toBeInTheDocument()

    // Buttons
    expect(screen.getByText('Começar')).toBeInTheDocument()
    expect(screen.getByText(/Visualizar Pricing/)).toBeInTheDocument()
    expect(screen.getByText('Documentação')).toBeInTheDocument()

    // Section headings
    expect(
      screen.getByText(/Construído para desenvolvedores, projetado para escala/),
    ).toBeInTheDocument()
    expect(screen.getByText('Velocidade Relâmpago')).toBeInTheDocument()
    expect(screen.getByText('Seguro e confiável')).toBeInTheDocument()
    expect(screen.getByText('Cobertura Global')).toBeInTheDocument()
    expect(screen.getByText('Amigável para Desenvolvedores')).toBeInTheDocument()

    // Final CTA
    expect(
      screen.getByText(/Pronto para simplificar sua integração de IA/),
    ).toBeInTheDocument()
  })

  test('English fallback works when lng="en"', async () => {
    const i18n = await loadI18n('en')

    render(
      <I18nextProvider i18n={i18n}>
        <Hero isAuthenticated={false} />
      </I18nextProvider>,
    )

    expect(screen.getByText(/Unified API Gateway for/)).toBeInTheDocument()
    expect(screen.getByText(/Vast Range of AI Models/)).toBeInTheDocument()
    expect(screen.getByText('Get Started')).toBeInTheDocument()
  })
})
