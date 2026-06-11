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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  CheckCircle2,
  Copy,
  Eye,
  EyeOff,
  Loader2,
  Upload,
  Terminal,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useQueryClient } from '@tanstack/react-query'
import { channelsQueryKeys } from '../../../lib'
import {
  startCodexDeviceAuth,
  pollCodexDeviceAuth,
  uploadCodexDeviceAuthJSON,
  startCodexOAuth,
  completeCodexOAuth,
  fetchCodexModels,
} from '../../../api'
import { formatModelsArray } from '../../../lib'

type CodexOAuthSectionProps = {
  channelId: number | null
  isEditing: boolean
  form: {
    getValues: () => { key?: string; models?: string }
    setValue: (field: 'key' | 'models', value: string, opts?: { shouldDirty?: boolean }) => void
    watch: (callback: (value: unknown, info: { name?: string }) => void) => () => void
  }
}

type AuthState = 'idle' | 'device_code_shown' | 'device_polling' | 'code_paste' | 'completing' | 'complete' | 'error'

type CredentialInfo = {
  email?: string
  accountId?: string
  expiresAt?: string
  lastRefresh?: string
  accessTokenPreview?: string
}

function maskToken(token: string): string {
  if (token.length <= 12) return token
  return token.slice(0, 6) + '••••••••' + token.slice(-4)
}

function parseExistingKey(key: string | undefined): CredentialInfo | null {
  if (!key?.trim() || !key.startsWith('{')) return null
  try {
    const parsed = JSON.parse(key)
    return {
      email: parsed.email || '',
      accountId: parsed.account_id || '',
      expiresAt: parsed.expired || '',
      lastRefresh: parsed.last_refresh || '',
      accessTokenPreview: parsed.access_token ? maskToken(parsed.access_token) : '',
    }
  } catch {
    return null
  }
}

