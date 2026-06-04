import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  poweredByHeader: false,
  // ATIUS CUSTOM: force trailing slashes so Apache's /en/ ProxyPass rule
  // doesn't get a 308 redirect to /en (which Apache then re-routes to
  // /en via the catch-all / ProxyPass and lands on the new-api SPA).
  trailingSlash: true,
  experimental: {
    serverActions: {
      allowedOrigins: [
        'localhost:3000',
        // ATIUS — docs site is served from router.atius.com.br via Apache
        'router.atius.com.br',
        'www.router.atius.com.br',
      ],
    },
  },
  async headers() {
    return [
      {
        // Apply charset to HTML pages
        source: '/:lang(en|zh|ja)/:path*',
        headers: [
          {
            key: 'Content-Type',
            value: 'text/html; charset=utf-8',
          },
        ],
      },
    ];
  },
  async rewrites() {
    return [
      {
        source: '/:lang/docs/:path*.mdx',
        destination: '/:lang/llms.mdx/:path*',
      },
    ];
  },
};

export default withMDX(config);
