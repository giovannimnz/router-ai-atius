const DOCS_LOCALE_PREFIXES = [
  ['pt', 'pt'],
  ['zh', 'zh'],
  ['ja', 'ja'],
  ['en', 'en'],
] as const

export function getDocsLocale(language?: string): string {
  const normalized = (language || 'en').toLowerCase()
  const match = DOCS_LOCALE_PREFIXES.find(([prefix]) =>
    normalized.startsWith(prefix)
  )
  return match?.[1] || 'en'
}

export function getDocsBasePath(language?: string): string {
  return `/${getDocsLocale(language)}/docs`
}

export function isDocsSameOriginPath(href: string): boolean {
  return /^\/(en|pt|zh|ja)\/docs(?:\/|$)/.test(href)
}
