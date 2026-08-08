import { request } from './client'
import type { User } from './auth'
import type { Analysis, Plan } from './plans'

export interface AdminUser extends User {
  updated_at: string
}

export interface AdminAnalysis extends Analysis {
  user_id: number
  plan_title: string
}

export interface PageMeta {
  page: number
  page_size: number
  total: number
}

export interface AISettings {
  endpoint: string
  model: string
  has_api_key: boolean
}

export async function listAdminUsers() {
  return (await request<{ data: AdminUser[] }>('/admin/users')).data
}

export async function setUserStatus(id: number, status: 'active' | 'disabled') {
  return (await request<{ data: { status: string } }>(`/admin/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  })).data
}

export async function listAdminPlans() {
  return (await request<{ data: Plan[] }>('/admin/plans')).data
}

export async function listAdminAnalysis(params: { status?: string; page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  if (params.status) query.set('status', params.status)
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('page_size', String(params.pageSize))
  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  return request<{ data: AdminAnalysis[]; meta: PageMeta }>(`/admin/analysis${suffix}`)
}

export async function getAISettings() {
  return (await request<{ data: AISettings }>('/admin/settings/ai')).data
}

export async function saveAISettings(settings: { endpoint: string; model: string; apiKey?: string }) {
  await request('/admin/settings/ai', {
    method: 'PATCH',
    body: JSON.stringify(settings),
  })
}
