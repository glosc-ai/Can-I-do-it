<script setup lang="ts">
import { ref } from 'vue'
import { CloudUploadIcon, FileTextIcon, XIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { fileTypeLabel, formatSize } from '@/lib/format'

const ACCEPTED_EXTENSIONS = ['.pdf', '.doc', '.docx', '.txt']

const props = defineProps<{ uploading: boolean }>()
const emit = defineEmits<{ submit: [{ file: File; title: string }] }>()

const file = ref<File | null>(null)
const title = ref('')
const fileError = ref('')
const dragging = ref(false)
const inputRef = ref<HTMLInputElement | null>(null)

function isAccepted(candidate: File): boolean {
  const name = candidate.name.toLowerCase()
  return ACCEPTED_EXTENSIONS.some(extension => name.endsWith(extension))
}

function select(candidate: File | null | undefined) {
  fileError.value = ''
  if (!candidate) return
  if (!isAccepted(candidate)) {
    file.value = null
    if (inputRef.value) inputRef.value.value = ''
    fileError.value = '仅支持 .pdf、.doc、.docx、.txt 格式的文件'
    return
  }
  file.value = candidate
  if (!title.value.trim()) {
    title.value = candidate.name.replace(/\.[^.]+$/, '')
  }
}

function onFileChange(event: Event) {
  select((event.target as HTMLInputElement).files?.[0])
}

function onDrop(event: DragEvent) {
  dragging.value = false
  select(event.dataTransfer?.files?.[0])
}

function clearFile() {
  file.value = null
  fileError.value = ''
  if (inputRef.value) inputRef.value.value = ''
}

function submit() {
  if (props.uploading) return
  if (!file.value) {
    fileError.value = '请选择要上传的文件'
    return
  }
  emit('submit', { file: file.value, title: title.value.trim() || file.value.name })
}

function reset() {
  clearFile()
  title.value = ''
}

defineExpose({ reset, focus: () => inputRef.value?.click() })
</script>

<template>
  <form class="flex flex-col gap-4" @submit.prevent="submit">
    <button
      type="button"
      class="flex min-h-36 w-full flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 py-8 text-center transition-colors"
      :class="dragging ? 'border-primary bg-muted/60' : 'border-border hover:bg-muted/40'"
      :aria-label="file ? '更换文件' : '选择或拖入计划书文件'"
      @click="inputRef?.click()"
      @dragenter.prevent="dragging = true"
      @dragover.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @drop.prevent="onDrop"
    >
      <template v-if="file">
        <span class="flex size-10 items-center justify-center rounded-lg bg-muted">
          <FileTextIcon class="size-5" />
        </span>
        <span class="max-w-full truncate text-sm font-medium">{{ file.name }}</span>
        <span class="text-xs text-muted-foreground">
          {{ fileTypeLabel(file.name, file.type) }} · {{ formatSize(file.size) }}
        </span>
      </template>
      <template v-else>
        <span class="flex size-10 items-center justify-center rounded-lg bg-muted">
          <CloudUploadIcon class="size-5" />
        </span>
        <span class="text-sm font-medium">拖拽文件到这里，或点击选择</span>
        <span class="text-xs text-muted-foreground">支持 .pdf、.doc、.docx、.txt，单个文件最大 20 MB</span>
      </template>
    </button>
    <input
      id="plan-file"
      ref="inputRef"
      type="file"
      class="hidden"
      accept=".pdf,.doc,.docx,.txt"
      :aria-invalid="fileError ? 'true' : undefined"
      @change="onFileChange"
    >
    <p v-if="fileError" class="text-sm text-destructive" role="alert">{{ fileError }}</p>

    <div v-if="file" class="flex flex-col gap-3 sm:flex-row sm:items-end">
      <div class="flex-1">
        <label for="plan-title" class="mb-1.5 block text-sm font-medium">标题</label>
        <Input id="plan-title" v-model="title" placeholder="例如：社区咖啡店商业计划书" />
      </div>
      <div class="flex gap-2">
        <Button type="submit" :disabled="uploading">
          <Spinner v-if="uploading" data-icon="inline-start" />
          <CloudUploadIcon v-else data-icon="inline-start" />
          {{ uploading ? '上传中…' : '上传计划书' }}
        </Button>
        <Button type="button" variant="ghost" :disabled="uploading" @click="clearFile">
          <XIcon data-icon="inline-start" />
          移除
        </Button>
      </div>
    </div>
  </form>
</template>
