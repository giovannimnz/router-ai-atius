export const getDocsLocale = (language = 'en') => {
  const normalized = String(language || 'en').toLowerCase();
  if (normalized.startsWith('pt')) return 'pt';
  if (normalized.startsWith('zh')) return 'zh';
  if (normalized.startsWith('ja')) return 'ja';
  return 'en';
};

export const getDocsBasePath = (language = 'en') =>
  `/${getDocsLocale(language)}/docs`;
