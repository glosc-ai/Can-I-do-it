<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowUpIcon,
  BrainCircuitIcon,
  FileTextIcon,
  LightbulbIcon,
  PaperclipIcon,
  SparklesIcon,
  XIcon,
} from '@lucide/vue'
import { toast } from '@/lib/message'
import { useIdeaAnalysisStore } from '@/features/ideas/store'
import { usePlansStore } from '@/features/plans/store'
import { errorMessage, fileTypeLabel, formatSize } from '@/lib/format'

type AnalysisMode = 'idea' | 'plan'

const ACCEPTED_EXTENSIONS = ['.pdf', '.doc', '.docx', '.txt', '.md', '.png', '.jpg', '.jpeg', '.webp']

const router = useRouter()
const ideaStore = useIdeaAnalysisStore()
const plansStore = usePlansStore()
const mode = ref<AnalysisMode>('idea')
const idea = ref('')
const file = ref<File | null>(null)
const title = ref('')
const fileError = ref('')
const dragging = ref(false)
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const queueing = ref(false)

const submitting = computed(() => ideaStore.submitting || plansStore.uploading || queueing.value)
const canSubmit = computed(() => {
  if (submitting.value) return false
  return mode.value === 'idea' ? idea.value.trim().length >= 10 : Boolean(file.value)
})

function setMode(nextMode: AnalysisMode) {
  mode.value = nextMode
  fileError.value = ''
  nextTick(() => {
    if (nextMode === 'idea') textareaRef.value?.focus()
    else if (!file.value) fileInputRef.value?.click()
  })
}

function isAccepted(candidate: File) {
  const name = candidate.name.toLowerCase()
  return ACCEPTED_EXTENSIONS.some(extension => name.endsWith(extension))
}

function selectFile(candidate: File | null | undefined) {
  fileError.value = ''
  if (!candidate) return
  if (!isAccepted(candidate)) {
    fileError.value = '仅支持 PDF、Word、TXT/Markdown 和常见图片格式'
    return
  }
  file.value = candidate
  title.value = candidate.name.replace(/\.[^.]+$/, '')
}

function onFileChange(event: Event) {
  selectFile((event.target as HTMLInputElement).files?.[0])
}

function onDrop(event: DragEvent) {
  dragging.value = false
  if (mode.value !== 'plan') mode.value = 'plan'
  selectFile(event.dataTransfer?.files?.[0])
}

function clearFile() {
  file.value = null
  title.value = ''
  fileError.value = ''
  if (fileInputRef.value) fileInputRef.value.value = ''
}

function attachFile() {
  mode.value = 'plan'
  nextTick(() => fileInputRef.value?.click())
}

async function submit() {
  if (!canSubmit.value) return
  try {
    if (mode.value === 'idea') {
      const plan = await ideaStore.submit({ idea: idea.value })
      toast.success('已开始分析', { description: 'AI 正在研究你的想法，报告完成后会自动保存。' })
      await router.push(`/plans/${plan.id}`)
      return
    }

    if (!file.value) return
    const plan = await plansStore.upload(file.value, title.value.trim() || file.value.name)
    if (!plan) return
    queueing.value = true
    await plansStore.analyze(plan.id)
    toast.success('已开始分析', { description: `「${plan.title}」已进入 AI 分析队列。` })
    await router.push(`/plans/${plan.id}`)
  } catch (error) {
    toast.error('提交失败', { description: errorMessage(error) })
  } finally {
    queueing.value = false
  }
}

function focus() {
  if (mode.value === 'idea') textareaRef.value?.focus()
  else fileInputRef.value?.click()
}

defineExpose({ focus })
</script>

