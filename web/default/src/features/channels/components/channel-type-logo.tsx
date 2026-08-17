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
import { Server } from 'lucide-react'

import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { CHANNEL_TYPE_OPTIONS } from '../constants'
import { CHANNEL_TYPE_ATIUS_LOCAL_EMBEDDINGS, getChannelTypeIcon } from '../lib'

export function ChannelTypeLogo(props: {
  type: number
  size?: number
  className?: string
}) {
  const size = props.size ?? 16

  if (props.type === CHANNEL_TYPE_ATIUS_LOCAL_EMBEDDINGS) {
    return (
      <svg
        viewBox='0 0 512 512'
        className={cn(
          'text-background shrink-0 rounded-sm object-contain',
          props.className
        )}
        style={{ width: size, height: size }}
        aria-hidden='true'
      >
        <rect width='512' height='512' fill='currentColor' />
        <path
          d='M52 450 256 49l204 401H360L256 219 152 450Z'
          fill='#0f3b25'
        />
        <path
          d='M296 300c-17-25-45-40-76-40-49 0-89 40-89 90s40 90 89 90c31 0 59-15 76-40h-52c-7 4-15 6-24 6-31 0-56-25-56-56s25-56 56-56c9 0 17 2 24 6Z'
          fill='#d2aa2a'
        />
        <path d='M190 396h131v7H190Z' fill='#d2aa2a' />
      </svg>
    )
  }

  const isKnownType = CHANNEL_TYPE_OPTIONS.some(
    (option) => option.value === props.type
  )
  if (!isKnownType) {
    return (
      <Server
        className={cn('text-muted-foreground shrink-0', props.className)}
        style={{ width: size, height: size }}
        aria-hidden='true'
      />
    )
  }

  return (
    <span className={cn('inline-flex shrink-0', props.className)}>
      {getLobeIcon(`${getChannelTypeIcon(props.type)}.Color`, size)}
    </span>
  )
}
