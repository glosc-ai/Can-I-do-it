<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { CircleAlertIcon, FileTextIcon, FileUpIcon, PlayIcon, UploadIcon } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { analyzePlan, listPlans, uploadPlan, type Plan } from '@/api/plans'
import AppHeader from '@/components/layout/AppHeader.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge, type BadgeVariants } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const ACCEPTED_EXTENSIONS = ['.pdf', '.doc', '.docx', '.txt']

const STATUS_META: Record<string, { label: string; variant: NonNullable<BadgeVariants['variant']> }> = {
  uploaded: { label: '待分析', variant: 'secondary' },
  queued: { label: '排队中', variant: 'outline' },
  processing: { label: '分析中', variant: 'outline' },
  completed: { label: '已完成', variant: 'default' },
  failed: { label: '失败', variant: 'destructive' },
}

const plans = ref<Plan[]>([])
const listLoading = ref(true)
const listError = ref('')

const title = ref('')
const file = ref<File | null>(null)
const fileError = ref('')
const uploading = ref(false)

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '请求失败，请稍后重试'
}

function statusMeta(status: string) {
  return STATUS_META[status] ?? { label: status, variant: 'outline' as const }
}

function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '-'
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB']
  let value = bytes
  let unit = -1
  do {
    value /= 1024
    unit += 1
  } while (value >= 1024 && unit < units.length - 1)
  return `${value.toFixed(1)} ${units[unit]}`
}

function formatTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function load() {
  listLoading.value = true
  listError.value = ''
  try {
    plans.value = await listPlans()
  } catch (error) {
    listError.value = errorMessage(error)
  } finally {
    listLoading.value = false
  }
}

function isAcceptedFile(candidate: File): boolean {
  const name = candidate.name.toLowerCase()
  return ACCEPTED_EXTENSIONS.some(extension => name.endsWith(extension))
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const selected = input.files?.[0] ?? null
  fileError.value = ''

  if (selected && !isAcceptedFile(selected)) {
    file.value = null
    input.value = ''
    fileError.value = '仅支持 .pdf、.doc、.docx、.txt 格式的文件'
    return
  }

  file.value = selected
  if (selected && !title.value.trim()) {
    title.value = selected.name.replace(/\.[^.]+$/, '')
  }
}

function resetFileInput() {
  const input = document.getElementById('plan-file') as HTMLInputElement | null
  if (input) input.value = ''
}

async function submit() {
  if (uploading.value) return
  if (!file.value) {
    fileError.value = '请选择要上传的文件'
    return
  }

  uploading.value = true
  try {
    await uploadPlan(file.value, title.value.trim() || file.value.name)
    toast.success('上传成功', { description: '计划书已保存，可以开始分析。' })
    title.value = ''
    file.value = null
    fileError.value = ''
    resetFileInput()
    await load()
  } catch (error) {
    toast.error('上传失败', { description: errorMessage(error) })
  } finally {
    uploading.value = false
  }
}

async function analyze(plan: Plan) {
  try {
    await analyzePlan(plan.id)
    plan.status = 'queued'
    toast.success('已提交分析', { description: `「${plan.title}」正在排队分析，稍后回来查看结果。` })
  } catch (error) {
    toast.error('提交分析失败', { description: errorMessage(error) })
  }
}

function focusUploadForm() {
  const input = document.getElementById('plan-title')
  input?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  input?.focus()
}

