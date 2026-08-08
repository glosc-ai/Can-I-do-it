<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ChevronLeftIcon, ChevronRightIcon, CircleAlertIcon } from '@lucide/vue'
import {
  listAdminAnalysis,
  listAdminPlans,
  type AdminAnalysis,
  type PageMeta,
} from '@/api/admin'
import type { Plan } from '@/api/plans'
import AdminLayout from '@/components/layout/AdminLayout.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
import { analysisStatusMeta } from '@/features/plans/status'
import { Badge } from '@/components/ui/badge'
import { errorMessage, fileTypeLabel, formatSize, formatTime } from '@/lib/format'

const PAGE_SIZE = 20

const STATUS_OPTIONS = [
  { value: 'all', label: '全部状态' },
  { value: 'queued', label: '排队中' },
  { value: 'running', label: '分析中' },
  { value: 'succeeded', label: '已完成' },
  { value: 'failed', label: '失败' },
]

const plans = ref<Plan[]>([])
const plansLoading = ref(true)
const plansError = ref('')

const jobs = ref<AdminAnalysis[]>([])
const meta = ref<PageMeta>({ page: 1, page_size: PAGE_SIZE, total: 0 })
const status = ref('all')
const jobsLoading = ref(true)
const jobsError = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(meta.value.total / meta.value.page_size)))

async function loadPlans() {
  plansLoading.value = true
  plansError.value = ''
  try {
    plans.value = await listAdminPlans()
  } catch (error) {
    plansError.value = errorMessage(error)
  } finally {
    plansLoading.value = false
  }
}

async function loadJobs(page = 1) {
  jobsLoading.value = true
  jobsError.value = ''
  try {
    const response = await listAdminAnalysis({
      status: status.value === 'all' ? undefined : status.value,
      page,
      pageSize: PAGE_SIZE,
    })
    jobs.value = response.data
    meta.value = response.meta
  } catch (error) {
    jobsError.value = errorMessage(error)
  } finally {
    jobsLoading.value = false
  }
}

function onStatusChange(value: unknown) {
  status.value = String(value)
  loadJobs(1)
}

function jobStatusMeta(job: AdminAnalysis) {
  return analysisStatusMeta(job.status)
}

onMounted(() => {
  loadPlans()
  loadJobs()
})
</script>

