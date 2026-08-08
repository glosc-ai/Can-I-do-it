import type { BadgeVariants } from '@/components/ui/badge'

export interface StatusMeta {
  label: string
  variant: NonNullable<BadgeVariants['variant']>
}

const PLAN_STATUS_META: Record<string, StatusMeta> = {
  uploaded: { label: '待分析', variant: 'secondary' },
  queued: { label: '排队中', variant: 'outline' },
  processing: { label: '分析中', variant: 'outline' },
  completed: { label: '已完成', variant: 'default' },
  failed: { label: '失败', variant: 'destructive' },
}

const ANALYSIS_STATUS_META: Record<string, StatusMeta> = {
  queued: { label: '排队中', variant: 'outline' },
  running: { label: '分析中', variant: 'outline' },
  succeeded: { label: '已完成', variant: 'default' },
  failed: { label: '失败', variant: 'destructive' },
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
