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
      className={cn('shrink-0 rounded-[5.5px] object-contain', props.className)}
      style={{ width: size, height: size, flex: 'none', lineHeight: 1 }}
      xmlns='http://www.w3.org/2000/svg'
      fill='none'
      aria-hidden='true'
    >
      <title>Antigravity</title>
      <rect width='24' height='24' rx='5.5' fill='#000000' />
      <rect
        width='23.2'
        height='23.2'
        x='0.4'
        y='0.4'
        rx='5.1'
        stroke='rgba(255,255,255,0.12)'
        strokeWidth='0.4'
      />
      <path
        fill='#FFFFFF'
        fillRule='evenodd'
        d='M21.751 22.607c1.34 1.005 3.35.335 1.508-1.508C17.73 15.74 18.904 1 12.037 1 5.17 1 6.342 15.74.815 21.1c-2.01 2.009.167 2.511 1.507 1.506 5.192-3.517 4.857-9.714 9.715-9.714 4.857 0 4.522 6.197 9.714 9.715z'
        transform='translate(12, 12) scale(0.68) translate(-12, -11.8)'
      />
    </svg>
  )
}
