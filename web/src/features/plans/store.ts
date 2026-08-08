import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { analyzePlan, listPlans, uploadPlan, type Plan } from '@/api/plans'
import { isPlanActive } from './status'

const POLL_INTERVAL_MS = 3000

export const usePlansStore = defineStore('plans', () => {
  const items = ref<Plan[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref('')
  const uploading = ref(false)

  let pollTimer: ReturnType<typeof setTimeout> | null = null

  const stats = computed(() => ({
    total: items.value.length,
    active: items.value.filter(p => isPlanActive(p.status)).length,
    completed: items.value.filter(p => p.status === 'completed').length,
    failed: items.value.filter(p => p.status === 'failed').length,
  }))

  function clearPoll() {
    if (pollTimer !== null) {
      clearTimeout(pollTimer)
      pollTimer = null
    }
  }

  function schedulePoll() {
    clearPoll()
    if (items.value.some(p => isPlanActive(p.status))) {
      pollTimer = setTimeout(fetch, POLL_INTERVAL_MS)
    }
  }

  async function fetch() {
    loading.value = true
    error.value = ''
    try {
      items.value = await listPlans()
      loaded.value = true
    } catch (e) {
      error.value = e instanceof Error ? e.message : '请求失败，请稍后重试'
    } finally {
      loading.value = false
      schedulePoll()
    }
  }

  async function upload(file: File, title: string) {
    if (uploading.value) return null
    uploading.value = true
    try {
      const plan = await uploadPlan(file, title)
      await fetch()
      return plan
    } finally {
      uploading.value = false
    }
  }

  async function analyze(id: number) {
    const result = await analyzePlan(id)
    const plan = items.value.find(p => p.id === id)
    if (plan) plan.status = 'queued'
    schedulePoll()
    return result
  }

  function stop() {
    clearPoll()
  }

  return { items, loading, loaded, error, uploading, stats, fetch, upload, analyze, stop }
})
