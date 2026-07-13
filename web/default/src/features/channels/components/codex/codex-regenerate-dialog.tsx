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
/* eslint-disable react-refresh/only-export-components */
import { ExternalLink, Loader2 } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { CodexDeviceAuthorization } from '../../types'

export interface CodexDevicePollOutcome {
  status: 'pending' | 'completed' | 'terminal' | 'cancelled' | 'expired'
  retryAfterMs?: number
  error?: unknown
}

interface CodexRegenerateDialogProps {
  open: boolean
  channelName: string
  deviceAuthorization?: CodexDeviceAuthorization
  deviceStartError?: string
  isStarting: boolean
  isCancelling: boolean
  onOpenChange: (open: boolean) => void | Promise<void>
  onExpireDevice: () => Promise<void>
  onPollDevice: (signal: AbortSignal) => Promise<CodexDevicePollOutcome>
  onRestartDevice: () => Promise<void>
}

export interface CodexDevicePollingScheduler {
  now: () => number
  setTimeout: (callback: () => void | Promise<void>, delayMs: number) => number
  clearTimeout: (timerId: number) => void
}

interface StartCodexDevicePollingOptions {
  expiresAt: string
  intervalMs: number
  poll: (signal: AbortSignal) => Promise<CodexDevicePollOutcome>
  onCompleted: () => void
  onExpired: () => void
  onError: (error: unknown) => void
  onPollingChange: (polling: boolean) => void
  scheduler?: CodexDevicePollingScheduler
}

const browserPollingScheduler: CodexDevicePollingScheduler = {
  now: () => Date.now(),
  setTimeout: (callback, delayMs) => window.setTimeout(callback, delayMs),
  clearTimeout: (timerId) => window.clearTimeout(timerId),
}

const CODEX_AUTHORIZATION_ORIGIN = 'https://auth.openai.com'

export function getTrustedCodexAuthorizationUrl(
  candidate: string
): string | null {
  try {
    const url = new URL(candidate)
    if (
      url.origin !== CODEX_AUTHORIZATION_ORIGIN ||
      url.pathname !== '/codex/device' ||
      url.username ||
      url.password
    ) {
      return null
    }
    return url.toString()
  } catch {
    return null
  }
}

export function startCodexDevicePolling({
  expiresAt,
  intervalMs,
  poll,
  onCompleted,
  onExpired,
  onError,
  onPollingChange,
  scheduler = browserPollingScheduler,
}: StartCodexDevicePollingOptions) {
  const expiresAtMs = Date.parse(expiresAt)
  let active = true
  let pollTimerId: number | undefined
  let expiryTimerId: number | undefined
  let pollController: AbortController | undefined

  const clearTimers = () => {
    if (pollTimerId !== undefined) scheduler.clearTimeout(pollTimerId)
    if (expiryTimerId !== undefined) scheduler.clearTimeout(expiryTimerId)
    pollTimerId = undefined
    expiryTimerId = undefined
  }

  const stop = () => {
    if (!active) return
    active = false
    clearTimers()
    pollController?.abort()
    pollController = undefined
    onPollingChange(false)
  }

  const expire = () => {
    if (!active) return
    stop()
    onExpired()
  }

  if (!Number.isFinite(expiresAtMs) || expiresAtMs <= scheduler.now()) {
    expire()
    return stop
  }

  const pollOnce = async () => {
    if (!active) return
    pollController = new AbortController()
    try {
      const outcome = await poll(pollController.signal)
      if (!active) return
      if (outcome.status === 'completed') {
        stop()
        onCompleted()
        return
      }
      if (
        outcome.status === 'terminal' ||
        outcome.status === 'cancelled' ||
        outcome.status === 'expired'
      ) {
        stop()
        onError(
          outcome.error ?? new Error(`device authorization ${outcome.status}`)
        )
        return
      }
      pollTimerId = scheduler.setTimeout(
        pollOnce,
        Math.max(1, outcome.retryAfterMs ?? intervalMs)
      )
    } catch (error) {
      if (!active) return
      pollTimerId = scheduler.setTimeout(pollOnce, Math.max(1, intervalMs))
    } finally {
      pollController = undefined
    }
  }

  onPollingChange(true)
  expiryTimerId = scheduler.setTimeout(expire, expiresAtMs - scheduler.now())
  pollTimerId = scheduler.setTimeout(pollOnce, Math.max(1, intervalMs))

  return stop
}

