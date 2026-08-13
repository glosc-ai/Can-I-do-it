<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRightIcon, CircleAlertIcon, CircleCheckIcon, ClipboardCheckIcon } from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'

const props = defineProps<{ result: unknown }>()

interface Dimension {
  key: string
  name: string
  score: number
  weight: number
  confidence?: number
  reasoning?: string
  evidence?: string[]
  gaps?: string[]
}

interface Step {
  step?: string
  title?: string
  status?: string
  summary?: string
  questions?: string[]
}

const record = computed<Record<string, any> | null>(() => {
  const value = props.result
  return value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, any> : null
})

const dimensions = computed<Dimension[]>(() => {
  const value = record.value?.dimensions
  if (!Array.isArray(value)) return []
  return value.map((item: any) => ({
    key: String(item?.key || ''),
    name: String(item?.name || item?.key || '未命名维度'),
    score: clamp(Number(item?.score || 0)),
    weight: Number(item?.weight || 0),
    confidence: clamp(Number(item?.confidence || 0)),
    reasoning: String(item?.reasoning || ''),
    evidence: list(item?.evidence),
    gaps: list(item?.gaps),
  }))
})

const process = computed<Step[]>(() => {
  const value = record.value?.analysis_process || record.value?.process
  return Array.isArray(value) ? value.map((item: any) => ({
    step: String(item?.step || ''),
    title: String(item?.title || item?.step || '分析步骤'),
    status: String(item?.status || 'completed'),
    summary: String(item?.summary || ''),
    questions: list(item?.questions),
  })) : []
})

const score = computed(() => clamp(Number(record.value?.overall_score ?? record.value?.score ?? 0)))
const verdict = computed(() => String(record.value?.verdict || verdictFor(score.value)))
const nextActions = computed(() => list(record.value?.next_actions || record.value?.next_steps || record.value?.recommendations))
const rawText = computed(() => {
  if (typeof props.result === 'string') return props.result
  if (!record.value && props.result) return JSON.stringify(props.result, null, 2)
  return ''
})

function clamp(value: number) { return Number.isFinite(value) ? Math.min(100, Math.max(0, value)) : 0 }
function list(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(item => typeof item === 'string' ? item : JSON.stringify(item)).filter(Boolean)
  if (typeof value === 'string' && value.trim()) return [value.trim()]
  return []
}
function verdictFor(value: number) {
  if (value >= 80) return '可行'
  if (value >= 60) return '有条件可行'
  if (value >= 40) return '风险较高'
  return '当前不可行'
}
function scoreColor(value: number) {
  if (value >= 75) return 'text-emerald-600 dark:text-emerald-400'
  if (value >= 55) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}
function scoreStroke(value: number) {
  if (value >= 75) return 'stroke-emerald-500 dark:stroke-emerald-400'
  if (value >= 55) return 'stroke-amber-500 dark:stroke-amber-400'
  return 'stroke-rose-500 dark:stroke-rose-400'
}
function progressColorClass(value: number) {
  if (value >= 75) return 'bg-emerald-500 dark:bg-emerald-400'
  if (value >= 55) return 'bg-amber-500 dark:bg-amber-400'
  return 'bg-rose-500 dark:bg-rose-400'
}

// Animate progress bars after mount
const mounted = ref(false)
onMounted(() => { setTimeout(() => { mounted.value = true }, 50) })

// SVG ring params
const RING_R = 42
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_R
const ringOffset = computed(() => RING_CIRCUMFERENCE * (1 - score.value / 100))
</script>

