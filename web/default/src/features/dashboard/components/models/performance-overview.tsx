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
import { useQuery } from '@tanstack/react-query'
import { Gauge, HeartPulse, Timer } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { getPerfMetricsSummary } from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
  getSuccessRateDotClass,
  getSuccessRateTextClass,
} from '@/features/performance-metrics/lib/format'
import { buildPerformanceSummary } from '@/features/performance-metrics/lib/summary'
import type { PerfModelSummary } from '@/features/performance-metrics/types'
import { cn } from '@/lib/utils'

const PERFORMANCE_WINDOW_HOURS = 24
const TOP_MODEL_LIMIT = 6

export function PerformanceOverview() {
  const { t } = useTranslation()
  const metricsQuery = useQuery({
    queryKey: ['perf-metrics-summary', PERFORMANCE_WINDOW_HOURS],
    queryFn: () => getPerfMetricsSummary(PERFORMANCE_WINDOW_HOURS),
    staleTime: 60 * 1000,
    retry: false,
  })

  const models = useMemo(
    () => metricsQuery.data?.data.models ?? [],
    [metricsQuery.data]
  )
  const summary = useMemo(() => buildPerformanceSummary(models), [models])
  const topModels = useMemo(() => models.slice(0, TOP_MODEL_LIMIT), [models])
  const loading = metricsQuery.isLoading
  const hasData = models.length > 0

  if (!loading && !hasData) {
    return (
      <div className='text-muted-foreground overflow-hidden rounded-lg border px-4 py-3 text-center text-xs'>
        {t('No performance data available')}
      </div>
    )
  }

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex flex-wrap items-center gap-x-5 gap-y-2.5 px-4 py-2.5 sm:px-5 sm:py-3'>
        {/* Title */}
        <div className='flex items-center gap-1.5'>
          <HeartPulse
            className='text-muted-foreground/60 size-3.5 shrink-0'
            aria-hidden='true'
          />
          <span className='text-xs font-semibold whitespace-nowrap'>
            {t('Performance health')}
          </span>
        </div>

        {/* Separator */}
        <div className='bg-border hidden h-4 w-px sm:block' />

        {/* 3 KPI inline metrics */}
        {loading ? (
          <div className='flex flex-wrap items-center gap-x-5 gap-y-2'>
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className='flex items-center gap-1.5'>
                <Skeleton className='h-3 w-14' />
                <Skeleton className='h-4 w-16' />
              </div>
            ))}
          </div>
        ) : (
          <div className='flex flex-wrap items-center gap-x-5 gap-y-2'>
            <InlineMetric
              icon={HeartPulse}
              label={t('Success rate')}
              value={formatUptimePct(summary.successRate)}
              valueClassName={getSuccessRateTextClass(summary.successRate)}
            />
            <InlineMetric
              icon={Timer}
              label={t('Average latency')}
              value={formatLatency(summary.avgLatencyMs)}
            />
            <InlineMetric
              icon={Gauge}
              label={t('Throughput')}
              value={formatThroughput(summary.avgTps)}
            />
          </div>
        )}

        {/* Separator */}
        <div className='bg-border hidden h-4 w-px lg:block' />

        {/* Top models inline badges */}
        {!loading && hasData && (
          <div className='flex flex-wrap items-center gap-1.5'>
            {topModels.map((model) => (
              <ModelBadge key={model.model_name} model={model} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function InlineMetric(props: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  valueClassName?: string
}) {
  const Icon = props.icon

  return (
    <div className='flex items-center gap-1.5'>
      <Icon
        className='text-muted-foreground/50 size-3 shrink-0'
        aria-hidden='true'
      />
      <span className='text-muted-foreground text-[11px]'>{props.label}</span>
      <span
        className={cn(
          'font-mono text-xs font-semibold tabular-nums',
          props.valueClassName
        )}
      >
        {props.value}
      </span>
    </div>
  )
}

function ModelBadge(props: { model: PerfModelSummary }) {
  const model = props.model

  return (
    <span className='bg-muted/50 inline-flex items-center gap-1.5 rounded-full px-2.5 py-1'>
      <span className='max-w-[10rem] truncate font-mono text-[11px]'>
        {model.model_name}
      </span>
      <span
        className={cn(
          'size-1.5 rounded-full',
          getSuccessRateDotClass(model.success_rate)
        )}
        aria-hidden='true'
      />
      <span
        className={cn(
          'font-mono text-[11px] font-semibold tabular-nums',
          getSuccessRateTextClass(model.success_rate)
        )}
      >
        {formatUptimePct(model.success_rate)}
      </span>
    </span>
  )
}
