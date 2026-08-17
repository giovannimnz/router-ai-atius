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
import { Textarea } from '@/components/ui/textarea'

import type { CodexDeviceAuthorization } from '../../types'

interface CodexRegenerateDialogProps {
  open: boolean
  channelName: string
  deviceAuthorization?: CodexDeviceAuthorization
  isStarting: boolean
  onOpenChange: (open: boolean) => void
  onPollDevice: () => Promise<'pending' | 'completed'>
  onStartBrowserFallback: () => Promise<string>
  onComplete: (input: string) => Promise<boolean>
}

export function CodexRegenerateDialog({
  open,
  channelName,
  deviceAuthorization,
  isStarting,
  onOpenChange,
  onPollDevice,
  onStartBrowserFallback,
  onComplete,
}: CodexRegenerateDialogProps) {
  const { t } = useTranslation()
  const [input, setInput] = useState('')
  const [isCompleting, setIsCompleting] = useState(false)
  const [isPolling, setIsPolling] = useState(false)
  const [isStartingFallback, setIsStartingFallback] = useState(false)
  const [browserAuthorizeUrl, setBrowserAuthorizeUrl] = useState('')
  const [flowError, setFlowError] = useState('')

  useEffect(() => {
    if (!open || !deviceAuthorization || browserAuthorizeUrl) return
    let active = true
    let inFlight = false
    const poll = async () => {
      if (!active || inFlight) return
      inFlight = true
      setIsPolling(true)
      try {
        const status = await onPollDevice()
        if (active && status === 'completed') {
          setInput('')
          setBrowserAuthorizeUrl('')
          setFlowError('')
          onOpenChange(false)
        }
      } catch (error) {
        if (active) {
          setFlowError(
            error instanceof Error
              ? error.message
              : t('Device authorization polling failed')
          )
        }
      } finally {
        inFlight = false
        if (active) setIsPolling(false)
      }
    }
    const interval = window.setInterval(
      poll,
      Math.max(2, deviceAuthorization.interval_seconds) * 1000
    )
    return () => {
      active = false
      window.clearInterval(interval)
    }
  }, [
    browserAuthorizeUrl,
    deviceAuthorization,
    onOpenChange,
    onPollDevice,
    open,
    t,
  ])

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && !isCompleting) {
      setInput('')
      setBrowserAuthorizeUrl('')
      setFlowError('')
    }
    onOpenChange(nextOpen)
  }

  const handleStartBrowserFallback = async () => {
    setFlowError('')
    setIsStartingFallback(true)
    try {
      setBrowserAuthorizeUrl(await onStartBrowserFallback())
    } catch (error) {
      setFlowError(
        error instanceof Error
          ? error.message
          : t('Failed to start browser authorization')
      )
    } finally {
      setIsStartingFallback(false)
    }
  }

  const handleComplete = async () => {
    const transientInput = input.trim()
    if (!transientInput) return

    setIsCompleting(true)
    try {
      if (await onComplete(transientInput)) {
        setInput('')
        setBrowserAuthorizeUrl('')
        setFlowError('')
        onOpenChange(false)
      }
    } finally {
      setIsCompleting(false)
    }
  }

  let deviceAuthorizationContent: ReactNode = null
  if (isStarting && !deviceAuthorization) {
    deviceAuthorizationContent = (
      <div className='flex items-center gap-2 py-4 text-sm'>
        <Loader2 className='h-4 w-4 animate-spin' />
        {t('Starting device authorization...')}
      </div>
    )
  } else if (deviceAuthorization) {
    deviceAuthorizationContent = (
      <div className='space-y-3 rounded-md border p-4'>
        <div>
          <p className='text-sm font-medium'>{t('Device login code')}</p>
          <code className='mt-1 block text-2xl font-semibold tracking-widest'>
            {deviceAuthorization.user_code}
          </code>
        </div>
        <Button
          render={
            <a
              href={deviceAuthorization.verification_url}
              target='_blank'
              rel='noopener noreferrer'
            />
          }
        >
          {t('Open device login')}
          <ExternalLink className='ml-2 h-4 w-4' />
        </Button>
        <p className='text-muted-foreground text-xs'>
          {isPolling
            ? t('Waiting for OpenAI authorization...')
            : t('This window will detect authorization automatically.')}
        </p>
      </div>
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

        <div className='space-y-3 border-t pt-4'>
          <div>
            <p className='text-sm font-medium'>
              {t('Manual browser fallback')}
            </p>
            <p className='text-muted-foreground text-xs'>
              {t(
                'Use this only if device login is unavailable. The callback field remains accessible without popups.'
              )}
            </p>
          </div>
          {!browserAuthorizeUrl ? (
            <Button
              type='button'
              variant='outline'
              onClick={handleStartBrowserFallback}
              disabled={isStartingFallback}
            >
              {isStartingFallback && (
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              )}
              {t('Prepare browser fallback')}
            </Button>
          ) : (
            <Button
              variant='outline'
              render={
                <a
                  href={browserAuthorizeUrl}
                  target='_blank'
                  rel='noopener noreferrer'
                />
              }
            >
              {t('Open browser authorization')}
              <ExternalLink className='ml-2 h-4 w-4' />
            </Button>
          )}
          <label htmlFor='codex-oauth-callback' className='text-sm font-medium'>
            {t('Authorization callback')}
          </label>
          <Textarea
            id='codex-oauth-callback'
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder={t(
              'Paste the final callback URL or the code#state pair.'
            )}
            rows={4}
            autoComplete='off'
            spellCheck={false}
            disabled={isCompleting}
          />
          <p className='text-muted-foreground text-xs'>
            {t('Tokens are never displayed on this screen.')}
          </p>
          {flowError && (
            <p className='text-destructive text-sm' role='alert'>
              {flowError}
            </p>
          )}
        </div>
        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => handleOpenChange(false)}
            disabled={isCompleting}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={handleComplete}
            disabled={!input.trim() || isCompleting}
          >
            {isCompleting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Complete regeneration')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