<template>
  <div v-if="record" class="flex flex-col gap-8">

    <!-- ─── Score overview ─────────────────────────────── -->
    <div class="grid gap-4 md:grid-cols-[minmax(0,1fr)_220px]">

      <!-- Score + summary -->
      <div class="flex items-start gap-5 rounded-xl border border-border/60 bg-muted/20 p-5">
        <!-- SVG ring -->
        <div class="shrink-0">
          <svg width="96" height="96" viewBox="0 0 96 96" class="-rotate-90" aria-hidden="true">
            <!-- Track -->
            <circle
              cx="48" cy="48"
              :r="RING_R"
              fill="none"
              stroke-width="7"
              class="stroke-muted"
            />
            <!-- Progress -->
            <circle
              cx="48" cy="48"
              :r="RING_R"
              fill="none"
              stroke-width="7"
              stroke-linecap="round"
              :stroke-dasharray="RING_CIRCUMFERENCE"
              :stroke-dashoffset="mounted ? ringOffset : RING_CIRCUMFERENCE"
              :class="scoreStroke(score)"
              style="transition: stroke-dashoffset 1s cubic-bezier(0.4, 0, 0.2, 1)"
            />
          </svg>
          <!-- Score in center (overlaid) -->
          <div class="-mt-[88px] flex h-24 flex-col items-center justify-center">
            <span class="text-xl font-semibold tabular-nums" :class="scoreColor(score)">
              {{ score.toFixed(0) }}
            </span>
            <span class="text-[10px] text-muted-foreground">/ 100</span>
          </div>
        </div>
        <!-- Text -->
        <div class="flex min-w-0 flex-col gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">综合可行性</p>
            <Badge
              variant="outline"
              class="shrink-0 border-border/80"
              :class="scoreColor(score)"
            >
              {{ verdict }}
            </Badge>
          </div>
          <p v-if="record.summary" class="text-sm leading-6 text-muted-foreground">{{ record.summary }}</p>
        </div>
      </div>

      <!-- Legend card -->
      <div class="flex flex-col gap-3 rounded-xl border border-border/60 p-5">
        <div class="flex items-center gap-2 text-sm font-medium">
          <ClipboardCheckIcon class="size-4 text-muted-foreground" />
          评分说明
        </div>
        <p class="text-sm leading-6 text-muted-foreground">各维度按权重加权平均。分数旁的置信度表示材料证据的完整程度。</p>
        <p v-if="record.source" class="text-xs text-muted-foreground">
          已解析 {{ record.source.characters || 0 }} 个字符{{ record.source.image_analyzed ? '，并完成图片理解' : '' }}
        </p>
      </div>
    </div>

    <!-- ─── Dimension scores ────────────────────────────── -->
    <section v-if="dimensions.length" class="flex flex-col gap-4" aria-labelledby="dimension-scores-title">
      <div>
        <h3 id="dimension-scores-title" class="text-sm font-semibold">九个维度评分</h3>
        <p class="mt-1 text-xs text-muted-foreground">每一项都保留评分依据、证据与待验证缺口。</p>
      </div>
      <div class="grid gap-3 lg:grid-cols-2">
        <article
          v-for="(dimension, i) in dimensions"
          :key="dimension.key"
          class="group rounded-xl border border-border/60 p-4 transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/20 hover:shadow-md"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <h4 class="truncate text-sm font-medium">{{ dimension.name }}</h4>
              <p class="text-xs text-muted-foreground">权重 {{ dimension.weight }}%</p>
            </div>
            <span class="shrink-0 text-lg font-semibold tabular-nums" :class="scoreColor(dimension.score)">
              {{ dimension.score.toFixed(0) }}
            </span>
          </div>
          <!-- Animated progress bar -->
          <div
            class="mt-3 h-1.5 overflow-hidden rounded-full bg-muted"
            role="progressbar"
            :aria-valuenow="dimension.score"
            aria-valuemin="0"
            aria-valuemax="100"
          >
            <div
              class="h-full rounded-full transition-[width] duration-700 ease-out"
              :class="progressColorClass(dimension.score)"
              :style="{
                width: mounted ? `${dimension.score}%` : '0%',
                transitionDelay: `${i * 60}ms`,
              }"
            />
          </div>
          <p v-if="dimension.reasoning" class="mt-3 text-sm leading-6 text-muted-foreground">{{ dimension.reasoning }}</p>
          <div v-if="dimension.evidence?.length" class="mt-3 rounded-lg bg-muted/50 p-3">
            <p class="mb-1 text-xs font-medium">依据</p>
            <ul class="list-disc space-y-1 pl-4 text-xs leading-5 text-muted-foreground">
              <li v-for="item in dimension.evidence" :key="item">{{ item }}</li>
            </ul>
          </div>
          <div v-if="dimension.gaps?.length" class="mt-2 rounded-lg border border-dashed border-border/80 p-3">
            <p class="mb-1 flex items-center gap-1 text-xs font-medium">
              <CircleAlertIcon class="size-3.5 text-amber-500" />
              待验证
            </p>
            <ul class="list-disc space-y-1 pl-4 text-xs leading-5 text-muted-foreground">
              <li v-for="item in dimension.gaps" :key="item">{{ item }}</li>
            </ul>
          </div>
          <p v-if="dimension.confidence !== undefined" class="mt-3 text-[11px] text-muted-foreground">
            证据置信度 {{ dimension.confidence.toFixed(0) }}%
          </p>
        </article>
      </div>
    </section>

    <!-- ─── Analysis process ───────────────────────────── -->
    <section v-if="process.length" class="flex flex-col gap-4" aria-labelledby="analysis-process-title">
      <div>
        <h3 id="analysis-process-title" class="text-sm font-semibold">分析过程</h3>
        <p class="mt-1 text-xs text-muted-foreground">保留 AI 实际检查过的步骤，方便追溯结论如何形成。</p>
      </div>
      <ol class="relative ml-2 border-l border-border/60 pl-6">
        <li
          v-for="(step, index) in process"
          :key="`${step.step}-${index}`"
          class="relative pb-6 last:pb-0"
        >
          <span class="absolute -left-[31px] top-0 flex size-5 items-center justify-center rounded-full bg-background ring-1 ring-border/60">
            <CircleCheckIcon class="size-4 text-primary" />
          </span>
          <div class="flex flex-wrap items-center gap-2">
            <h4 class="text-sm font-medium">{{ step.title }}</h4>
            <Badge v-if="step.status" variant="secondary" class="text-[10px]">
              {{ step.status === 'completed' ? '已完成' : step.status }}
            </Badge>
          </div>
          <p v-if="step.summary" class="mt-1 text-sm leading-6 text-muted-foreground">{{ step.summary }}</p>
          <ul v-if="step.questions?.length" class="mt-2 list-disc space-y-1 pl-4 text-xs leading-5 text-muted-foreground">
            <li v-for="question in step.questions" :key="question">{{ question }}</li>
          </ul>
          <Separator v-if="index < process.length - 1" class="mt-5 opacity-40" />
        </li>
      </ol>
    </section>

    <!-- ─── Next actions ───────────────────────────────── -->
    <section
      v-if="nextActions.length"
      class="rounded-xl border border-primary/20 bg-primary/5 p-5"
      aria-labelledby="next-actions-title"
    >
      <h3 id="next-actions-title" class="flex items-center gap-2 text-sm font-semibold text-primary">
        <ArrowRightIcon class="size-4" />
        优先验证动作
      </h3>
      <ol class="mt-3 list-decimal space-y-2 pl-5 text-sm leading-6 text-muted-foreground">
        <li v-for="action in nextActions" :key="action">{{ action }}</li>
      </ol>
    </section>
  </div>

  <pre v-else-if="rawText" class="overflow-x-auto rounded-xl bg-muted p-4 text-xs leading-6">{{ rawText }}</pre>
</template>
