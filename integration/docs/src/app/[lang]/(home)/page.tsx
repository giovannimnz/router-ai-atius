import Link from 'next/link';
import { Github, BookOpen } from 'lucide-react';
import { Hero } from './page.client';
import { getLocalePath, i18n } from '@/lib/i18n';
import Image from 'next/image';
import { AntifraudDialog } from '@/components/antifraud-dialog';

const contentMap: Record<
  string,
  {
    badge: string;
    title: string;
    subtitle: string;
    highlight: string;
    getStarted: string;
    github: string;
    partnersTitle: string;
    partnersSubtitle: string;
    sponsorPartnersTitle: string;
    sponsorPartnersSubtitle: string;
    devContributorsTitle: string;
    docsContributorsTitle: string;
  }
> = {
  en: {
    // ATIUS: title tuned for the Atius AI Router (was "Connect all AI providers...")
    badge: 'Atius AI Router — Production Ready',
    title: 'Aggregate 40+ AI providers behind a single OpenAI/Anthropic-compatible API.',
    subtitle: 'Built on QuantumNous/new-api, hardened for',
    highlight: 'production',
    getStarted: 'Read the docs',
    github: 'GitHub',
    partnersTitle: 'Compatible AI Providers',
    partnersSubtitle: 'OpenAI, Anthropic, Gemini, DeepSeek, Mistral, and 35+ more',
    sponsorPartnersTitle: 'Atius-Sponsored Partners',
    sponsorPartnersSubtitle: 'Trusted integrations with the Atius AI Router',
    devContributorsTitle: 'Development Contributors',
    docsContributorsTitle: 'Documentation Contributors',
  },
  zh: {
    badge: 'Atius AI Router — 生产就绪',
    title: '将 40+ AI 提供商聚合到统一的 OpenAI / Anthropic 兼容 API 之后。',
    subtitle: '基于 QuantumNous/new-api，专为',
    highlight: '生产环境',
    getStarted: '阅读文档',
    github: 'GitHub',
    partnersTitle: '兼容的 AI 提供商',
    partnersSubtitle: 'OpenAI、Anthropic、Gemini、DeepSeek、Mistral 以及 35+ 其他',
    sponsorPartnersTitle: 'Atius 赞助合作伙伴',
    sponsorPartnersSubtitle: '与 Atius AI Router 值得信赖的集成',
    devContributorsTitle: '开发贡献者',
    docsContributorsTitle: '文档贡献者',
  },
  ja: {
    badge: 'Atius AI Router — 本番運用対応',
    title: '40 以上の AI プロバイダーを単一の OpenAI / Anthropic 互換 API に集約。',
    subtitle: 'QuantumNous/new-api をベースに、',
    highlight: '本番環境',
    getStarted: 'ドキュメントを読む',
    github: 'GitHub',
    partnersTitle: '対応 AI プロバイダー',
    partnersSubtitle: 'OpenAI、Anthropic、Gemini、DeepSeek、Mistral ほか 35 以上',
    sponsorPartnersTitle: 'Atius スポンサーパートナー',
    sponsorPartnersSubtitle: 'Atius AI Router との信頼性の高い統合',
    devContributorsTitle: '開発貢献者',
    docsContributorsTitle: 'ドキュメント貢献者',
  },
} as const;

