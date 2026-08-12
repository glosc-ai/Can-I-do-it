<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeftIcon,
  CircleAlertIcon,
  DownloadIcon,
  PlayIcon,
  RotateCcwIcon,
} from '@lucide/vue'
import { toast } from '@/lib/message'
import {
  analyzePlan,
  getAnalysis,
  getPlan,
  retryAnalysis,
  type Analysis,
  type Plan,
} from '@/api/plans'
import { APIError } from '@/api/client'
import WorkspaceLayout from '@/components/layout/WorkspaceLayout.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import AnalysisResult from '@/features/plans/components/AnalysisResult.vue'
import AnalysisTimeline from '@/features/plans/components/AnalysisTimeline.vue'
import PlanStatusBadge from '@/features/plans/components/PlanStatusBadge.vue'
import { isAnalysisActive } from '@/features/plans/status'
import { errorMessage, fileTypeLabel, formatSize, formatTime } from '@/lib/format'

const POLL_INTERVAL_MS = 2500

const route = useRoute()
const router = useRouter()
const planId = Number(route.params.id)

const plan = ref<Plan | null>(null)
const analysis = ref<Analysis | null>(null)
const loading = ref(true)
const notFound = ref(!Number.isInteger(planId) || planId <= 0)
const loadError = ref('')
const actionPending = ref(false)

let pollTimer: ReturnType<typeof setTimeout> | null = null

const canAnalyze = computed(
  () => plan.value && plan.value.status === 'uploaded' && !analysis.value,
)
const canRetry = computed(() => analysis.value?.status === 'failed')
const canReanalyze = computed(() => analysis.value?.status === 'succeeded')
const isIdea = computed(() => plan.value?.filename === 'idea-analysis.txt')

