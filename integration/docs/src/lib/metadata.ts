import type { Metadata } from 'next';

export function createMetadata(override: Metadata): Metadata {
  return {
    ...override,
    icons: {
      icon: '/favicon.ico',
      shortcut: '/favicon.ico',
      apple: '/assets/logo.png',
    },
    openGraph: {
      title: override.title ?? undefined,
      description: override.description ?? undefined,
      url: 'https://router.atius.com.br',
      images: '/assets/logo.png',
      siteName: 'New API',
      type: 'website',
      ...override.openGraph,
    },
    twitter: {
      card: 'summary_large_image',
      title: override.title ?? undefined,
      description: override.description ?? undefined,
      images: '/assets/logo.png',
      ...override.twitter,
    },
  };
}

export const baseUrl =
  process.env.NODE_ENV === 'development' ||
  !process.env.VERCEL_PROJECT_PRODUCTION_URL
    // ATIUS CUSTOM: hardcoded production URL (was localhost:3000 upstream).
    // The docs site runs on https://router.atius.com.br/docs via Apache,
    // so all metadata (sitemap, llms.txt, OG) must use that host.
    ? new URL('https://router.atius.com.br')
    : new URL(`https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`);
