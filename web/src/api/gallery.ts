import { request } from './client'
import type { Analysis, Plan } from './plans'

export interface GalleryPlan {
  id: number
  title: string
  filename: string
  mime_type: string
  overall_score?: number
  verdict?: string
  author_name: string
  author_avatar?: string
  created_at: string
}

export interface GalleryPlanDetail {
  plan: Plan
  author_name: string
  author_avatar?: string
  analysis: Analysis | null
}

export interface SimilarPlan {
  id: number
  title: string
  overall_score?: number
  verdict?: string
  created_at: string
}

export async function listGalleryPlans(page = 1, pageSize = 20) {
  const query = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
  return (await request<{ data: GalleryPlan[] }>(`/gallery/plans?${query.toString()}`)).data
}

export async function getGalleryPlan(id: number) {
  return (await request<{ data: GalleryPlanDetail }>(`/gallery/plans/${id}`)).data
}

export async function searchSimilarPlans(q: string) {
  const query = new URLSearchParams({ q })
  return (await request<{ data: SimilarPlan[] }>(`/gallery/similar?${query.toString()}`)).data
}
