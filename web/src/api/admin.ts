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

export interface StorageSettings {
  enabled: boolean
  endpoint: string
  bucket: string
  public_url: string
  region: string
  force_path_style: boolean
  has_credentials: boolean
  configured: boolean
  using_r2: boolean
}

export interface StorageAsset {
  id: number
  user_id: number
  plan_id?: number
  source: 'upload' | 'ai_generated' | 'fetched'
  name: string
  mime_type: string
  size_bytes: number
  metadata?: Record<string, unknown>
  download_url: string
  created_at: string
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

export async function getStorageSettings() {
  return (await request<{ data: StorageSettings }>('/admin/settings/storage')).data
}

export async function saveStorageSettings(settings: {
  enabled: boolean
  endpoint: string
  bucket: string
  publicUrl?: string
  region?: string
  accessKeyId?: string
  secretAccessKey?: string
  forcePathStyle?: boolean
  clearCredentials?: boolean
}) {
  return (await request<{ data: StorageSettings }>('/admin/settings/storage', {
    method: 'PATCH',
    body: JSON.stringify({
      enabled: settings.enabled,
      endpoint: settings.endpoint,
      bucket: settings.bucket,
      public_url: settings.publicUrl,
      region: settings.region,
      access_key_id: settings.accessKeyId,
      secret_access_key: settings.secretAccessKey,
      force_path_style: settings.forcePathStyle,
      clear_credentials: settings.clearCredentials,
    }),
  })).data
}

export async function testStorageSettings() {
  return (await request<{ data: { status: string } }>('/admin/settings/storage/test', {
    method: 'POST',
  })).data
}

export async function listAdminAssets(params: { source?: string; page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  if (params.source) query.set('source', params.source)
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('page_size', String(params.pageSize))
  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  return request<{ data: StorageAsset[]; meta: PageMeta }>(`/admin/assets${suffix}`)
}

export async function deleteAdminAsset(id: number) {
  await request(`/admin/assets/${id}`, { method: 'DELETE' })
}
