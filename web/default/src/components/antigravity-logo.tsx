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

export function AntigravityLogo(props: { size?: number; className?: string }) {
  const size = props.size ?? 20

  return (
    <svg
      viewBox='0 0 24 24'
      className={cn('shrink-0 rounded-sm object-contain', props.className)}
      style={{ width: size, height: size }}
      fill='none'
      xmlns='http://www.w3.org/2000/svg'
      aria-hidden='true'
    >
      <defs>
        <linearGradient id='ag-grad-primary' x1='2' y1='2' x2='22' y2='22' gradientUnits='userSpaceOnUse'>
          <stop offset='0%' stopColor='#4285F4' />
          <stop offset='40%' stopColor='#9B72CB' />
          <stop offset='80%' stopColor='#D96570' />
          <stop offset='100%' stopColor='#F4B400' />
        </linearGradient>
      </defs>
      {/* Central Antigravity Star / Inverted Gravity Prism */}
      <path
        d='M12 2C12 7.52285 7.52285 12 2 12C7.52285 12 12 16.4772 12 22C12 16.4772 16.4772 12 22 12C16.4772 12 12 7.52285 12 2Z'
        fill='url(#ag-grad-primary)'
      />
      {/* Antigravity Orbit Ring */}
      <circle
        cx='12'
        cy='12'
        r='9.5'
        stroke='url(#ag-grad-primary)'
        strokeWidth='1.5'
        strokeDasharray='4 2'
        opacity='0.85'
      />
    </svg>
  )
}
