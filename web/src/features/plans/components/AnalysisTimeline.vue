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
  <ol class="flex flex-col gap-3" aria-label="分析进度">
    <li v-for="step in steps" :key="step.label" class="flex items-center gap-3">
      <Spinner v-if="step.state === 'active'" class="size-4" />
      <CheckCircle2Icon v-else-if="step.state === 'done'" class="size-4 text-primary" />
      <XCircleIcon v-else-if="step.state === 'failed'" class="size-4 text-destructive" />
      <CircleIcon v-else class="size-4 text-muted-foreground/40" />
      <span
        class="text-sm"
        :class="step.state === 'pending' ? 'text-muted-foreground/60' : step.state === 'failed' ? 'text-destructive' : ''"
      >
        {{ step.label }}
      </span>
      <span v-if="step.time" class="ml-auto text-xs text-muted-foreground">
        {{ formatTime(step.time) }}
      </span>
    </li>
  </ol>
</template>