<template>
  <AdminLayout>
    <div class="flex flex-col gap-8 px-4 py-8 sm:px-6">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">计划与分析</h1>
        <p class="text-sm text-muted-foreground">
          查看全部用户的计划书和分析任务执行情况。
        </p>
      </div>

      <section class="flex flex-col gap-3" aria-labelledby="admin-plans-title">
        <h2 id="admin-plans-title" class="text-lg font-semibold tracking-tight">
          全部计划书
        </h2>

        <Alert v-if="plansError && !plansLoading" variant="destructive">
          <CircleAlertIcon />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{{ plansError }}</AlertDescription>
          <AlertAction>
            <Button variant="outline" size="sm" @click="loadPlans">重试</Button>
          </AlertAction>
        </Alert>

        <div class="overflow-x-auto rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>标题</TableHead>
                <TableHead>用户 ID</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>大小</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>更新时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody v-if="plansLoading">
              <TableRow v-for="row in 3" :key="row">
                <TableCell><Skeleton class="h-4 w-40" /></TableCell>
                <TableCell><Skeleton class="h-4 w-10" /></TableCell>
                <TableCell><Skeleton class="h-5 w-10" /></TableCell>
                <TableCell><Skeleton class="h-4 w-12" /></TableCell>
                <TableCell><Skeleton class="h-5 w-14" /></TableCell>
                <TableCell><Skeleton class="h-4 w-28" /></TableCell>
              </TableRow>
            </TableBody>
            <TableBody v-else-if="plans.length > 0">
              <TableRow v-for="plan in plans" :key="plan.id">
                <TableCell>
                  <span class="block max-w-48 truncate font-medium sm:max-w-72">{{ plan.title }}</span>
                </TableCell>
                <TableCell class="text-muted-foreground">{{ plan.user_id }}</TableCell>
                <TableCell>
                  <Badge variant="secondary">{{ fileTypeLabel(plan.filename, plan.mime_type) }}</Badge>
                </TableCell>
                <TableCell class="text-muted-foreground">{{ formatSize(plan.size_bytes) }}</TableCell>
                <TableCell>
                  <PlanStatusBadge :status="plan.status" />
                </TableCell>
                <TableCell class="text-muted-foreground">
                  {{ formatTime(plan.updated_at || plan.created_at) }}
                </TableCell>
              </TableRow>
            </TableBody>
            <TableBody v-else>
              <TableRow>
                <TableCell colspan="6" class="py-8 text-center text-muted-foreground">
                  还没有任何计划书
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </section>

      <section class="flex flex-col gap-3" aria-labelledby="admin-analysis-title">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 id="admin-analysis-title" class="text-lg font-semibold tracking-tight">
            分析任务
          </h2>
          <Select :model-value="status" @update:model-value="onStatusChange">
            <SelectTrigger class="w-36" aria-label="按状态筛选">
              <SelectValue placeholder="全部状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="option in STATUS_OPTIONS" :key="option.value" :value="option.value">
                {{ option.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Alert v-if="jobsError && !jobsLoading" variant="destructive">
          <CircleAlertIcon />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{{ jobsError }}</AlertDescription>
          <AlertAction>
            <Button variant="outline" size="sm" @click="loadJobs(meta.page)">重试</Button>
          </AlertAction>
        </Alert>

        <div class="overflow-x-auto rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>计划书</TableHead>
                <TableHead>用户 ID</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>错误</TableHead>
                <TableHead>更新时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody v-if="jobsLoading">
              <TableRow v-for="row in 5" :key="row">
                <TableCell><Skeleton class="h-4 w-40" /></TableCell>
                <TableCell><Skeleton class="h-4 w-10" /></TableCell>
                <TableCell><Skeleton class="h-5 w-14" /></TableCell>
                <TableCell><Skeleton class="h-4 w-32" /></TableCell>
                <TableCell><Skeleton class="h-4 w-28" /></TableCell>
              </TableRow>
            </TableBody>
            <TableBody v-else-if="jobs.length > 0">
              <TableRow v-for="job in jobs" :key="job.id">
                <TableCell>
                  <span class="block max-w-48 truncate font-medium sm:max-w-72">{{ job.plan_title }}</span>
                </TableCell>
                <TableCell class="text-muted-foreground">{{ job.user_id }}</TableCell>
                <TableCell>
                  <Badge :variant="jobStatusMeta(job).variant">{{ jobStatusMeta(job).label }}</Badge>
                </TableCell>
                <TableCell>
                  <span class="block max-w-48 truncate text-muted-foreground">{{ job.error || '—' }}</span>
                </TableCell>
                <TableCell class="text-muted-foreground">{{ formatTime(job.updated_at) }}</TableCell>
              </TableRow>
            </TableBody>
            <TableBody v-else>
              <TableRow>
                <TableCell colspan="5" class="py-8 text-center text-muted-foreground">
                  没有匹配的分析任务
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>

        <div class="flex items-center justify-between text-sm text-muted-foreground">
          <span>共 {{ meta.total }} 条</span>
          <div class="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon-sm"
              :disabled="jobsLoading || meta.page <= 1"
              aria-label="上一页"
              @click="loadJobs(meta.page - 1)"
            >
              <ChevronLeftIcon />
            </Button>
            <span class="tabular-nums">{{ meta.page }} / {{ totalPages }}</span>
            <Button
              variant="outline"
              size="icon-sm"
              :disabled="jobsLoading || meta.page >= totalPages"
              aria-label="下一页"
              @click="loadJobs(meta.page + 1)"
            >
              <ChevronRightIcon />
            </Button>
          </div>
        </div>
      </section>
    </div>
  </AdminLayout>
</template>
