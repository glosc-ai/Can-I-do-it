<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ActivityIcon,
  CheckCircle2Icon,
  CircleAlertIcon,
  DownloadIcon,
  FileTextIcon,
  PlayIcon,
  XCircleIcon,
} from '@lucide/vue'
import { toast } from '@/lib/message'
import type { Plan } from '@/api/plans'
import WorkspaceLayout from '@/components/layout/WorkspaceLayout.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import PlanStatusBadge from '@/features/plans/components/PlanStatusBadge.vue'
import AnalysisComposer from '@/features/plans/components/AnalysisComposer.vue'
import { usePlansStore } from '@/features/plans/store'
import { errorMessage, fileTypeLabel, formatSize, formatTime } from '@/lib/format'

const store = usePlansStore()
const router = useRouter()
const composerRef = ref<InstanceType<typeof AnalysisComposer> | null>(null)
const analyzingId = ref<number | null>(null)

async function analyze(plan: Plan) {
  if (analyzingId.value !== null) return
  analyzingId.value = plan.id
  try {
    await store.analyze(plan.id)
    toast.success('已提交分析', { description: `「${plan.title}」正在排队分析。` })
  } catch (error) {
    toast.error('提交分析失败', { description: errorMessage(error) })
  } finally {
    analyzingId.value = null
  }
}

function openDetail(plan: Plan) {
  router.push(`/plans/${plan.id}`)
}

interface StatCard {
  key: 'total' | 'active' | 'completed' | 'failed'
  label: string
  icon: typeof ActivityIcon
  colorClass: string
  iconClass: string
}

const statCards: StatCard[] = [
  {
    key: 'total',
    label: '全部分析',
    icon: FileTextIcon,
    colorClass: 'text-foreground',
    iconClass: 'text-muted-foreground',
  },
  {
    key: 'active',
    label: '分析中',
    icon: ActivityIcon,
    colorClass: 'text-amber-600/90 dark:text-amber-400/90',
    iconClass: 'text-amber-500/70 dark:text-amber-400/70',
  },
  {
    key: 'completed',
    label: '已完成',
    icon: CheckCircle2Icon,
    colorClass: 'text-emerald-600/90 dark:text-emerald-400/90',
    iconClass: 'text-emerald-500/70 dark:text-emerald-400/70',
  },
  {
    key: 'failed',
    label: '失败',
    icon: XCircleIcon,
    colorClass: 'text-rose-600/90 dark:text-rose-400/90',
    iconClass: 'text-rose-500/70 dark:text-rose-400/70',
  },
]

onMounted(store.fetch)
onUnmounted(store.stop)
</script>

