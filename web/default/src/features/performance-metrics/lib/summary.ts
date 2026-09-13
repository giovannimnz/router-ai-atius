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
import type { PerfModelSummary } from '@/features/performance-metrics/types'

type WeightedMetric = 'avg_latency_ms' | 'avg_tps' | 'success_rate'

export type PerformanceSummary = {
  totalRequests: number
  avgLatencyMs: number
  avgTps: number
  successRate: number
}

function requestWeight(row: PerfModelSummary): number {
  const value = Number(row.request_count)
  return Number.isFinite(value) && value > 0 ? value : 0
}

function weightedAverage(
  rows: PerfModelSummary[],
  metric: WeightedMetric,
  isValid: (value: number) => boolean
): number {
  let weightedTotal = 0
  let totalWeight = 0

  for (const row of rows) {
    const value = Number(row[metric])
    const weight = requestWeight(row)
    if (!isValid(value) || weight === 0) continue
    weightedTotal += value * weight
    totalWeight += weight
  }

  return totalWeight > 0 ? weightedTotal / totalWeight : NaN
}

export function buildPerformanceSummary(
  rows: PerfModelSummary[]
): PerformanceSummary {
  let totalRequests = 0
  for (const row of rows) totalRequests += requestWeight(row)

  return {
    totalRequests,
    avgLatencyMs: Math.round(
      weightedAverage(
        rows,
        'avg_latency_ms',
        (value) => Number.isFinite(value) && value > 0
      )
    ),
    avgTps: weightedAverage(
      rows,
      'avg_tps',
      (value) => Number.isFinite(value) && value > 0
    ),
    successRate: weightedAverage(rows, 'success_rate', Number.isFinite),
  }
}
