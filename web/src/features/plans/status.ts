import type { BadgeVariants } from '@/components/ui/badge'

export interface StatusMeta {
  label: string
  variant: NonNullable<BadgeVariants['variant']>
  /** Extra Tailwind classes applied on top of the badge variant */
  className?: string
}

const PLAN_STATUS_META: Record<string, StatusMeta> = {
  uploaded: {
    label: '待分析',
    variant: 'secondary',
    className: 'text-muted-foreground',
  },
  queued: {
    label: '排队中',
    variant: 'secondary',
    className: 'text-amber-700/80 dark:text-amber-400/80',
  },
  processing: {
    label: '分析中',
    variant: 'secondary',
    className: 'text-violet-700/80 dark:text-violet-400/80',
  },
  completed: {
    label: '已完成',
    variant: 'secondary',
    className: 'text-emerald-700/80 dark:text-emerald-400/80',
  },
  failed: {
    label: '失败',
    variant: 'secondary',
    className: 'text-rose-700/80 dark:text-rose-400/80',
  },
}

const ANALYSIS_STATUS_META: Record<string, StatusMeta> = {
  queued: {
    label: '排队中',
    variant: 'secondary',
    className: 'text-amber-700/80 dark:text-amber-400/80',
  },
  running: {
    label: '分析中',
    variant: 'secondary',
    className: 'text-violet-700/80 dark:text-violet-400/80',
  },
  succeeded: {
    label: '已完成',
    variant: 'secondary',
    className: 'text-emerald-700/80 dark:text-emerald-400/80',
  },
  failed: {
    label: '失败',
    variant: 'secondary',
    className: 'text-rose-700/80 dark:text-rose-400/80',
  },
}

export function planStatusMeta(status: string): StatusMeta {
  return PLAN_STATUS_META[status] ?? { label: status, variant: 'outline' }
}

export function analysisStatusMeta(status: string): StatusMeta {
  return ANALYSIS_STATUS_META[status] ?? { label: status, variant: 'outline' }
}

export function isPlanActive(status: string): boolean {
  return status === 'queued' || status === 'processing'
}

export function isAnalysisActive(status: string): boolean {
  return status === 'queued' || status === 'running'
}
