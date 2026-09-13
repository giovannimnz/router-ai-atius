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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { PerfModelSummary } from '@/features/performance-metrics/types'

import { buildPerformanceSummary } from './summary'

describe('buildPerformanceSummary', () => {
  test('weights health metrics by request traffic', () => {
    const rows: PerfModelSummary[] = [
      {
        model_name: 'embedding-gte-v1',
        request_count: 47,
        success_rate: 68.09,
        avg_latency_ms: 11330,
        avg_tps: 10,
      },
      {
        model_name: 'gpt-5.6-sol',
        request_count: 20,
        success_rate: 95,
        avg_latency_ms: 55201,
        avg_tps: 40,
      },
      {
        model_name: 'reranker-gte-v1',
        request_count: 14,
        success_rate: 100,
        avg_latency_ms: 61010,
        avg_tps: 20,
      },
      {
        model_name: 'gpt-5.4-mini',
        request_count: 1,
        success_rate: 100,
        avg_latency_ms: 5339,
        avg_tps: 100,
      },
    ]

    const summary = buildPerformanceSummary(rows)

    assert.equal(summary.totalRequests, 82)
    assert.ok(Math.abs(summary.successRate - 80.49) < 0.005)
    assert.equal(summary.avgLatencyMs, 30439)
    assert.ok(Math.abs(summary.avgTps - 20.12) < 0.005)
  })

  test('ignores rows without positive request traffic', () => {
    const summary = buildPerformanceSummary([
      {
        model_name: 'missing-count',
        success_rate: 0,
        avg_latency_ms: 999999,
        avg_tps: 999,
      },
      {
        model_name: 'valid',
        request_count: 2,
        success_rate: 100,
        avg_latency_ms: 500,
        avg_tps: 8,
      },
    ])

    assert.deepEqual(summary, {
      totalRequests: 2,
      successRate: 100,
      avgLatencyMs: 500,
      avgTps: 8,
    })
  })
})