function clearPoll() {
  if (pollTimer !== null) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function schedulePoll() {
  clearPoll()
  if (analysis.value && isAnalysisActive(analysis.value.status)) {
    pollTimer = setTimeout(refresh, POLL_INTERVAL_MS)
  }
}

async function refresh() {
  try {
    const [nextPlan, nextAnalysis] = await Promise.all([getPlan(planId), getAnalysis(planId)])
    plan.value = nextPlan
    analysis.value = nextAnalysis
  } catch (error) {
    if (error instanceof APIError && error.status === 404) {
      notFound.value = true
      clearPoll()
      return
    }
    loadError.value = errorMessage(error)
  } finally {
    loading.value = false
    schedulePoll()
  }
}

async function startAnalysis() {
  if (actionPending.value) return
  actionPending.value = true
  try {
    await analyzePlan(planId)
    toast.success('已提交分析', { description: '分析任务已排队，页面会自动刷新进度。' })
    await refresh()
  } catch (error) {
    toast.error('提交分析失败', { description: errorMessage(error) })
  } finally {
    actionPending.value = false
  }
}

async function retry() {
  if (actionPending.value) return
  actionPending.value = true
  try {
    await retryAnalysis(planId)
    toast.success('已重新排队', { description: '分析任务已重新入队。' })
    await refresh()
  } catch (error) {
    toast.error('重试失败', { description: errorMessage(error) })
  } finally {
    actionPending.value = false
  }
}

onMounted(() => {
  if (notFound.value) {
    loading.value = false
    return
  }
  refresh()
})
onUnmounted(clearPoll)
</script>

<template>
  <WorkspaceLayout>
    <div class="flex flex-col gap-6 px-4 py-8 sm:px-6">
      <div>
        <Button variant="ghost" size="sm" @click="router.push('/plans')">
          <ArrowLeftIcon data-icon="inline-start" />
          返回列表
        </Button>
      </div>

      <div v-if="loading" class="flex flex-col gap-6">
        <Skeleton class="h-8 w-64" />
        <div class="grid gap-4 lg:grid-cols-2">
          <Skeleton class="h-48 w-full" />
          <Skeleton class="h-48 w-full" />
        </div>
      </div>

      <Alert v-else-if="notFound" variant="destructive">
        <CircleAlertIcon />
        <AlertTitle>计划书不存在</AlertTitle>
        <AlertDescription>该计划书可能已被删除，或不属于当前账号。</AlertDescription>
      </Alert>

      <Alert v-else-if="loadError && !plan" variant="destructive">
        <CircleAlertIcon />
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{{ loadError }}</AlertDescription>
      </Alert>

      <template v-else-if="plan">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="flex min-w-0 flex-col gap-1">
            <div class="flex items-center gap-3">
              <h1 class="truncate text-2xl font-semibold tracking-tight">{{ plan.title }}</h1>
              <PlanStatusBadge :status="plan.status" />
            </div>
            <p class="truncate text-sm text-muted-foreground">{{ plan.filename }}</p>
          </div>
          <div class="flex gap-2">
            <Button v-if="plan.download_url" as="a" :href="plan.download_url" target="_blank" rel="noopener noreferrer" variant="outline">
              <DownloadIcon data-icon="inline-start" />
              下载文件
            </Button>
            <Button v-if="canAnalyze" :disabled="actionPending" @click="startAnalysis">
              <PlayIcon data-icon="inline-start" />
              开始分析
            </Button>
            <Button v-if="canRetry" variant="outline" :disabled="actionPending" @click="retry">
              <RotateCcwIcon data-icon="inline-start" />
              重试分析
            </Button>
            <Button v-if="canReanalyze" variant="outline" :disabled="actionPending" @click="startAnalysis">
              <RotateCcwIcon data-icon="inline-start" />
              重新分析
            </Button>
          </div>
        </div>

        <div class="grid items-start gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>{{ isIdea ? '想法信息' : '文件信息' }}</CardTitle>
              <CardDescription>{{ isIdea ? '本次想法分析的来源信息。' : '计划书的基本信息与版本。' }}</CardDescription>
            </CardHeader>
            <CardContent>
              <dl class="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                <dt class="text-muted-foreground">类型</dt>
                <dd><Badge variant="secondary">{{ isIdea ? '文字想法' : fileTypeLabel(plan.filename, plan.mime_type) }}</Badge></dd>
                <dt class="text-muted-foreground">{{ isIdea ? '内容大小' : '大小' }}</dt>
                <dd>{{ formatSize(plan.size_bytes) }}</dd>
                <dt class="text-muted-foreground">版本</dt>
                <dd><Badge variant="outline">v{{ plan.version }}</Badge></dd>
                <dt class="text-muted-foreground">上传时间</dt>
                <dd>{{ formatTime(plan.created_at) }}</dd>
                <dt class="text-muted-foreground">更新时间</dt>
                <dd>{{ formatTime(plan.updated_at || plan.created_at) }}</dd>
              </dl>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>分析进度</CardTitle>
              <CardDescription>分析任务会在后台异步执行。</CardDescription>
            </CardHeader>
            <CardContent>
              <div v-if="!analysis" class="flex flex-col items-start gap-3">
                <p class="text-sm text-muted-foreground">
                  还没有分析任务。提交后通常几分钟内出结果。
                </p>
                <Button v-if="canAnalyze" variant="outline" size="sm" :disabled="actionPending" @click="startAnalysis">
                  <PlayIcon data-icon="inline-start" />
                  开始分析
                </Button>
              </div>
              <AnalysisTimeline
                v-else
                :status="analysis.status"
                :created-at="analysis.created_at"
                :updated-at="analysis.updated_at"
              />
            </CardContent>
          </Card>
        </div>

        <template v-if="analysis">
          <Alert v-if="analysis.status === 'failed'" variant="destructive">
            <CircleAlertIcon />
            <AlertTitle>分析失败</AlertTitle>
            <AlertDescription>
              {{ analysis.error || '分析过程中出现未知错误，请重试。' }}
            </AlertDescription>
          </Alert>

          <Card v-if="analysis.status === 'succeeded'">
            <CardHeader>
              <CardTitle>分析结果</CardTitle>
              <CardDescription v-if="analysis.summary">{{ analysis.summary }}</CardDescription>
            </CardHeader>
            <CardContent>
              <AnalysisResult :result="analysis.result || analysis" />
            </CardContent>
          </Card>
        </template>
      </template>
    </div>
  </WorkspaceLayout>
</template>