export function CodexOAuthSection({
  isEditing,
  form,
}: CodexOAuthSectionProps) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const queryClient = useQueryClient()

  const [authState, setAuthState] = useState<AuthState>('idle')
  const [authMethod, setAuthMethod] = useState<'device' | 'pkce'>(() => {
    return (localStorage.getItem('codex-auth-method') as 'device' | 'pkce') || 'device'
  })
  const [deviceCode, setDeviceCode] = useState('')
  const [deviceURL, setDeviceURL] = useState('')
  const [sessionID, setSessionID] = useState('')
  const [pkceCode, setPkceCode] = useState('')
  const [jsonPaste, setJsonPaste] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [credentialInfo, setCredentialInfo] = useState<CredentialInfo | null>(null)
  const [showToken, setShowToken] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Existing credential for editing. The form is populated asynchronously
  // when the channel data loads (parent useEffect that calls form.reset()).
  // We use form.watch (the actual RHF form supports it) to re-derive the
  // memo on every key change.
  const [keyValue, setKeyValue] = useState<string | undefined>(() => form.getValues().key)
  useEffect(() => {
    setKeyValue(form.getValues().key)
    // RHF form.watch is the canonical subscription API — the form ref
    // is stable but the value updates fire on reset(). We capture name === 'key'.
    if (!isEditing) return
    let unsub: (() => void) | null = null
    try {
      unsub = form.watch((value, info) => {
        if (info?.name === 'key' || typeof value === 'string') {
          setKeyValue((value as string | undefined) ?? form.getValues().key)
        }
      })
    } catch {
      // Some injected test form refs may not implement watch — fall back
      // to polling form.getValues() once on isEditing flip.
    }
    return () => {
      if (unsub) unsub()
    }
  }, [form, isEditing])
  const existingKey = useMemo(() => {
    if (!isEditing) return null
    return parseExistingKey(keyValue)
  }, [isEditing, keyValue])

  // ── Load models from upstream ──
  const loadModels = useCallback(async () => {
    const rawKey = form.getValues().key
    if (!rawKey) return
    try {
      const res = await fetchCodexModels(rawKey)
      if (res.success && res.data?.models && res.data.models.length > 0) {
        form.setValue('models', formatModelsArray(res.data.models), { shouldDirty: true })
      }
      queryClient.invalidateQueries({ queryKey: ['channel_models'] })
      queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
    } catch {
      // Silently fail
    }
  }, [form, queryClient])

  useEffect(() => {
    if (existingKey) {
      setCredentialInfo(existingKey)
      setAuthState('complete')
      loadModels()
    }
  }, [existingKey, loadModels])

  // Persist auth method preference
  useEffect(() => {
    localStorage.setItem('codex-auth-method', authMethod)
  }, [authMethod])

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [])

  // ── Device Auth: Start ──
  const handleDeviceStart = useCallback(async () => {
    setIsLoading(true)
    try {
      const res = await startCodexDeviceAuth()
      if (!res.success || !res.data) {
        throw new Error(res.message || t('Failed to generate authentication code'))
      }

      setSessionID(res.data.session_id)
      setDeviceCode(res.data.user_code)
      setDeviceURL(res.data.verification_url)
      setAuthState('device_code_shown')

      // Auto-copy the short code so the user can paste it on the OpenAI
      // auth page without reaching for the Copy button. Best-effort —
      // a clipboard permission denial should not block the flow.
      try {
        await copyToClipboard(res.data.user_code)
        toast.success(t('Code copied to clipboard automatically'))
      } catch {
        toast.info(t('Authentication code generated'))
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Authentication flow failed'))
      setAuthState('error')
    } finally {
      setIsLoading(false)
    }
  }, [t, copyToClipboard])

  // ── Device Auth: Poll ──
  const startPolling = useCallback(() => {
    if (!sessionID) return
    setAuthState('device_polling')

    pollRef.current = setInterval(async () => {
      try {
        const res = await pollCodexDeviceAuth(sessionID)
        if (!res.success) return

        if (res.data?.status === 'complete') {
          if (pollRef.current) clearInterval(pollRef.current)
          const rawKey = res.data.key || ''
          if (!rawKey) throw new Error(t('Missing key'))
          form.setValue('key', rawKey, { shouldDirty: true })

          const info = parseExistingKey(rawKey)
          if (info) setCredentialInfo(info)

          setAuthState('complete')
          toast.success(t('Codex authorized'))
          loadModels()
        }
      } catch {
        // Keep polling
      }
    }, 3000)
  }, [sessionID, t, form])

  // ── Device Auth: Upload JSON ──
  const handleJSONUpload = useCallback(async (json: string) => {
    if (!json.trim()) return
    setIsLoading(true)
    try {
      const res = await uploadCodexDeviceAuthJSON(json.trim())
      if (!res.success || !res.data?.key) throw new Error(res.message || t('Upload failed'))

      const rawKey = res.data.key
      form.setValue('key', rawKey, { shouldDirty: true })

      const info = parseExistingKey(rawKey)
      if (info) setCredentialInfo(info)

      setAuthState('complete')
      toast.success(t('Codex credential saved'))
      loadModels()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Upload failed'))
    } finally {
      setIsLoading(false)
    }
  }, [t, form])

  // ── File input handler ──
  const handleFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = (ev) => {
      handleJSONUpload(ev.target?.result as string)
    }
    reader.readAsText(file)
  }, [handleJSONUpload])

  // ── PKCE: Code paste ──
  const handlePkceComplete = useCallback(async () => {
    if (!pkceCode.trim()) return
    setIsLoading(true)
    setAuthState('completing')
    try {
      const res = await completeCodexOAuth(pkceCode.trim())
      if (!res.success) throw new Error(res.message || 'OAuth failed')

      const rawKey = res.data?.key || ''
      if (!rawKey) throw new Error('Missing key')

      form.setValue('key', rawKey, { shouldDirty: true })
      const info = parseExistingKey(rawKey)
      if (info) setCredentialInfo(info)

      setAuthState('complete')
      toast.success(t('Codex credential generated'))
      loadModels()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('OAuth failed'))
      setAuthState('code_paste')
    } finally {
      setIsLoading(false)
    }
  }, [pkceCode, t, form])

  const isExpired = useMemo(() => {
    if (!credentialInfo?.expiresAt) return false
    return new Date(credentialInfo.expiresAt) < new Date()
  }, [credentialInfo])

  return (
    <div className='border-border/60 flex flex-col gap-4 border-y py-4'>
      {/* Header */}
      <div className='flex items-center justify-between'>
        <div className='flex flex-col gap-0.5'>
          <div className='text-sm font-semibold'>
            {t('OpenAI Codex OAuth Authorization')}
          </div>
        </div>
        {authState === 'complete' && credentialInfo && (
          <Badge variant={isExpired ? 'destructive' : 'default'} className='shrink-0'>
            {isExpired ? t('Expired') : credentialInfo.email || t('Authenticated')}
          </Badge>
        )}
      </div>

      {/* ── Method toggle (when idle) ── */}
      {(authState === 'idle' || authState === 'error') && !existingKey && (
        <div className='flex items-center gap-2 text-sm'>
          <span className='text-muted-foreground'>{t('Method')}</span>
          <button
            type='button'
            className={`underline ${authMethod === 'device' ? 'font-semibold' : 'text-muted-foreground'}`}
            onClick={() => setAuthMethod('device')}
          >
            {t('Authenticate with code')}
          </button>
          <span className='text-muted-foreground'>|</span>
          <button
            type='button'
            className={`underline ${authMethod === 'pkce' ? 'font-semibold' : 'text-muted-foreground'}`}
            onClick={() => setAuthMethod('pkce')}
          >
            {t('Authenticate with callback')}
          </button>
        </div>
      )}

      {/* ═══════ DEVICE AUTH (PRIMARY) ═══════ */}
      {authMethod === 'device' && !existingKey && (
        <>
          {/* Device code shown */}
          {authState === 'device_code_shown' && (
            <div className='bg-muted/50 flex flex-col gap-3 rounded-lg p-4'>
              <div className='flex items-start gap-2'>
                <Terminal className='mt-0.5 h-4 w-4 text-muted-foreground' />
                <div className='flex flex-col gap-1 text-sm'>
                  <div className='font-medium'>{t('Authentication code')}</div>
                  <div className='font-mono text-2xl font-bold tracking-widest'>
                    {deviceCode}
                  </div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Expires in 15 minutes')}
                  </div>
                </div>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  className='ml-auto'
                  onClick={async () => {
                    await copyToClipboard(deviceCode)
                    toast.success(t('Code copied'))
                  }}
                >
                  <Copy className='h-4 w-4' />
                </Button>
              </div>
              <Button
                type='button'
                variant='default'
                size='sm'
                onClick={() => {
                  try {
                    window.open(deviceURL, '_blank', 'noopener,noreferrer')
                  } catch { /* noop */ }
                  startPolling()
                }}
              >
                {t('Open OpenAI authentication and continue')}
              </Button>
              <div className='text-muted-foreground text-xs'>
                {t('A temporary authentication code has been generated for this integration.')}
                <br />
                1. {t('Open the OpenAI authentication page')}: <span className='font-mono underline'>{deviceURL}</span>
                <br />
                2. {t('Enter the authentication code shown above')}
                <br />
                3. {t('Return here and click the button to continue. Authorization will complete automatically.')}
              </div>
            </div>
          )}

          {/* Polling */}
          {authState === 'device_polling' && (
            <div className='bg-muted/50 flex items-center gap-3 rounded-lg p-4'>
              <Loader2 className='h-5 w-5 animate-spin text-muted-foreground' />
              <div className='text-sm'>
                <div className='font-medium'>{t('Waiting for authorization...')}</div>
                <div className='text-muted-foreground text-xs'>
                  {t('Use code {{code}} on the OpenAI authentication page at {{url}}.', { code: deviceCode, url: deviceURL })}
                </div>
              </div>
            </div>
          )}

          {/* Idle — show command + upload */}
          {(authState === 'idle' || authState === 'error') && (
            <div className='flex flex-col gap-3'>
              {/* Generate device code button */}
              <div className='flex flex-wrap items-center gap-2'>
                <Button
                  type='button'
                  variant='default'
                  size='sm'
                  onClick={handleDeviceStart}
                  disabled={isLoading}
                >
                  {isLoading ? (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  ) : (
                    <Terminal className='mr-2 h-4 w-4' />
                  )}
                  {t('Generate authentication code')}
                </Button>
              </div>

              <Alert>
                <AlertDescription>
                  {t(
                    'The router will generate a temporary authentication code for this integration. Open the OpenAI authentication page, enter the code, and approve access.'
                  )}
                </AlertDescription>
              </Alert>

              <Separator />

              {/* Upload auth.json */}
              <div className='text-sm font-medium'>{t('Already have auth.json?')}</div>
              <div className='flex flex-wrap items-center gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Upload className='mr-2 h-4 w-4' />
                  {t('Upload auth.json')}
                </Button>
                <input
                  ref={fileInputRef}
                  type='file'
                  accept='.json'
                  className='hidden'
                  onChange={handleFileChange}
                />
                <span className='text-muted-foreground text-xs'>{t('or')}</span>
              </div>
              <div className='space-y-1.5'>
                <Label htmlFor='codex-json-paste'>{t('Paste auth.json')}</Label>
                <Textarea
                  id='codex-json-paste'
                  value={jsonPaste}
                  onChange={(e) => setJsonPaste(e.target.value)}
                  placeholder={t('Paste the contents of ~/.codex/auth.json')}
                  rows={4}
                  className='font-mono text-xs'
                />
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => handleJSONUpload(jsonPaste)}
                  disabled={!jsonPaste.trim() || isLoading}
                >
                  {t('Save credential')}
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* ═══════ PKCE CALLBACK PASTE (SECONDARY) ═══════ */}
      {authMethod === 'pkce' && !existingKey && (
        <>
          {authState === 'idle' && (
            <div className='flex flex-col gap-3'>
              <div className='flex flex-wrap items-center gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={async () => {
                    setIsLoading(true)
                    try {
                      const res = await startCodexOAuth()
                      if (!res.success) throw new Error(res.message || 'Failed')
                      const url = res.data?.authorize_url
                      if (url) {
                        try { window.open(url, '_blank', 'noopener,noreferrer') } catch { /* noop */ }
                        toast.success(t('Authorization page opened'))
                      }
                      setAuthState('code_paste')
                    } catch (error) {
                      toast.error(error instanceof Error ? error.message : t('Failed'))
                    } finally {
                      setIsLoading(false)
                    }
                  }}
                  disabled={isLoading}
                >
                  {t('Open authorization page')}
                </Button>
              </div>
              <Alert>
                <AlertDescription>
                  {t(
                    'Continue in your browser. After OpenAI redirects, copy the code from the callback URL and paste it below.'
                  )}
                </AlertDescription>
              </Alert>
            </div>
          )}

          {authState === 'code_paste' && (
            <div className='flex flex-col gap-3'>
              <div className='space-y-1.5'>
                <Label htmlFor='codex-pkce-code'>{t('Authorization code')}</Label>
                <div className='flex gap-2'>
                  <Input
                    id='codex-pkce-code'
                    value={pkceCode}
                    onChange={(e) => setPkceCode(e.target.value)}
                    placeholder={t('Paste the code from the callback URL')}
                    className='font-mono flex-1'
                  />
                  <Button
                    type='button'
                    variant='default'
                    onClick={handlePkceComplete}
                    disabled={!pkceCode.trim() || isLoading}
                  >
                    {isLoading ? (
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    ) : (
                      <CheckCircle2 className='mr-2 h-4 w-4' />
                    )}
                    {t('Complete')}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* Loading state */}
      {authState === 'completing' && (
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Loader2 className='h-4 w-4 animate-spin' />
          {t('Exchanging authorization code...')}
        </div>
      )}

      {/* ── Credential status (after auth) ── */}
      {(authState === 'complete' || existingKey) && credentialInfo && (
        <div className='bg-muted/50 flex flex-col gap-2 rounded-lg p-3'>
          <div className='grid grid-cols-[100px_1fr] gap-x-3 gap-y-1.5 text-sm'>
            {credentialInfo.email && (
              <>
                <div className='text-muted-foreground'>{t('Email')}</div>
                <div className='font-medium'>{credentialInfo.email}</div>
              </>
            )}
            {credentialInfo.accountId && (
              <>
                <div className='text-muted-foreground'>{t('Account')}</div>
                <div className='font-mono text-xs'>{credentialInfo.accountId}</div>
              </>
            )}
            {credentialInfo.accessTokenPreview && (
              <>
                <div className='text-muted-foreground'>{t('Access token')}</div>
                <div className='flex items-center gap-2'>
                  <span className='font-mono text-xs'>
                    {showToken ? credentialInfo.accessTokenPreview : maskToken(credentialInfo.accessTokenPreview)}
                  </span>
                  <button
                    type='button'
                    className='text-muted-foreground hover:text-foreground'
                    onClick={() => setShowToken(!showToken)}
                  >
                    {showToken ? <EyeOff className='h-3.5 w-3.5' /> : <Eye className='h-3.5 w-3.5' />}
                  </button>
                </div>
              </>
            )}
            {credentialInfo.expiresAt && (
              <>
                <div className='text-muted-foreground'>{t('Expires')}</div>
                <div className='font-mono text-xs'>
                  {new Date(credentialInfo.expiresAt).toLocaleString()}
                  {isExpired && (
                    <Badge variant='destructive' className='ml-2'>{t('Expired')}</Badge>
                  )}
                </div>
              </>
            )}
          </div>
          <div className='text-muted-foreground mt-1 text-xs'>
            {t('Save the channel to persist this credential.')}
          </div>
        </div>
      )}
    </div>
  )
}

function Separator() {
  const { t } = useTranslation()

  return (
    <div className='flex items-center gap-2 text-xs text-muted-foreground'>
      <div className='h-px flex-1 bg-border' />
      <span className='shrink-0'>{t('or')}</span>
      <div className='h-px flex-1 bg-border' />
    </div>
  )
}
