<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeftIcon,
  CircleAlertIcon,
  ClockIcon,
  DownloadIcon,
  FileTextIcon,
  Globe2Icon,
  HashIcon,
  LockIcon,
  PackageIcon,
  PlayIcon,
  RotateCcwIcon,
} from '@lucide/vue'
import { toast } from '@/lib/message'
import {
  analyzePlan,
  getAnalysis,
  getPlan,
  retryAnalysis,
  setPlanVisibility,
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

const visibilityPending = ref(false)

async function toggleVisibility() {
  if (!plan.value || visibilityPending.value) return
  const next = plan.value.visibility === 'public' ? 'private' : 'public'
  visibilityPending.value = true
  try {
    await setPlanVisibility(plan.value.id, next)
    plan.value.visibility = next
    toast.success(next === 'public' ? '已公开到项目广场' : '已设为私有', {
      description: next === 'public' ? '其他人现在可以在项目广场看到完整报告。' : '只有你能看到这份报告了。',
    })
  } catch (error) {
    toast.error('切换失败', { description: errorMessage(error) })
  } finally {
    visibilityPending.value = false
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

      <!-- 返回按钮 -->
      <div class="animate-fade-in">
        <Button
          variant="ghost"
          size="sm"
          class="gap-1.5 text-muted-foreground transition-all duration-150 hover:-translate-x-0.5 hover:text-foreground"
          @click="router.push('/plans')"
        >
          <ArrowLeftIcon class="size-4" />
          返回列表
        </Button>
      </div>

      <!-- 加载骨架 -->
      <div v-if="loading" class="flex flex-col gap-6 animate-fade-in">
        <Skeleton class="h-8 w-64" />
        <div class="grid gap-4 lg:grid-cols-2">
          <Skeleton class="h-48 w-full rounded-xl" />
          <Skeleton class="h-48 w-full rounded-xl" />
        </div>
      </div>

      <!-- 404 -->
      <Alert v-else-if="notFound" variant="destructive" class="animate-fade-in rounded-xl">
        <CircleAlertIcon />
        <AlertTitle>计划书不存在</AlertTitle>
        <AlertDescription>该计划书可能已被删除，或不属于当前账号。</AlertDescription>
      </Alert>

      <!-- 加载错误 -->
      <Alert v-else-if="loadError && !plan" variant="destructive" class="animate-fade-in rounded-xl">
        <CircleAlertIcon />
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{{ loadError }}</AlertDescription>
      </Alert>

      <!-- 主内容 -->
      <template v-else-if="plan">

        <!-- 标题区 -->
        <div class="flex flex-wrap items-start justify-between gap-4 animate-fade-up">
          <div class="flex min-w-0 flex-col gap-1.5">
            <div class="flex flex-wrap items-center gap-3">
              <h1 class="truncate text-2xl font-bold tracking-tight">{{ plan.title }}</h1>
              <PlanStatusBadge :status="plan.status" />
            </div>
            <p class="flex items-center gap-1.5 truncate text-sm text-muted-foreground">
              <FileTextIcon class="size-3.5 shrink-0" />
              {{ plan.filename }}
            </p>
          </div>
          <!-- 操作按钮组 -->
          <div class="flex flex-wrap gap-2">
            <Button
              variant="outline"
              :disabled="visibilityPending"
              class="gap-2 transition-all duration-150"
              @click="toggleVisibility"
            >
              <Globe2Icon v-if="plan.visibility === 'public'" class="size-4" />
              <LockIcon v-else class="size-4" />
              {{ plan.visibility === 'public' ? '已公开，点击设为私有' : '设为公开' }}
            </Button>
            <Button
              v-if="plan.download_url"
              as="a"
              :href="plan.download_url"
              target="_blank"
              rel="noopener noreferrer"
              variant="outline"
              class="gap-2 transition-all duration-150 hover:border-primary/40 hover:text-primary"
            >
              <DownloadIcon class="size-4" />
              下载文件
            </Button>
            <Button
              v-if="canAnalyze"
              :disabled="actionPending"
              class="gap-2 shadow-sm shadow-primary/20 transition-all duration-150 hover:shadow-primary/30"
              @click="startAnalysis"
            >
              <PlayIcon class="size-4" />
              开始分析
            </Button>
            <Button
              v-if="canRetry"
              variant="outline"
              :disabled="actionPending"
              class="gap-2 transition-all duration-150 hover:border-amber-500/40 hover:text-amber-600 dark:hover:text-amber-400"
              @click="retry"
            >
              <RotateCcwIcon class="size-4" />
              重试分析
            </Button>
            <Button
              v-if="canReanalyze"
              variant="outline"
              :disabled="actionPending"
              class="gap-2 transition-all duration-150 hover:border-primary/40 hover:text-primary"
              @click="startAnalysis"
            >
              <RotateCcwIcon class="size-4" />
              重新分析
            </Button>
          </div>
        </div>

        <!-- 信息卡片双列 -->
        <div class="grid items-start gap-4 lg:grid-cols-2 animate-fade-up delay-100">

          <!-- 文件/想法信息 -->
          <Card class="rounded-xl shadow-sm">
            <CardHeader class="border-b bg-muted/20 pb-4">
              <CardTitle class="text-base">{{ isIdea ? '想法信息' : '文件信息' }}</CardTitle>
              <CardDescription>{{ isIdea ? '本次想法分析的来源信息。' : '计划书的基本信息与版本。' }}</CardDescription>
            </CardHeader>
            <CardContent class="pt-4">
              <dl class="flex flex-col gap-0">
                <!-- 类型 -->
                <div class="flex items-center justify-between gap-4 py-2.5">
                  <dt class="flex items-center gap-2 text-sm text-muted-foreground">
                    <PackageIcon class="size-3.5" />
                    类型
                  </dt>
                  <dd>
                    <Badge variant="secondary" class="text-xs">
                      {{ isIdea ? '文字想法' : fileTypeLabel(plan.filename, plan.mime_type) }}
                    </Badge>
                  </dd>
                </div>
                <div class="h-px bg-border/50" />
                <!-- 大小 -->
                <div class="flex items-center justify-between gap-4 py-2.5">
                  <dt class="flex items-center gap-2 text-sm text-muted-foreground">
                    <FileTextIcon class="size-3.5" />
                    {{ isIdea ? '内容大小' : '大小' }}
                  </dt>
                  <dd class="text-sm font-medium">{{ formatSize(plan.size_bytes) }}</dd>
                </div>
                <div class="h-px bg-border/50" />
                <!-- 版本 -->
                <div class="flex items-center justify-between gap-4 py-2.5">
                  <dt class="flex items-center gap-2 text-sm text-muted-foreground">
                    <HashIcon class="size-3.5" />
                    版本
                  </dt>
                  <dd>
                    <Badge variant="outline" class="font-mono text-xs">v{{ plan.version }}</Badge>
                  </dd>
                </div>
                <div class="h-px bg-border/50" />
                <!-- 上传时间 -->
                <div class="flex items-center justify-between gap-4 py-2.5">
                  <dt class="flex items-center gap-2 text-sm text-muted-foreground">
                    <ClockIcon class="size-3.5" />
                    上传时间
                  </dt>
                  <dd class="text-sm text-muted-foreground">{{ formatTime(plan.created_at) }}</dd>
                </div>
                <div class="h-px bg-border/50" />
                <!-- 更新时间 -->
                <div class="flex items-center justify-between gap-4 py-2.5">
                  <dt class="flex items-center gap-2 text-sm text-muted-foreground">
                    <ClockIcon class="size-3.5" />
                    更新时间
                  </dt>
                  <dd class="text-sm text-muted-foreground">{{ formatTime(plan.updated_at || plan.created_at) }}</dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          <!-- 分析进度 -->
          <Card class="rounded-xl shadow-sm">
            <CardHeader class="border-b bg-muted/20 pb-4">
              <CardTitle class="text-base">分析进度</CardTitle>
              <CardDescription>分析任务会在后台异步执行。</CardDescription>
            </CardHeader>
            <CardContent class="pt-5">
              <div v-if="!analysis" class="flex flex-col items-start gap-4">
                <p class="text-sm leading-6 text-muted-foreground">
                  还没有分析任务。提交后通常几分钟内出结果。
                </p>
                <Button
                  v-if="canAnalyze"
                  variant="outline"
                  size="sm"
                  :disabled="actionPending"
                  class="gap-2 transition-all duration-150 hover:border-primary/40 hover:text-primary"
                  @click="startAnalysis"
                >
                  <PlayIcon class="size-4" />
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

        <!-- 分析失败提示 -->
        <Alert
          v-if="analysis?.status === 'failed'"
          variant="destructive"
          class="animate-fade-in rounded-xl"
        >
          <CircleAlertIcon />
          <AlertTitle>分析失败</AlertTitle>
          <AlertDescription>
            {{ analysis.error || '分析过程中出现未知错误，请重试。' }}
          </AlertDescription>
        </Alert>

        <!-- 分析结果 -->
        <Card
          v-if="analysis?.status === 'succeeded'"
          class="rounded-xl shadow-sm animate-fade-up delay-200"
        >
          <CardHeader class="border-b bg-muted/20 pb-4">
            <CardTitle class="text-base">分析结果</CardTitle>
            <CardDescription v-if="analysis.summary">{{ analysis.summary }}</CardDescription>
          </CardHeader>
          <CardContent class="pt-6">
            <AnalysisResult :result="analysis.result || analysis" />
          </CardContent>
        </Card>

      </template>
    </div>
  </WorkspaceLayout>
</template>
