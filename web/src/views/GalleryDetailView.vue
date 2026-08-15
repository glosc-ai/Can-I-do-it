<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeftIcon, CircleAlertIcon, FileTextIcon } from '@lucide/vue'
import { getGalleryPlan, type GalleryPlanDetail } from '@/api/gallery'
import { APIError } from '@/api/client'
import PublicLayout from '@/components/layout/PublicLayout.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import AnalysisResult from '@/features/plans/components/AnalysisResult.vue'
import { errorMessage, fileTypeLabel, formatTime } from '@/lib/format'

const route = useRoute()
const router = useRouter()
const planId = Number(route.params.id)

const detail = ref<GalleryPlanDetail | null>(null)
const loading = ref(true)
const notFound = ref(!Number.isInteger(planId) || planId <= 0)
const loadError = ref('')

const isIdea = computed(() => detail.value?.plan.filename === 'idea-analysis.txt')

onMounted(async () => {
  if (notFound.value) {
    loading.value = false
    return
  }
  try {
    detail.value = await getGalleryPlan(planId)
  } catch (error) {
    if (error instanceof APIError && error.status === 404) {
      notFound.value = true
    } else {
      loadError.value = errorMessage(error)
    }
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <PublicLayout>
    <div class="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-10 sm:px-6">
      <div class="animate-fade-in">
        <Button
          variant="ghost"
          size="sm"
          class="gap-1.5 text-muted-foreground transition-all duration-150 hover:-translate-x-0.5 hover:text-foreground"
          @click="router.push('/gallery')"
        >
          <ArrowLeftIcon class="size-4" />
          返回项目广场
        </Button>
      </div>

      <div v-if="loading" class="flex flex-col gap-6 animate-fade-in">
        <Skeleton class="h-8 w-64" />
        <Skeleton class="h-48 w-full rounded-xl" />
      </div>

      <Alert v-else-if="notFound" variant="destructive" class="animate-fade-in rounded-xl">
        <CircleAlertIcon />
        <AlertTitle>项目不存在</AlertTitle>
        <AlertDescription>该项目可能已被作者设为私有，或已被删除。</AlertDescription>
      </Alert>

      <Alert v-else-if="loadError" variant="destructive" class="animate-fade-in rounded-xl">
        <CircleAlertIcon />
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{{ loadError }}</AlertDescription>
      </Alert>

      <template v-else-if="detail">
        <div class="flex flex-wrap items-start justify-between gap-4 animate-fade-up">
          <div class="flex min-w-0 flex-col gap-1.5">
            <h1 class="truncate text-2xl font-bold tracking-tight">{{ detail.plan.title }}</h1>
            <p class="flex items-center gap-1.5 truncate text-sm text-muted-foreground">
              <FileTextIcon class="size-3.5 shrink-0" />
              {{ isIdea ? '想法分析' : fileTypeLabel(detail.plan.filename, detail.plan.mime_type) }}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <img
              v-if="detail.author_avatar"
              :src="detail.author_avatar"
              :alt="detail.author_name"
              class="size-8 rounded-full ring-1 ring-border"
              referrerpolicy="no-referrer"
            >
            <span
              v-else
              class="flex size-8 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary"
              aria-hidden="true"
            >
              {{ detail.author_name.charAt(0).toUpperCase() || '·' }}
            </span>
            <div class="flex flex-col">
              <span class="text-sm font-medium">{{ detail.author_name }}</span>
              <span class="text-xs text-muted-foreground">{{ formatTime(detail.plan.created_at) }}</span>
            </div>
          </div>
        </div>

        <Card v-if="detail.analysis" class="rounded-xl shadow-sm animate-fade-up delay-100">
          <CardHeader class="border-b bg-muted/20 pb-4">
            <div class="flex items-center gap-2">
              <CardTitle class="text-base">分析结果</CardTitle>
              <Badge variant="secondary" class="text-xs">已公开</Badge>
            </div>
          </CardHeader>
          <CardContent class="pt-6">
            <AnalysisResult :result="detail.analysis.result || detail.analysis" />
          </CardContent>
        </Card>
      </template>
    </div>
  </PublicLayout>
</template>