onMounted(load)
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />

    <main class="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-10 sm:px-6">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">商业计划书</h1>
        <p class="text-sm text-muted-foreground">
          上传文档并提交 AI 可行性分析，结果会集中展示在这里。
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>上传计划书</CardTitle>
          <CardDescription>支持 PDF、Word 和 TXT 文件，上传后可随时发起分析。</CardDescription>
        </CardHeader>
        <CardContent>
          <form @submit.prevent="submit">
            <FieldGroup>
              <Field>
                <FieldLabel for="plan-title">标题</FieldLabel>
                <Input
                  id="plan-title"
                  v-model="title"
                  placeholder="例如：社区咖啡店商业计划书"
                />
              </Field>
              <Field :data-invalid="fileError ? 'true' : undefined">
                <FieldLabel for="plan-file">文件</FieldLabel>
                <Input
                  id="plan-file"
                  type="file"
                  accept=".pdf,.doc,.docx,.txt"
                  :aria-invalid="fileError ? 'true' : undefined"
                  @change="onFileChange"
                />
                <FieldDescription>仅支持 .pdf、.doc、.docx、.txt 格式。</FieldDescription>
                <FieldError v-if="fileError">{{ fileError }}</FieldError>
              </Field>
              <div>
                <Button type="submit" :disabled="uploading">
                  <Spinner v-if="uploading" data-icon="inline-start" />
                  <UploadIcon v-else data-icon="inline-start" />
                  {{ uploading ? '上传中…' : '上传计划书' }}
                </Button>
              </div>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>

      <section class="flex flex-col gap-3" aria-labelledby="plans-list-title">
        <div class="flex items-center justify-between gap-4">
          <h2 id="plans-list-title" class="text-lg font-semibold tracking-tight">
            我的计划书
          </h2>
          <span v-if="!listLoading && !listError" class="text-sm text-muted-foreground">
            共 {{ plans.length }} 份
          </span>
        </div>

        <Alert v-if="listError" variant="destructive">
          <CircleAlertIcon />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{{ listError }}</AlertDescription>
          <AlertAction>
            <Button variant="outline" size="sm" @click="load">
              重试
            </Button>
          </AlertAction>
        </Alert>

        <template v-else>
          <div v-if="listLoading || plans.length > 0" class="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>标题</TableHead>
                  <TableHead>文件名</TableHead>
                  <TableHead>大小</TableHead>
                  <TableHead>版本</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>上传时间</TableHead>
                  <TableHead class="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody v-if="listLoading && plans.length === 0">
                <TableRow v-for="row in 3" :key="row">
                  <TableCell><Skeleton class="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton class="h-4 w-40" /></TableCell>
                  <TableCell><Skeleton class="h-4 w-12" /></TableCell>
                  <TableCell><Skeleton class="h-5 w-8" /></TableCell>
                  <TableCell><Skeleton class="h-5 w-14" /></TableCell>
                  <TableCell><Skeleton class="h-4 w-28" /></TableCell>
                  <TableCell />
                </TableRow>
              </TableBody>
              <TableBody v-else>
                <TableRow v-for="plan in plans" :key="plan.id">
                  <TableCell>
                    <span class="block max-w-40 truncate font-medium sm:max-w-64">
                      {{ plan.title }}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span class="block max-w-40 truncate text-muted-foreground sm:max-w-56">
                      {{ plan.filename }}
                    </span>
                  </TableCell>
                  <TableCell class="text-muted-foreground">
                    {{ formatSize(plan.size_bytes) }}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">v{{ plan.version }}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge :variant="statusMeta(plan.status).variant">
                      {{ statusMeta(plan.status).label }}
                    </Badge>
                  </TableCell>
                  <TableCell class="text-muted-foreground">
                    {{ formatTime(plan.created_at) }}
                  </TableCell>
                  <TableCell class="text-right">
                    <Button
                      v-if="plan.status === 'uploaded'"
                      variant="outline"
                      size="sm"
                      @click="analyze(plan)"
                    >
                      <PlayIcon data-icon="inline-start" />
                      开始分析
                    </Button>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          <Empty v-else class="border">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <FileTextIcon />
              </EmptyMedia>
              <EmptyTitle>还没有计划书</EmptyTitle>
              <EmptyDescription>
                上传第一份商业计划书，获取市场、竞争、风险与机会的可行性分析。
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button variant="outline" size="sm" @click="focusUploadForm">
                <FileUpIcon data-icon="inline-start" />
                上传第一份计划书
              </Button>
            </EmptyContent>
          </Empty>
        </template>
      </section>
    </main>
  </div>
</template>
