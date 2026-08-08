<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { CircleAlertIcon, DownloadIcon, FileTextIcon, FileUpIcon, PlayIcon } from '@lucide/vue'
import { toast } from '@/lib/message'
import type { Plan } from '@/api/plans'
import WorkspaceLayout from '@/components/layout/WorkspaceLayout.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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
import UploadDropzone from '@/features/plans/components/UploadDropzone.vue'
import { usePlansStore } from '@/features/plans/store'
import { errorMessage, fileTypeLabel, formatSize, formatTime } from '@/lib/format'

const store = usePlansStore()
const router = useRouter()
const dropzoneRef = ref<InstanceType<typeof UploadDropzone> | null>(null)
const analyzingId = ref<number | null>(null)

async function onUpload(payload: { file: File; title: string }) {
  try {
    await store.upload(payload.file, payload.title)
    dropzoneRef.value?.reset()
    toast.success('上传成功', { description: '计划书已保存，可以开始分析。' })
  } catch (error) {
    toast.error('上传失败', { description: errorMessage(error) })
  }
}

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

const statCards = [
  { key: 'total', label: '全部计划书' },
  { key: 'active', label: '分析中' },
  { key: 'completed', label: '已完成' },
  { key: 'failed', label: '失败' },
] as const

onMounted(store.fetch)
onUnmounted(store.stop)
</script>

<template>
  <WorkspaceLayout>
    <div class="flex flex-col gap-8 px-4 py-8 sm:px-6">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="flex flex-col gap-1">
          <h1 class="text-2xl font-semibold tracking-tight">我的计划书</h1>
          <p class="text-sm text-muted-foreground">
            上传商业计划书并提交 AI 可行性分析。
          </p>
        </div>
        <Button variant="outline" @click="dropzoneRef?.focus()">
          <FileUpIcon data-icon="inline-start" />
          上传计划书
        </Button>
      </div>

      <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
        <Card v-for="card in statCards" :key="card.key" size="sm">
          <CardContent class="flex flex-col gap-1">
            <span class="text-xs text-muted-foreground">{{ card.label }}</span>
            <span class="text-2xl font-semibold tabular-nums">
              <Skeleton v-if="!store.loaded && store.loading" class="inline-block h-7 w-10" />
              <template v-else>{{ store.stats[card.key] }}</template>
            </span>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardContent>
          <UploadDropzone ref="dropzoneRef" :uploading="store.uploading" @submit="onUpload" />
        </CardContent>
      </Card>

      <section class="flex flex-col gap-3" aria-labelledby="plans-list-title">
        <h2 id="plans-list-title" class="text-lg font-semibold tracking-tight">
          计划书列表
        </h2>

        <Alert v-if="store.error && !store.loading" variant="destructive">
          <CircleAlertIcon />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{{ store.error }}</AlertDescription>
          <AlertAction>
            <Button variant="outline" size="sm" @click="store.fetch">
              重试
            </Button>
          </AlertAction>
        </Alert>

        <template v-else>
          <div v-if="!store.loaded || store.items.length > 0" class="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>标题</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>大小</TableHead>
                  <TableHead>版本</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead class="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
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
              <TableBody v-else>
                <TableRow
                  v-for="plan in store.items"
                  :key="plan.id"
                  class="cursor-pointer"
                  @click="openDetail(plan)"
                >
                  <TableCell>
                    <div class="flex flex-col">
                      <span class="max-w-40 truncate font-medium sm:max-w-64">{{ plan.title }}</span>
                      <span class="max-w-40 truncate text-xs text-muted-foreground sm:max-w-64">{{ plan.filename }}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">{{ fileTypeLabel(plan.filename, plan.mime_type) }}</Badge>
                  </TableCell>
                  <TableCell class="text-muted-foreground">{{ formatSize(plan.size_bytes) }}</TableCell>
                  <TableCell>
                    <Badge variant="outline">v{{ plan.version }}</Badge>
                  </TableCell>
                  <TableCell>
                    <PlanStatusBadge :status="plan.status" />
                  </TableCell>
                  <TableCell class="text-muted-foreground">
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
                      @click.stop
                    >
                      <DownloadIcon data-icon="inline-start" />
                      下载
                    </Button>
                    <Button
                      v-if="plan.status === 'uploaded'"
                      variant="outline"
                      size="sm"
                      :disabled="analyzingId !== null"
                      @click.stop="analyze(plan)"
                    >
                      <PlayIcon data-icon="inline-start" />
                      开始分析
                    </Button>
                    <Button v-else variant="ghost" size="sm" @click.stop="openDetail(plan)">
                      查看详情
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
              <Button variant="outline" size="sm" @click="dropzoneRef?.focus()">
                <FileUpIcon data-icon="inline-start" />
                上传第一份计划书
              </Button>
            </EmptyContent>
          </Empty>
        </template>
      </section>
    </div>
  </WorkspaceLayout>
</template>