export default async function Page({
  params,
}: {
  params: Promise<{ lang: string }>;
}) {
  const { lang } = await params;
  const content = contentMap[lang] || contentMap.en;

  // ATIUS: removed partner logos (Cherry Studio, AionUi, PKU, UCloud, Alibaba, IO.NET, RixAPI)
  // because they were QuantumNous' commercial partners. Atius is a
  // self-hosted fork; we don't have those relationships. Keeping the
  // section for layout but emptying the data.
  const partners: { name: string; url: string; logo: string }[] = [];

  const sponsorPartners: {
    name: string;
    url: string;
    lightLogo: string;
    darkLogo: string;
  }[] = [];

  return (
    <main className="text-landing-foreground dark:text-landing-foreground-dark pt-4 pb-6 md:pb-12">
      <div className="relative mx-auto flex h-[70vh] max-h-[900px] min-h-[600px] w-full max-w-[1400px] overflow-hidden rounded-2xl border bg-origin-border">
        <Hero />
        <div className="z-2 flex size-full flex-col px-4 max-md:items-center max-md:text-center md:p-12">
          <p className="border-brand/50 text-brand mt-12 w-fit rounded-full border p-2 text-xs font-medium">
            {content.badge}
          </p>
          <h1 className="leading-tighter my-8 text-4xl font-medium xl:mb-12 xl:text-5xl">
            {content.title}
            <br />
            {content.subtitle}{' '}
            <span className="text-brand">{content.highlight}</span>.
          </h1>
          <div className="flex w-fit flex-row flex-wrap items-center justify-center gap-4">
            <Link
              href={getLocalePath(lang, 'docs')}
              className="bg-brand text-brand-foreground hover:bg-brand-200 inline-flex items-center justify-center gap-2 rounded-full px-5 py-3 font-medium tracking-tight transition-colors max-sm:text-sm"
            >
              <BookOpen className="size-4" />
              {content.getStarted}
            </Link>
            {/* ATIUS: replaced AtomGit (China-specific) with single GitHub link */}
            <a
              href="https://github.com/giovannimnz/router-ai-atius"
              target="_blank"
              rel="noreferrer noopener"
              className="bg-fd-secondary text-fd-secondary-foreground hover:bg-fd-accent inline-flex items-center justify-center gap-2 rounded-full border px-5 py-3 font-medium tracking-tight transition-colors max-sm:text-sm"
            >
              <Github className="size-4" />
              {content.github}
            </a>
          </div>
        </div>
      </div>

      {/* Partners Section — empty for Atius (no commercial partners to list) */}
      {partners.length > 0 && (
        <section className="mx-auto mt-12 max-w-[1400px] px-4 text-center">
          <h2 className="text-2xl font-semibold md:text-3xl">
            {content.partnersTitle}
          </h2>
          <p className="text-muted-foreground mt-2 text-sm">
            {content.partnersSubtitle}
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-6 md:gap-10">
            {partners.map((partner) => (
              <a
                key={partner.name}
                href={partner.url}
                target="_blank"
                rel="noopener noreferrer"
                className="opacity-70 grayscale-[50%] transition-all duration-300 hover:opacity-100 hover:grayscale-0"
              >
                <Image
                  src={partner.logo}
                  alt={partner.name}
                  width={72}
                  height={60}
                  className="h-[50px] w-auto md:h-[60px]"
                  loading="lazy"
                  decoding="async"
                />
              </a>
            ))}
          </div>
        </section>
      )}

      {/* Sponsor Partners Section — empty for Atius */}
      {sponsorPartners.length > 0 && (
        <section className="mx-auto mt-16 max-w-[1400px] px-4 text-center">
          <h2 className="text-2xl font-semibold md:text-3xl">
            {content.sponsorPartnersTitle}
          </h2>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-6 md:gap-10">
            {sponsorPartners.map((partner) => (
              <a
                key={partner.name}
                href={partner.url}
                target="_blank"
                rel="noopener noreferrer"
                className="opacity-70 grayscale-[50%] transition-all duration-300 hover:opacity-100 hover:grayscale-0"
              >
                <Image
                  src={partner.lightLogo}
                  alt={partner.name}
                  width={120}
                  height={60}
                  className="block h-[50px] w-auto md:h-[60px] dark:hidden"
                  loading="lazy"
                  decoding="async"
                />
                <Image
                  src={partner.darkLogo}
                  alt={partner.name}
                  width={120}
                  height={60}
                  className="hidden h-[50px] w-auto md:h-[60px] dark:block"
                  loading="lazy"
                  decoding="async"
                />
              </a>
            ))}
          </div>
        </section>
      )}

      {/* Development Contributors Section — points to Atius fork */}
      <section className="mx-auto mt-16 max-w-[1400px] px-4 text-center">
        <h2 className="text-2xl font-semibold md:text-3xl">
          {content.devContributorsTitle}
        </h2>
        <div className="mt-8 flex justify-center">
          <a
            href="https://github.com/giovannimnz/router-ai-atius/graphs/contributors"
            target="_blank"
            rel="noopener noreferrer"
          >
            <img
              src="https://contrib.rocks/image?repo=giovannimnz/router-ai-atius"
              alt="Development Contributors"
              loading="lazy"
              decoding="async"
              className="max-w-full"
            />
          </a>
        </div>
      </section>

      {/* Documentation Contributors Section — points to docs fork */}
      <section className="mx-auto mt-16 max-w-[1400px] px-4 text-center">
        <h2 className="text-2xl font-semibold md:text-3xl">
          {content.docsContributorsTitle}
        </h2>
        <div className="mt-8 flex justify-center">
          <a
            href="https://github.com/QuantumNous/new-api-docs-v1/graphs/contributors"
            target="_blank"
            rel="noopener noreferrer"
          >
            <img
              src="https://contrib.rocks/image?repo=QuantumNous/new-api-docs-v1"
              alt="Documentation Contributors"
              loading="lazy"
              decoding="async"
              className="max-w-full"
            />
          </a>
        </div>
      </section>

      <AntifraudDialog lang={lang} />
    </main>
  );
}

export async function generateStaticParams() {
  return i18n.languages.map((lang) => ({ lang }));
}
