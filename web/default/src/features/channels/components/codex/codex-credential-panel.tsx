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
import {
  AlertCircle,
  Loader2,
  RefreshCw,
  RotateCw,
  ShieldCheck,
} from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import type { CodexCredentialMetadata } from "../../types";

type Translate = (key: string) => string;

interface CodexCredentialPanelProps {
  metadata?: CodexCredentialMetadata;
  error?: string;
  hasSavedChannel: boolean;
  canManage: boolean;
  isLoading: boolean;
  isRefreshing: boolean;
  isProbing: boolean;
  isRegenerating?: boolean;
  onRefresh: () => void;
  onProbe: () => void;
  onRegenerate: () => void;
  t: Translate;
}

function displayValue(value: string | number | undefined, t: Translate) {
  if (value === undefined || value === "") return t("Not available");
  return String(value);
}

function formatDate(value: string | undefined, t: Translate) {
  if (!value) return t("Not available");
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
}

function hasFutureExpiration(metadata: CodexCredentialMetadata) {
  if (!metadata.expires_at) return false;
  const expiresAt = new Date(metadata.expires_at).getTime();
  return Number.isFinite(expiresAt) && expiresAt > Date.now();
}

export function isCodexChannelType(channelType: number) {
  return channelType === 57;
}

export function CodexCredentialPanel({
  metadata,
  error,
  hasSavedChannel,
  canManage,
  isLoading,
  isRefreshing,
  isProbing,
  isRegenerating = false,
  onRefresh,
  onProbe,
  onRegenerate,
  t,
}: CodexCredentialPanelProps) {
  const upstreamProbeFailed = Boolean(
    metadata?.last_probe_status && metadata.last_probe_status !== "ok",
  );
  const upstreamAuthFailed = Boolean(
    metadata?.last_upstream_auth_error || upstreamProbeFailed,
  );

  return (
    <section className="border-border/60 bg-muted/10 flex flex-col gap-4 rounded-lg border p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <h3 className="flex items-center gap-2 text-sm font-semibold">
            <ShieldCheck className="h-4 w-4" aria-hidden="true" />
            {t("Codex OAuth credential")}
          </h3>
          <p className="text-muted-foreground text-xs">
            {t("Tokens are never displayed on this screen.")}
          </p>
        </div>
        {metadata && (
          <div className="flex flex-wrap gap-2">
            <Badge
              variant={metadata.authenticated ? "secondary" : "destructive"}
            >
              {metadata.authenticated
                ? t("Authenticated")
                : t("Not authenticated")}
            </Badge>
            {metadata.requires_regeneration && (
              <Badge variant="destructive">{t("Requires regeneration")}</Badge>
            )}
            {!metadata.has_refresh_token && (
              <Badge variant="outline">{t("No refresh_token")}</Badge>
            )}
            {metadata.last_probe_status === "ok" && (
              <Badge variant="secondary">{t("Probe OK")}</Badge>
            )}
            {upstreamAuthFailed && (
              <Badge variant="destructive">{t("Upstream error")}</Badge>
            )}
          </div>
        )}
      </div>

      {!hasSavedChannel && (
        <Alert>
          <AlertDescription>
            {t(
              "Save the canonical Codex channel before managing its OAuth credential.",
            )}
          </AlertDescription>
        </Alert>
      )}

      {isLoading && (
        <div className="text-muted-foreground flex items-center gap-2 text-sm">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          {t("Loading credential metadata...")}
        </div>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertCircle aria-hidden="true" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {metadata && (
        <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
          {[
            [
              t("Channel"),
              `${metadata.channel_name} (#${metadata.channel_id})`,
            ],
            [
              t("Account"),
              displayValue(metadata.email || metadata.account_id, t),
            ],
            [t("Expiration"), formatDate(metadata.expires_at, t)],
            [
              t("Refresh token"),
              metadata.has_refresh_token ? t("Available") : t("Unavailable"),
            ],
            [t("Last refresh"), formatDate(metadata.last_refresh, t)],
            [t("Last probe"), formatDate(metadata.last_probe_at, t)],
            [t("Probe status"), displayValue(metadata.last_probe_status, t)],
            [
              t("Upstream status"),
              displayValue(metadata.last_upstream_status, t),
            ],
            [
              t("Last upstream auth error"),
              displayValue(metadata.last_upstream_auth_error, t),
            ],
            [
              t("Regeneration reason"),
              displayValue(metadata.regeneration_reason, t),
            ],
          ].map(([label, value]) => (
            <div key={label} className="min-w-0">
              <dt className="text-muted-foreground text-xs">{label}</dt>
              <dd className="mt-1 break-words font-medium">{value}</dd>
            </div>
          ))}
        </dl>
      )}

      {metadata && !metadata.has_refresh_token && (
        <Alert className="border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-50">
          <AlertDescription>
            <p>
              {t(
                "This credential has no refresh_token and cannot be renewed automatically. Regeneration is the definitive fix.",
              )}
            </p>
            <p className="mt-2">
              {t(
                "Temporary fallback: Codex CLI access_token, without refresh_token or automatic renewal.",
              )}
            </p>
          </AlertDescription>
        </Alert>
      )}

      {metadata && upstreamProbeFailed && hasFutureExpiration(metadata) && (
        <Alert variant="destructive">
          <AlertCircle aria-hidden="true" />
          <AlertDescription>
            {t(
              "Local expiration is still in the future, but the latest upstream probe failed. Regenerate the credential.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={
            !hasSavedChannel ||
            !canManage ||
            isRefreshing ||
            isProbing ||
            isRegenerating
          }
        >
          {isRefreshing ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          {t("Refresh credential")}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onProbe}
          disabled={
            !hasSavedChannel ||
            !canManage ||
            isRefreshing ||
            isProbing ||
            isRegenerating
          }
        >
          {isProbing ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <ShieldCheck className="mr-2 h-4 w-4" />
          )}
          {t("Probe upstream authentication")}
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={onRegenerate}
          disabled={
            !hasSavedChannel ||
            !canManage ||
            isRefreshing ||
            isProbing ||
            isRegenerating
          }
        >
          {isRegenerating ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RotateCw className="mr-2 h-4 w-4" />
          )}
          {t("Regenerate credential")}
        </Button>
      </div>

      {hasSavedChannel && !canManage && (
        <p className="text-muted-foreground text-xs">
          {t(
            "Sensitive-write permission is required to manage this credential.",
          )}
        </p>
      )}
    </section>
  );
}
