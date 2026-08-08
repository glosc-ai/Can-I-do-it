import { request } from './client'

export interface Plan {
  id: number
  user_id: number
  title: string
  filename: string
  mime_type: string
  size_bytes: number
  version: number
  status: string
  created_at: string
  updated_at: string
}

export type AnalysisStatus = 'queued' | 'running' | 'succeeded' | 'failed'

export interface Analysis {
  id: number
  plan_id: number
  status: AnalysisStatus
  error?: string
  summary?: string
  result?: unknown
  created_at: string
  updated_at: string
}

export async function listPlans() {
  return (await request<{ data: Plan[] }>('/plans')).data
}

export async function getPlan(id: number) {
  return (await request<{ data: Plan }>(`/plans/${id}`)).data
}

export async function uploadPlan(file: File, title: string) {
  const body = new FormData()
  body.append('file', file)
  body.append('title', title)
  return (await request<{ data: Plan }>('/plans', { method: 'POST', body })).data
}

export async function analyzePlan(id: number) {
  return (await request<{ data: { id: number; status: string } }>(`/plans/${id}/analyze`, { method: 'POST' })).data
}

export async function getAnalysis(planId: number) {
  return (await request<{ data: Analysis | null }>(`/plans/${planId}/analysis`)).data
}

export async function retryAnalysis(planId: number) {
  return (await request<{ data: { id: number; status: string } }>(`/plans/${planId}/analysis/retry`, { method: 'POST' })).data
}
