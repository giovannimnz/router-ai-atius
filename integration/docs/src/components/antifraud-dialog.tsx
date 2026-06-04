'use client';

import { type ReactNode, useCallback, useEffect, useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { ShieldAlert } from 'lucide-react';

const STORAGE_KEY_PREFIX = 'newapi-antifraud-acknowledged';
const COOLDOWN_SECONDS = 10;

const B = ({ children }: { children: ReactNode }) => (
  <strong className="text-fd-foreground">{children}</strong>
);

const i18nContent: Record<
  string,
  {
    title: string;
    sections: { heading: string; body: ReactNode }[];
    advice: { heading: string; body: ReactNode };
    button: string;
    buttonCountdown: (s: number) => string;
  }
> = {
  zh: {
    title: '关于打击假冒 New API 官方及侵权行为的严正声明',
    sections: [
      {
        heading: '第一：官方绝未公开发售接口，谨防受骗！',
        body: <>New API 为纯粹的开源项目，我司<B>从未授权任何机构或个人开展代理或销售</B>。凡在外打着&ldquo;New API 官方 / 合作方 / 自营&rdquo;旗号售卖 API 额度或中转服务的，<B>100% 均为假冒李鬼</B>，请广大用户务必擦亮眼睛！</>,
      },
      {
        heading: '第二：开源不等于弃权，法务已持续监控取证！',
        body: <>针对违规闭源魔改代码、涉嫌窃取隐私与虚假宣传的黑产平台，<B>我司不承担其带来的任何技术与法律责任</B>。同时，我司已委派专业法务团队持续监控取证，针对冒用名义、诈骗等侵权行为，将视情况通过<B>行政举报及诉讼等法律手段</B>依法追责。</>,
      },
    ],
    advice: {
      heading: '官方忠告',
      body: <>强烈建议开发者前往官方 GitHub 仓库拉取源码进行<B>本地私有化部署</B>，牢牢掌握自己的网关数据，切勿将核心业务接入不明黑产站点！</>,
    },
    button: '我已了解，认准官方开源',
    buttonCountdown: (s) => `请阅读以上声明 (${s}s)`,
  },
  en: {
    title: 'Official Statement Against Counterfeit New API Services & Infringement',
    sections: [
      {
        heading: 'We have NEVER publicly sold API access — beware of scams!',
        body: <>New API is a purely open-source project. We have <B>never authorized any organization or individual to act as an agent or reseller</B>. Anyone selling API quotas or relay services under the name &ldquo;New API Official / Partner / Self-operated&rdquo; is <B>100% fraudulent</B>.</>,
      },
      {
        heading: 'Open source does not mean waiving rights — legal action is ongoing!',
        body: <>We bear <B>no technical or legal responsibility</B> for unauthorized closed-source forks, privacy-stealing modifications, or fraudulent platforms. Our legal team is actively monitoring and collecting evidence, and will pursue <B>administrative complaints and litigation</B> against impersonation and fraud as appropriate.</>,
      },
    ],
    advice: {
      heading: 'Official Recommendation',
      body: <>We strongly recommend developers pull the source code from the official GitHub repository for <B>private local deployment</B>. Maintain full control of your gateway data and never connect critical services to unknown third-party platforms!</>,
    },
    button: 'I understand — trust only the official open source',
    buttonCountdown: (s) => `Please read the statement above (${s}s)`,
  },
  ja: {
    title: 'New API の偽造サービスおよび権利侵害に対する公式声明',
    sections: [
      {
        heading: '公式は API を一切販売していません — 詐欺にご注意ください！',
        body: <>New API は純粋なオープンソースプロジェクトです。<B>いかなる機関や個人にも代理販売を許可したことはありません</B>。「New API 公式 / パートナー / 自営」を名乗り API クォータや中継サービスを販売するものは、<B>100% 偽物です</B>。</>,
      },
      {
        heading: 'オープンソースは権利放棄ではありません — 法的措置を継続中！',
        body: <>無断でクローズドソース化した改変コード、プライバシー窃取、虚偽宣伝を行う不正プラットフォームについて、<B>当社は一切の技術的・法的責任を負いません</B>。法務チームが継続的に監視・証拠収集を行い、なりすましや詐欺行為に対して<B>行政通報および訴訟</B>を通じて法的責任を追及します。</>,
      },
    ],
    advice: {
      heading: '公式からの推奨',
      body: <>開発者の皆様には、公式 GitHub リポジトリからソースコードを取得し、<B>ローカルでプライベートデプロイ</B>を行うことを強くお勧めします。ゲートウェイデータを自身で管理し、不明な第三者プラットフォームには絶対に接続しないでください！</>,
    },
    button: '了解しました — 公式オープンソースのみを信頼します',
    buttonCountdown: (s) => `上記の声明をお読みください (${s}s)`,
  },
};

export function AntifraudDialog({ lang }: { lang: string }) {
  const storageKey = `${STORAGE_KEY_PREFIX}-${lang}`;
  const [open, setOpen] = useState(false);
  const [countdown, setCountdown] = useState(COOLDOWN_SECONDS);

  useEffect(() => {
    if (localStorage.getItem(storageKey) !== 'true') {
      setOpen(true);
      setCountdown(COOLDOWN_SECONDS);
    } else {
      setOpen(false);
    }
  }, [storageKey]);

  useEffect(() => {
    if (!open || countdown <= 0) return;

    const timer = window.setTimeout(() => {
      setCountdown((prev) => prev - 1);
    }, 1000);

    return () => window.clearTimeout(timer);
  }, [open, countdown]);

  const handleAcknowledge = useCallback(() => {
    localStorage.setItem(storageKey, 'true');
    setOpen(false);
  }, [storageKey]);

  const content = i18nContent[lang] || i18nContent.en;
  const canConfirm = countdown <= 0;

  return (
    <Dialog.Root open={open}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=closed]:animate-out data-[state=closed]:fade-out-0" />
        <Dialog.Content
          onPointerDownOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => e.preventDefault()}
          onInteractOutside={(e) => e.preventDefault()}
          className="fixed top-1/2 left-1/2 z-50 w-[calc(100vw-2rem)] max-w-xl -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-2xl border bg-fd-background shadow-2xl data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95"
        >
          {/* Red accent strip */}
          <div className="h-1 w-full bg-gradient-to-r from-red-500 via-red-400 to-amber-400" />

          <div className="max-h-[80vh] overflow-y-auto px-6 pt-5 pb-6 sm:px-8">
            {/* Header */}
            <div className="mb-5 flex items-start gap-3.5">
              <div className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-red-500/10">
                <ShieldAlert className="size-[18px] text-red-500" />
              </div>
              <Dialog.Title className="text-[15px] font-bold leading-snug tracking-tight text-fd-foreground sm:text-base">
                {content.title}
              </Dialog.Title>
            </div>

            {/* Sections */}
            <div className="space-y-4">
              {content.sections.map((section, i) => (
                <div
                  key={i}
                  className="rounded-lg border border-fd-border/60 bg-fd-muted/30 p-4"
                >
                  <div className="mb-2 flex items-center gap-2">
                    <span className="inline-block size-2 shrink-0 rounded-full bg-red-500" />
                    <h3 className="text-[13px] font-semibold leading-snug text-fd-foreground sm:text-sm">
                      {section.heading}
                    </h3>
                  </div>
                  <p className="text-[13px] leading-[1.8] text-fd-muted-foreground">
                    {section.body}
                  </p>
                </div>
              ))}

              {/* Advice */}
              <div className="rounded-lg border border-amber-400/40 bg-amber-50 p-4 dark:bg-amber-500/5">
                <div className="mb-2 flex items-center gap-2">
                  <span className="inline-block size-2 shrink-0 rounded-full bg-amber-500" />
                  <h3 className="text-[13px] font-semibold text-amber-700 dark:text-amber-400 sm:text-sm">
                    {content.advice.heading}
                  </h3>
                </div>
                <p className="text-[13px] leading-[1.8] text-amber-900/70 dark:text-amber-200/70">
                  {content.advice.body}
                </p>
              </div>
            </div>

            {/* Button */}
            <button
              type="button"
              disabled={!canConfirm}
              onClick={handleAcknowledge}
              className="mt-6 w-full rounded-lg bg-brand px-4 py-3 text-sm font-semibold text-brand-foreground transition-colors hover:bg-brand-200 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {canConfirm ? content.button : content.buttonCountdown(countdown)}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
