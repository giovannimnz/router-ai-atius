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
import { Shield, AlertTriangle, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useDialogs } from '@/hooks/use-dialog'

import { useTwoFA } from '../hooks'
import { TwoFABackupDialog } from './dialogs/two-fa-backup-dialog'
import { TwoFADisableDialog } from './dialogs/two-fa-disable-dialog'
import { TwoFASetupDialog } from './dialogs/two-fa-setup-dialog'

// ============================================================================
// Two-Factor Authentication Card Component
// ============================================================================

interface TwoFACardProps {
  loading: boolean
}

type DialogKey = 'setup' | 'disable' | 'backup'

export function TwoFACard({ loading: pageLoading }: TwoFACardProps) {
  const { t } = useTranslation()
  const { status, loading, error, refetch } = useTwoFA(!pageLoading)
  const dialogs = useDialogs<DialogKey>()

  if (pageLoading || loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <Skeleton className='h-6 w-48' />
          <Skeleton className='mt-2 h-4 w-64' />
        </CardHeader>
        <CardContent className='p-3 sm:p-5'>
          <Skeleton className='h-20 w-full' />
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <CardTitle>
            <h2 className='text-lg tracking-tight sm:text-xl'>
              {t('Two-Factor Authentication')}
            </h2>
          </CardTitle>
          <CardDescription className='text-xs sm:text-sm'>
            {t('Add an extra layer of security to your account')}
          </CardDescription>
        </CardHeader>

        <CardContent className='p-3 sm:p-5'>
          {error && (
            <Alert variant='destructive'>
              <AlertTriangle />
              <AlertTitle>{t('Failed to load')}</AlertTitle>
              <AlertDescription className='flex flex-wrap items-center justify-between gap-3'>
                <span>{t('Failed to load 2FA status')}</span>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => void refetch()}
                >
                  {t('Retry')}
                </Button>
              </AlertDescription>
            </Alert>
          )}
          {!error && (
            <div className='min-w-0 space-y-6'>
              {/* Status Section */}
              <div className='flex min-w-0 flex-col gap-4 sm:flex-row sm:items-start sm:justify-between xl:flex-col 2xl:flex-row'>
                <div className='flex min-w-0 items-start gap-4'>
                  <div className='bg-muted rounded-md p-2'>
                    <Shield className='h-5 w-5' />
                  </div>
                  <div className='min-w-0 space-y-1'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <p className='font-medium'>
                        {t('Two-Step Verification')}
                      </p>
                      {status.enabled ? (
                        <StatusBadge
                          label={t('Enabled')}
                          variant='success'
                          showDot
                          copyable={false}
                        />
                      ) : (
                        <StatusBadge
                          label={t('Disabled')}
                          variant='neutral'
                          showDot
                          copyable={false}
                        />
                      )}
                      {status.locked && (
                        <StatusBadge
                          label={t('Locked')}
                          variant='danger'
                          showDot
                          copyable={false}
                        />
                      )}
                    </div>
                    <p className='text-muted-foreground text-sm'>
                      {status.enabled
                        ? t('Backup codes remaining: {{count}}', {
                            count: status.backup_codes_remaining,
                          })
                        : t('Add an extra layer of security to your account')}
                    </p>
                  </div>
                </div>

                {!status.enabled && (
                  <Button
                    className='w-full whitespace-normal sm:w-auto xl:w-full 2xl:w-auto'
                    onClick={() => dialogs.open('setup')}
                  >
                    {t('Enable')}
                  </Button>
                )}
              </div>

              {/* Actions Section - Only show when enabled */}
              {status.enabled && (
                <div className='flex flex-col gap-3 border-t pt-6 2xl:flex-row'>
                  <Button
                    variant='outline'
                    className='h-auto min-w-0 items-start justify-start px-3 py-2.5 text-left whitespace-normal 2xl:flex-1'
                    onClick={() => dialogs.open('backup')}
                  >
                    <RefreshCw className='mt-0.5 mr-2 h-4 w-4 shrink-0' />
                    {t('Regenerate Backup Codes')}
                  </Button>
                  <Button
                    variant='destructive'
                    className='h-auto min-w-0 items-start justify-start px-3 py-2.5 text-left whitespace-normal 2xl:flex-1'
                    onClick={() => dialogs.open('disable')}
                  >
                    <AlertTriangle className='mt-0.5 mr-2 h-4 w-4 shrink-0' />
                    {t('Disable 2FA')}
                  </Button>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Dialogs */}
      <TwoFASetupDialog
        open={dialogs.isOpen('setup')}
        onOpenChange={(open) =>
          open ? dialogs.open('setup') : dialogs.close('setup')
        }
        onSuccess={refetch}
      />

      <TwoFADisableDialog
        open={dialogs.isOpen('disable')}
        onOpenChange={(open) =>
          open ? dialogs.open('disable') : dialogs.close('disable')
        }
        onSuccess={refetch}
      />

      <TwoFABackupDialog
        open={dialogs.isOpen('backup')}
        onOpenChange={(open) =>
          open ? dialogs.open('backup') : dialogs.close('backup')
        }
        onSuccess={refetch}
      />
    </>
  )
}
