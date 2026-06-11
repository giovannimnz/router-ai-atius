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
import { useEffect, useState } from 'react'
import { ExternalLink, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { tryPrettyJson } from '@/lib/utils'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { completeCodexOAuth, startCodexOAuth } from '../../api'

type CodexOAuthDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onKeyGenerated: (key: string) => void
}

export function CodexOAuthDialog({
  open,
  onOpenChange,
  onKeyGenerated,
}: CodexOAuthDialogProps) {
  const { t } = useTranslation()

  const [state, setState] = useState({
    authorizeUrl: '',
    authCode: '',
    isStarting: false,
    isCompleting: false,
  })

  useEffect(() => {
    if (!open) {
      setState({
        authorizeUrl: '',
        authCode: '',
        isStarting: false,
        isCompleting: false,
      })
    }
  }, [open])

  const handleStart = async () => {
    setState((prev) => ({ ...prev, isStarting: true }))
    try {
      const res = await startCodexOAuth()
      if (!res.success) {
        throw new Error(res.message || 'Failed to start OAuth')
      }

      const url = res.data?.authorize_url || ''
      if (!url) {
        throw new Error('Missing authorize_url in response')
      }

      setState((prev) => ({ ...prev, authorizeUrl: url }))
      try {
        window.open(url, '_blank', 'noopener,noreferrer')
        toast.success(t('Opened authorization page'))
      } catch (error) {
        console.warn('Failed to open authorization page:', error)
        toast.warning(t('Please manually copy and open the authorization link'))
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('OAuth start failed')
      )
    } finally {
      setState((prev) => ({ ...prev, isStarting: false }))
    }
  }

  const handleComplete = async () => {
    if (!state.authCode.trim()) return
    setState((prev) => ({ ...prev, isCompleting: true }))
    try {
      const res = await completeCodexOAuth(state.authCode.trim())
      if (!res.success) {
        throw new Error(res.message || 'OAuth failed')
      }

      const rawKey = res.data?.key || ''
      if (!rawKey) {
        throw new Error('Missing key in response')
      }

      onKeyGenerated(tryPrettyJson(rawKey))
      toast.success(t('Credential generated'))
      onOpenChange(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('OAuth failed'))
    } finally {
      setState((prev) => ({ ...prev, isCompleting: false }))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>{t('Codex Authorization')}</DialogTitle>
          <DialogDescription>
            {t(
              'Authorize the router to use your Codex Pro subscription.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-4'>
          <Alert>
            <AlertDescription>
              {t(
                '1) Click "Open authorization page" and log into your OpenAI account. 2) After login, the browser redirects to a page that fails to load — this is normal. 3) From the address bar URL, locate &p=arameter code=... and copy ONLY the code value. 4) Paste it below and click "Generate credential".'
              )}
            </AlertDescription>
          </Alert>

          <div className='flex flex-wrap gap-2'>
            <Button onClick={handleStart} disabled={state.isStarting}>
              {state.isStarting ? (
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              ) : (
                <ExternalLink className='mr-2 h-4 w-4' />
              )}
              {t('Open authorization page')}
            </Button>
          </div>

          <div className='space-y-2'>
            <div className='text-sm font-medium'>{t('Authorization code')}</div>
            <Input
              value={state.authCode}
              onChange={(e) =>
                setState((prev) => ({ ...prev, authCode: e.target.value }))
              }
              placeholder={t(
                'Paste the code parameter from the URL'
              )}
              autoComplete='off'
              spellCheck={false}
            />
            <div className='text-muted-foreground text-xs'>
              {t(
                'Find code=... in the address bar after login. Only the value, not the full URL.'
              )}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={state.isStarting || state.isCompleting}
          >
            {t('Cancel')}
          </Button>
          <Button
            onClick={handleComplete}
            disabled={!state.authCode.trim() || state.isCompleting}
          >
            {state.isCompleting && (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            )}
            {state.isCompleting
              ? t('Generating...')
              : t('Generate credential')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
