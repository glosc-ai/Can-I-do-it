import { ref } from 'vue'
import { defineStore } from 'pinia'
import { listGalleryPlans, type GalleryPlan } from '@/api/gallery'

const PAGE_SIZE = 20

export const useGalleryStore = defineStore('gallery', () => {
  const items = ref<GalleryPlan[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref('')
  const page = ref(1)
  const hasMore = ref(true)

  async function fetch() {
    loading.value = true
    error.value = ''
    try {
      items.value = await listGalleryPlans(1, PAGE_SIZE)
      page.value = 1
      hasMore.value = items.value.length === PAGE_SIZE
      loaded.value = true
    } catch (e) {
      error.value = e instanceof Error ? e.message : '请求失败，请稍后重试'
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (loading.value || !hasMore.value) return
    loading.value = true
    try {
      const next = await listGalleryPlans(page.value + 1, PAGE_SIZE)
      items.value = [...items.value, ...next]
      page.value += 1
      hasMore.value = next.length === PAGE_SIZE
    } catch (e) {
      error.value = e instanceof Error ? e.message : '请求失败，请稍后重试'
    } finally {
      loading.value = false
    }
  }

  return { items, loading, loaded, error, hasMore, fetch, loadMore }
})