<template>
  <WorkspaceLayout>
    <div class="flex flex-col gap-10 px-4 py-8 sm:px-6 sm:py-12">

      <!-- 标题区 -->
      <div class="mx-auto flex max-w-2xl flex-col items-center gap-2 text-center animate-fade-up">
        <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">今天想验证什么？</h1>
        <p class="text-sm leading-6 text-muted-foreground sm:text-base">
          输入一个想法，或上传已有计划书，AI 会完成市场调查与商业可行性分析。
        </p>
      </div>

      <!-- 分析输入框 -->
      <div class="animate-fade-up delay-100">
        <AnalysisComposer ref="composerRef" />
      </div>

      <!-- 统计卡片 -->
      <div class="grid grid-cols-2 gap-px overflow-hidden rounded-xl border bg-border md:grid-cols-4 animate-fade-up delay-200">
        <div
          v-for="card in statCards"
          :key="card.key"
          class="flex items-center justify-between gap-3 bg-card p-4 transition-colors duration-150 hover:bg-muted/30"
        >
          <div class="flex flex-col gap-0.5">
            <span class="text-xs text-muted-foreground">{{ card.label }}</span>
            <span class="text-2xl font-semibold tabular-nums" :class="card.colorClass">
              <Skeleton v-if="!store.loaded && store.loading" class="inline-block h-7 w-10" />
              <template v-else>{{ store.stats[card.key] }}</template>
            </span>
          </div>
          <component :is="card.icon" class="size-4 shrink-0" :class="card.iconClass" />
        </div>
      </div>

      <!-- 分析历史 -->
      <section class="flex flex-col gap-3 animate-fade-up delay-300" aria-labelledby="plans-list-title">
        <h2 id="plans-list-title" class="text-lg font-semibold tracking-tight">
          分析历史
        </h2>

        <Alert v-if="store.error && !store.loading" variant="destructive">
          <CircleAlertIcon />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{{ store.error }}</AlertDescription>
          <AlertAction>
            <Button variant="outline" size="sm" @click="store.fetch">重试</Button>
          </AlertAction>
        </Alert>

        <template v-else>
          <div
            v-if="!store.loaded || store.items.length > 0"
            class="overflow-hidden rounded-xl border"
          >
            <Table>
              <TableHeader>
                <TableRow class="hover:bg-transparent">
                  <TableHead class="text-xs font-medium text-muted-foreground">标题</TableHead>
                  <TableHead class="text-xs font-medium text-muted-foreground">类型</TableHead>
                  <TableHead class="text-xs font-medium text-muted-foreground">大小</TableHead>
                  <TableHead class="text-xs font-medium text-muted-foreground">版本</TableHead>
                  <TableHead class="text-xs font-medium text-muted-foreground">状态</TableHead>
                  <TableHead class="text-xs font-medium text-muted-foreground">更新时间</TableHead>
                  <TableHead class="text-right text-xs font-medium text-muted-foreground">操作</TableHead>
                </TableRow>
              </TableHeader>
              <!-- 加载骨架 -->
              <TableBody v-if="!store.loaded">
                <TableRow v-for="row in 3" :key="row">
                  <TableCell><Skeleton class="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton class="h-5 w-10" /></TableCell>
                  <TableCell><Skeleton class="h-4 w-12" /></TableCell>
                  <TableCell><Skeleton class="h-5 w-8" /></TableCell>
                  <TableCell><Skeleton class="h-5 w-14" /></TableCell>
                  <TableCell><Skeleton class="h-4 w-28" /></TableCell>
                  <TableCell />
                </TableRow>
              </TableBody>
              <!-- 数据行 -->
              <TableBody v-else>
                <TableRow
                  v-for="plan in store.items"
                  :key="plan.id"
                  class="cursor-pointer transition-colors duration-150 hover:bg-muted/40"
                  @click="openDetail(plan)"
                >
                  <TableCell>
                    <div class="flex flex-col gap-0.5">
                      <span class="max-w-40 truncate font-medium sm:max-w-64">{{ plan.title }}</span>
                      <span class="max-w-40 truncate text-xs text-muted-foreground sm:max-w-64">{{ plan.filename }}</span>
                    </div>
                  </TableCell>
                  <TableCell class="text-sm text-muted-foreground">
                    {{ plan.filename === 'idea-analysis.txt' ? '想法' : fileTypeLabel(plan.filename, plan.mime_type) }}
                  </TableCell>
                  <TableCell class="text-sm text-muted-foreground">{{ formatSize(plan.size_bytes) }}</TableCell>
                  <TableCell class="text-sm text-muted-foreground">v{{ plan.version }}</TableCell>
                  <TableCell>
                    <PlanStatusBadge :status="plan.status" />
                  </TableCell>
                  <TableCell class="text-sm text-muted-foreground">
                    {{ formatTime(plan.updated_at || plan.created_at) }}
                  </TableCell>
                  <TableCell class="text-right">
                    <Button
                      v-if="plan.download_url"
                      as="a"
                      :href="plan.download_url"
                      target="_blank"
                      rel="noopener noreferrer"
                      variant="ghost"
                      size="sm"
                      class="transition-colors duration-150 hover:text-primary"
                      @click.stop
                    >
                      <DownloadIcon class="size-4" />
                      下载
                    </Button>
                    <Button
                      v-if="plan.status === 'uploaded'"
                      variant="outline"
                      size="sm"
                      :disabled="analyzingId !== null"
                      class="transition-all duration-150 hover:border-primary/40 hover:text-primary"
                      @click.stop="analyze(plan)"
                    >
                      <PlayIcon class="size-4" />
                      开始分析
                    </Button>
                    <Button
                      v-else
                      variant="ghost"
                      size="sm"
                      class="transition-colors duration-150 hover:text-primary"
                      @click.stop="openDetail(plan)"
                    >
                      查看详情
                    </Button>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          <!-- 空状态 -->
          <Empty v-else class="rounded-xl border">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <FileTextIcon />
              </EmptyMedia>
              <EmptyTitle>还没有分析记录</EmptyTitle>
              <EmptyDescription>
                输入一个想法或上传计划书，获取第一份完整的可行性分析。
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button
                variant="outline"
                size="sm"
                class="transition-all duration-150 hover:border-primary/40 hover:text-primary"
                @click="composerRef?.focus()"
              >
                开始第一次分析
              </Button>
            </EmptyContent>
          </Empty>
        </template>
      </section>
    </div>
  </WorkspaceLayout>
</template>
