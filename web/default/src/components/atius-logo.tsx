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
import { cn } from '@/lib/utils'

export function AtiusLogo(props: { size?: number; className?: string }) {
  const size = props.size ?? 16

  return (
    <svg
      xmlns='http://www.w3.org/2000/svg'
      viewBox='0 0 512 512'
      className={cn('shrink-0 rounded-[22%] object-contain', props.className)}
      style={{ width: size, height: size, flex: 'none', lineHeight: 1 }}
      fill='none'
      role='img'
      aria-labelledby='atius-logo-title atius-logo-desc'
    >
      <title id='atius-logo-title'>Atius</title>
      <desc id='atius-logo-desc'>White letter A with a light gray letter C on adaptive dark background.</desc>
      <rect width='512' height='512' rx='112' fill='#000000' />
      <rect
        width='496'
        height='496'
        x='8'
        y='8'
        rx='104'
        stroke='rgba(255,255,255,0.12)'
        strokeWidth='8'
      />
      <path d='M52 450 256 49l204 401H360L256 219 152 450Z' fill='#FFFFFF' />
      <path
        d='M296 300c-17-25-45-40-76-40-49 0-89 40-89 90s40 90 89 90c31 0 59-15 76-40h-52c-7 4-15 6-24 6-31 0-56-25-56-56s25-56 56-56c9 0 17 2 24 6Z'
        fill='none'
        stroke='#000000'
        strokeWidth='24'
        strokeLinejoin='round'
      />
      <path
        d='M296 300c-17-25-45-40-76-40-49 0-89 40-89 90s40 90 89 90c31 0 59-15 76-40h-52c-7 4-15 6-24 6-31 0-56-25-56-56s25-56 56-56c9 0 17 2 24 6Z'
        fill='#D1D5DB'
      />
      <path d='M190 396h131v7H190Z' fill='#D1D5DB' />
    </svg>
  )
}
