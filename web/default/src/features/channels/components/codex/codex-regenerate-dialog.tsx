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
import { Loader2 } from 'lucide-react'
import { useState } from 'react'
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

interface CodexRegenerateDialogProps {
  open: boolean
  channelName: string
  onOpenChange: (open: boolean) => void
  onComplete: (input: string) => Promise<boolean>
}

type OpenAuthorizationWindow = (
  url: string,
  target: string,
  features: string
) => Window | null

export function openOAuthAuthorizationWindow(
  authorizeUrl: string,
  openWindow: OpenAuthorizationWindow
) {
  return openWindow(authorizeUrl, '_blank', 'noopener,noreferrer') !== null
}

export function CodexRegenerateDialog({
  open,
  channelName,
  onOpenChange,
  onComplete,
}: CodexRegenerateDialogProps) {
  const { t } = useTranslation()
  const [input, setInput] = useState('')
  const [isCompleting, setIsCompleting] = useState(false)

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && !isCompleting) setInput('')
    onOpenChange(nextOpen)
  }

  const handleComplete = async () => {
    const transientInput = input.trim()
    if (!transientInput) return

    setIsCompleting(true)
    try {
      if (await onComplete(transientInput)) {
        setInput('')
        onOpenChange(false)
      }
    } finally {
      setIsCompleting(false)
    }
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
              'Complete OpenAI authorization for channel {{channel}} in the new browser tab.',
              { channel: channelName }
            )}
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-2'>
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
