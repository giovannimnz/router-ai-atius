/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { describe, expect, test } from 'vitest'

import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from '@/i18n/locales/en.json'
import ptBR from '@/i18n/locales/pt-BR.json'
describe('inline', () => {
  test('inline works?', async () => {
    const a = i18next.createInstance()
    await new Promise<void>((r) => a.use(initReactI18next).init({
      resources: { en, 'pt-BR': ptBR },
      lng: 'pt-BR',
      fallbackLng: 'en',
      supportedLngs: ['en', 'pt-BR'],
      nsSeparator: false,
      interpolation: { escapeValue: false },
    }, () => r()))
    console.log('INLINE-FULL t Get Started:', a.t('Get Started'))
  })
})

import { loadI18n } from '../../test/setup'

describe('i18n runtime behavior (Brief Tests 8-9)', () => {
  test('init with lng="pt-BR" resolves the translation', async () => {
    const i18n = await loadI18n('pt-BR')
    expect(i18n.language).toBe('pt-BR')
    const pt = i18n.t('Get Started')
    expect(pt).toBe('Começar')
  })

  test('init with lng="en" and switch to pt-BR via changeLanguage', async () => {
    const i18n = await loadI18n('en')
    expect(i18n.t('Get Started')).toBe('Get Started')
    await new Promise<void>((r) => i18n.changeLanguage('pt-BR', () => r()))
    expect(i18n.language).toBe('pt-BR')
    const pt = i18n.t('Get Started')
    expect(pt).toBe('Começar')
  })

  test('init with lng="pt-BR" and switch to en via changeLanguage', async () => {
    const i18n = await loadI18n('pt-BR')
    expect(i18n.t('Get Started')).toBe('Começar')
    await new Promise<void>((r) => i18n.changeLanguage('en', () => r()))
    expect(i18n.language).toBe('en')
    expect(i18n.t('Get Started')).toBe('Get Started')
  })

  test('all 7 locales can be selected and resolve a known key', async () => {
    for (const lng of ['en', 'zh', 'fr', 'ja', 'pt-BR', 'ru', 'vi']) {
      const i18n = await loadI18n(lng)
      expect(i18n.language, `lang not set for ${lng}`).toBe(lng)
      const v = i18n.t('Get Started')
      expect(v, `no translation for ${lng}`).toBeTypeOf('string')
      expect(v.length, `empty translation for ${lng}`).toBeGreaterThan(0)
    }
  })

  test('i18n changeLanguage does not throw for any of the 7 locales', async () => {
    const i18n = await loadI18n('en')
    for (const lng of ['en', 'zh', 'fr', 'ja', 'pt-BR', 'ru', 'vi']) {
      await expect(
        new Promise<void>((r, j) => i18n.changeLanguage(lng, (err) => (err ? j(err) : r()))),
      ).resolves.toBeUndefined()
    }
  })

  test('config.ts resources key matches supportedLngs — pt-BR resolves', async () => {
    const i18n = await loadI18n('pt-BR')
    // Strong signals: strings that are intentionally Portuguese (not brand).
    expect(i18n.t('Unified API Gateway for')).toBe('API Gateway unificado para')
    expect(i18n.t('Lightning Fast')).toBe('Velocidade Relâmpago')
    expect(i18n.t('Built for developers, designed for scale')).toBe(
      'Construído para desenvolvedores, projetado para escala',
    )
    expect(i18n.t('Get Started')).toBe('Começar')
    expect(i18n.t('Docs')).toBe('Documentação')
  })

  test('persistence — changeLanguage caches in localStorage', async () => {
    const i18n = await loadI18n('en')
    await new Promise<void>((r) => i18n.changeLanguage('pt-BR', () => r()))
    // LanguageDetector default lookup key is 'i18nextLng'
    const stored = localStorage.getItem('i18nextLng')
    expect(stored).toBe('pt-BR')
  })
})
