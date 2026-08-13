import { request } from './client'

export interface User {
  id: number
  name: string
  nickname: string
  email: string
  avatar: string
  role: 'owner' | 'user'
  status: string
}

export async function currentUser() {
  const r = await request<{ data: User }>('/auth/me')
  return r.data
}

export async function logout() {
  await request<void>('/auth/logout', { method: 'POST' })
}