<template>
  <section class="mx-auto w-full max-w-5xl" aria-labelledby="analysis-composer-title">
    <h2 id="analysis-composer-title" class="sr-only">开始新的可行性分析</h2>

    <div class="mb-6 flex justify-center">
      <div class="inline-flex rounded-full bg-muted p-1 ring-1 ring-border" role="tablist" aria-label="分析方式">
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'idea'"
          class="flex h-11 items-center gap-2 rounded-full px-5 text-sm font-medium transition-all sm:px-7 sm:text-base"
          :class="mode === 'idea' ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
          @click="setMode('idea')"
        >
          <LightbulbIcon class="size-4" />
          想法分析
          <SparklesIcon class="size-3.5" />
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'plan'"
          class="flex h-11 items-center gap-2 rounded-full px-5 text-sm font-medium transition-all sm:px-7 sm:text-base"
          :class="mode === 'plan' ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
          @click="setMode('plan')"
        >
          <FileTextIcon class="size-4" />
          计划书分析
        </button>
      </div>
    </div>

    <form
      class="overflow-hidden rounded-[2rem] border bg-card text-card-foreground shadow-sm transition-colors"
      :class="dragging ? 'border-primary/40 bg-muted/40' : ''"
      @submit.prevent="submit"
      @dragenter.prevent="dragging = true"
      @dragover.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @drop.prevent="onDrop"
    >
      <div class="min-h-48 px-6 pt-6 sm:min-h-56 sm:px-8 sm:pt-8">
        <textarea
          v-if="mode === 'idea'"
          ref="textareaRef"
          v-model="idea"
          rows="6"
          maxlength="4000"
          autofocus
          placeholder="描述你的想法：想解决什么问题、为谁解决、准备怎么做……"
          class="min-h-36 w-full resize-none bg-transparent text-base leading-7 text-foreground outline-none placeholder:text-muted-foreground/60 sm:text-lg"
          @keydown.meta.enter.prevent="submit"
          @keydown.ctrl.enter.prevent="submit"
        />

        <button
          v-else-if="!file"
          type="button"
          class="flex min-h-36 w-full flex-col items-center justify-center gap-3 rounded-2xl border border-dashed text-center text-muted-foreground transition-colors hover:border-primary/30 hover:bg-muted/30 hover:text-foreground"
          @click="fileInputRef?.click()"
        >
          <span class="flex size-11 items-center justify-center rounded-xl bg-muted text-foreground">
            <PaperclipIcon class="size-5" />
          </span>
          <span class="text-base text-foreground/80">拖入计划书，或点击选择文件</span>
          <span class="px-4 text-xs">PDF、Word、TXT、Markdown 或图片，最大 20 MB</span>
        </button>

        <div v-else class="flex min-h-36 flex-col justify-center gap-4">
          <div class="flex items-center gap-3 rounded-2xl border bg-muted/40 p-4">
            <span class="flex size-11 shrink-0 items-center justify-center rounded-xl bg-muted">
              <FileTextIcon class="size-5" />
            </span>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium">{{ file.name }}</p>
              <p class="mt-1 text-xs text-muted-foreground">{{ fileTypeLabel(file.name, file.type) }} · {{ formatSize(file.size) }}</p>
            </div>
            <button type="button" class="rounded-full p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" aria-label="移除文件" @click="clearFile">
              <XIcon class="size-4" />
            </button>
          </div>
          <input
            v-model="title"
            type="text"
            maxlength="160"
            placeholder="为这次分析添加标题"
            class="w-full border-b bg-transparent px-1 py-3 text-base text-foreground outline-none transition-colors placeholder:text-muted-foreground/60 focus:border-primary/40"
          >
        </div>
      </div>

      <input
        ref="fileInputRef"
        type="file"
        class="hidden"
        accept=".pdf,.doc,.docx,.txt,.md,.png,.jpg,.jpeg,.webp"
        @change="onFileChange"
      >

      <p v-if="fileError" class="px-6 pb-2 text-sm text-destructive sm:px-8" role="alert">{{ fileError }}</p>

      <div class="flex items-center justify-between gap-3 border-t px-5 pb-5 pt-3 sm:px-7 sm:pb-6">
        <div class="flex min-w-0 items-center gap-1 text-muted-foreground sm:gap-2">
          <button type="button" class="flex size-9 shrink-0 items-center justify-center rounded-full transition-colors hover:bg-muted hover:text-foreground" aria-label="上传计划书" @click="attachFile">
            <PaperclipIcon class="size-5" />
          </button>
          <span class="hidden h-9 items-center gap-2 rounded-full px-3 text-sm text-muted-foreground sm:flex">
            <BrainCircuitIcon class="size-4" />
            九维分析
          </span>
          <span class="hidden h-4 w-px bg-border sm:block" />
          <span class="hidden text-xs text-muted-foreground/70 md:inline">{{ mode === 'idea' ? `${idea.trim().length} / 4000` : '上传后自动开始分析' }}</span>
        </div>

        <button
          type="submit"
          class="flex size-12 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground transition-all hover:scale-105 disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:hover:scale-100"
          :disabled="!canSubmit"
          :aria-label="submitting ? '正在提交' : '开始分析'"
        >
          <span v-if="submitting" class="size-5 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
          <ArrowUpIcon v-else class="size-5" />
        </button>
      </div>
    </form>

    <p class="mt-4 text-center text-xs leading-5 text-muted-foreground">
      <SparklesIcon class="mr-1 inline size-3.5" />
      AI 会从需求、市场、竞争、商业模式、获客、合规与风险等九个维度展开分析
    </p>
  </section>
</template>