export function CodexDeviceAuthorizationDetails({
  deviceAuthorization,
  deviceFlowError,
  isPolling,
  onRestartDevice,
  translate,
}: {
  deviceAuthorization: CodexDeviceAuthorization
  deviceFlowError: string
  isPolling: boolean
  onRestartDevice: () => void | Promise<void>
  translate: (key: string) => string
}) {
  const verificationUrl = getTrustedCodexAuthorizationUrl(
    deviceAuthorization.verification_url
  )

  return (
    <div className='space-y-3 rounded-md border p-4'>
      <div>
        <p className='text-sm font-medium'>{translate('Device login code')}</p>
        <code className='mt-1 block text-2xl font-semibold tracking-widest'>
          {deviceAuthorization.user_code}
        </code>
        <p className='text-warning mt-2 text-xs font-medium'>
          {translate(
            'Continue only if you started this login on this screen. Enter the code only at auth.openai.com and never share it.'
          )}
        </p>
      </div>
      {verificationUrl ? (
        <Button
          render={
            <a
              href={verificationUrl}
              target='_blank'
              rel='noopener noreferrer'
            />
          }
        >
          {translate('Open device login')}
          <ExternalLink className='ml-2 h-4 w-4' />
        </Button>
      ) : (
        <p className='text-destructive text-sm' role='alert'>
          {translate('The authorization URL is not secure')}
        </p>
      )}
      {deviceFlowError ? (
        <div className='space-y-2'>
          <p className='text-destructive text-sm' role='alert'>
            {deviceFlowError}
          </p>
          <Button type='button' variant='outline' onClick={onRestartDevice}>
            {translate('Restart device authorization')}
          </Button>
        </div>
      ) : (
        <p className='text-muted-foreground text-xs'>
          {isPolling
            ? translate('Waiting for OpenAI authorization...')
            : translate('This window will detect authorization automatically.')}
        </p>
      )}
    </div>
  )
}

export function CodexRegenerateDialog({
  open,
  channelName,
  deviceAuthorization,
  deviceStartError,
  isStarting,
  isCancelling,
  onOpenChange,
  onExpireDevice,
  onPollDevice,
  onRestartDevice,
}: CodexRegenerateDialogProps) {
  const { t } = useTranslation()
  const [isPolling, setIsPolling] = useState(false)
  const [deviceFlowError, setDeviceFlowError] = useState('')

  useEffect(() => {
    if (!open || !deviceAuthorization) return
    setDeviceFlowError('')
    return startCodexDevicePolling({
      expiresAt: deviceAuthorization.expires_at,
      intervalMs: Math.max(2, deviceAuthorization.interval_seconds) * 1000,
      poll: onPollDevice,
      onCompleted: () => {
        setDeviceFlowError('')
        onOpenChange(false)
      },
      onExpired: () => {
        void onExpireDevice().finally(() => {
          setDeviceFlowError(t('Device authorization expired. Start again.'))
        })
      },
      onError: (error) => {
        void onExpireDevice().finally(() => {
          setDeviceFlowError(
            error instanceof Error
              ? error.message
              : t('Device authorization polling failed')
          )
        })
      },
      onPollingChange: setIsPolling,
    })
  }, [deviceAuthorization, onOpenChange, onExpireDevice, onPollDevice, open, t])

  const handleOpenChange = async (nextOpen: boolean) => {
    if (!nextOpen) {
      setDeviceFlowError('')
    }
    await onOpenChange(nextOpen)
  }

  const handleRestartDevice = async () => {
    setDeviceFlowError('')
    await onRestartDevice()
  }

  let deviceAuthorizationContent: ReactNode = null
  if (isStarting && !deviceAuthorization) {
    deviceAuthorizationContent = (
      <div className='flex items-center gap-2 py-4 text-sm'>
        <Loader2 className='h-4 w-4 animate-spin' />
        {t('Starting device authorization...')}
      </div>
    )
  } else if (!deviceAuthorization) {
    deviceAuthorizationContent = (
      <div className='space-y-3 rounded-md border p-4'>
        <p className='text-destructive text-sm' role='alert'>
          {deviceStartError ||
            t('Device authorization could not be started. Start again.')}
        </p>
        <Button type='button' onClick={handleRestartDevice}>
          {t('Restart device authorization')}
        </Button>
      </div>
    )
  } else if (deviceAuthorization) {
    deviceAuthorizationContent = (
      <CodexDeviceAuthorizationDetails
        deviceAuthorization={deviceAuthorization}
        deviceFlowError={deviceFlowError}
        isPolling={isPolling}
        onRestartDevice={handleRestartDevice}
        translate={t}
      />
    )
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {t('Generate a new Router-owned OAuth credential')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Authorize channel {{channel}} with a Router-owned OpenAI session.',
              { channel: channelName }
            )}
          </DialogDescription>
        </DialogHeader>

        {deviceAuthorizationContent}
        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            disabled={isCancelling}
            onClick={() => void handleOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
