<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle2Icon, CircleIcon, XCircleIcon } from '@lucide/vue'
import { Spinner } from '@/components/ui/spinner'
import { formatTime } from '@/lib/format'

const props = defineProps<{
  status: string
  createdAt: string
  updatedAt: string
}>()

type StepState = 'done' | 'active' | 'pending' | 'failed'

interface Step {
  label: string
  state: StepState
  time?: string
}

const steps = computed<Step[]>(() => {
  const { status, createdAt, updatedAt } = props
  const failed = status === 'failed'
  return [
    { label: '已提交', state: 'done', time: createdAt },
    {
      label: '排队等待',
      state: status === 'queued' ? 'active' : 'done',
      time: status === 'queued' ? undefined : createdAt,
    },
    {
      label: 'AI 分析',
      state:
        status === 'running' ? 'active'
        : status === 'succeeded' ? 'done'
        : failed ? 'failed'
        : 'pending',
      time: status === 'succeeded' || failed ? updatedAt : undefined,
    },
    {
      label: failed ? '分析失败' : '生成结果',
      state: status === 'succeeded' ? 'done' : failed ? 'failed' : 'pending',
      time: status === 'succeeded' || failed ? updatedAt : undefined,
    },
  ]
})
</script>

<template>
  <ol class="relative flex flex-col gap-0" aria-label="分析进度">
    <li
      v-for="(step, index) in steps"
      :key="step.label"
      class="relative flex items-start gap-4 pb-5 last:pb-0"
    >
      <!-- 连接线（非最后一项） -->
      <div
        v-if="index < steps.length - 1"
        class="absolute left-[13px] top-7 bottom-0 w-px"
        :class="
          step.state === 'done'
            ? 'bg-primary/40'
            : step.state === 'failed'
              ? 'bg-destructive/30'
              : 'bg-border'
        "
        aria-hidden="true"
      />

      <!-- 步骤图标 -->
      <div class="relative z-10 mt-0.5 shrink-0">
        <!-- 激活中：脉冲圆点 -->
        <span v-if="step.state === 'active'" class="relative flex size-6 items-center justify-center">
          <span class="absolute inline-flex size-full animate-ping rounded-full bg-primary opacity-30" />
          <span class="relative flex size-6 items-center justify-center rounded-full bg-primary/10 ring-2 ring-primary/30">
            <Spinner class="size-3 text-primary" />
          </span>
        </span>
        <!-- 完成：scale-in 动画 -->
        <span v-else-if="step.state === 'done'" class="flex size-6 items-center justify-center animate-scale-in">
          <CheckCircle2Icon class="size-5 text-primary" />
        </span>
        <!-- 失败 -->
        <span v-else-if="step.state === 'failed'" class="flex size-6 items-center justify-center animate-scale-in">
          <XCircleIcon class="size-5 text-destructive" />
        </span>
        <!-- 待定 -->
        <span v-else class="flex size-6 items-center justify-center">
          <CircleIcon class="size-5 text-muted-foreground/30" />
        </span>
      </div>

      <!-- 步骤文字 -->
      <div class="flex min-w-0 flex-1 items-center justify-between gap-2 pt-0.5">
        <span
          class="text-sm leading-5 transition-colors duration-200"
          :class="
            step.state === 'pending'
              ? 'text-muted-foreground/50'
              : step.state === 'failed'
                ? 'font-medium text-destructive'
                : step.state === 'active'
                  ? 'font-medium text-foreground'
                  : 'text-foreground'
          "
        >
          {{ step.label }}
        </span>
        <span v-if="step.time" class="shrink-0 text-xs text-muted-foreground/60">
          {{ formatTime(step.time) }}
        </span>
      </div>
    </li>
  </ol>
</template>
