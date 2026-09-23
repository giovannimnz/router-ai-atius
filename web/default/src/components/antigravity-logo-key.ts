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
export const ANTIGRAVITY_ICON_KEY = 'Internal.antigravity'
export const ANTIGRAVITY_COLOR_ICON_KEY = 'Internal.antigravity-color'

export function isAntigravityColorIcon(iconKey: string | null | undefined): boolean {
  if (!iconKey) return false
  const trimmed = iconKey.trim().toLowerCase()
  if (
    trimmed === 'internal.antigravity-color' ||
    trimmed === 'internal.antigravity.color' ||
    trimmed === 'internal.antigravity_color' ||
    trimmed === 'antigravity-color' ||
    trimmed.startsWith('antigravity.color')
  ) {
    return true
  }
  return false
}

export function isAntigravityIcon(iconKey: string | null | undefined): boolean {
  if (!iconKey) return false
  const trimmed = iconKey.trim().toLowerCase()
  return trimmed.startsWith('internal.antigravity') || trimmed.startsWith('antigravity')
}

export function getInternalIconIdentifier(iconKey: string | null | undefined): string | null {
  if (!iconKey) return null
  const trimmed = iconKey.trim()
  if (/^internal\./i.test(trimmed)) {
    return trimmed.slice(9).trim().toLowerCase()
  }
  return null
}

