<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { CircleAlertIcon, FileTextIcon, Globe2Icon } from '@lucide/vue'
import PublicLayout from '@/components/layout/PublicLayout.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { useGalleryStore } from '@/features/gallery/store'
import { formatTime } from '@/lib/format'

const store = useGalleryStore()
const router = useRouter()

function openDetail(id: number) {
  router.push(`/gallery/${id}`)
}

function scoreColor(score?: number) {
  if (score === undefined) return 'text-muted-foreground'
  if (score >= 75) return 'text-emerald-600 dark:text-emerald-400'
  if (score >= 55) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}

onMounted(store.fetch)
</script>

<template>
  <PublicLayout>
    <div class="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-10 sm:px-6 sm:py-14">
      <div class="flex flex-col gap-2 animate-fade-up">
        <div class="flex items-center gap-2">
          <Badge variant="outline" class="gap-1.5 border-primary/30 bg-primary/5 text-primary">
            <Globe2Icon class="size-3" />
            项目广场
          </Badge>
        </div>
        <h1 class="text-3xl font-bold tracking-tight sm:text-4xl">看看别人在验证什么</h1>
        <p class="max-w-2xl text-sm leading-6 text-muted-foreground sm:text-base">
          这里展示用户主动公开的可行性分析报告，任何人都可以浏览完整结论。
        </p>
      </div>

      <Alert v-if="store.error && !store.loading" variant="destructive" class="animate-fade-in">
        <CircleAlertIcon />
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{{ store.error }}</AlertDescription>
        <AlertAction>
          <Button variant="outline" size="sm" @click="store.fetch">重试</Button>
        </AlertAction>
      </Alert>

      <template v-else>
        <div v-if="!store.loaded" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Skeleton v-for="row in 6" :key="row" class="h-40 w-full rounded-2xl" />
        </div>

        <div
          v-else-if="store.items.length > 0"
          class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 animate-fade-up"
        >
          <button
            v-for="item in store.items"
            :key="item.id"
            type="button"
            class="flex flex-col gap-3 rounded-2xl border bg-card p-5 text-left shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/30 hover:shadow-md"
            @click="openDetail(item.id)"
          >
            <div class="flex items-start justify-between gap-3">
              <h3 class="min-w-0 truncate text-base font-semibold">{{ item.title }}</h3>
              <span v-if="item.overall_score !== undefined" class="shrink-0 text-lg font-semibold tabular-nums" :class="scoreColor(item.overall_score)">
                {{ Math.round(item.overall_score) }}
              </span>
            </div>
            <Badge v-if="item.verdict" variant="outline" class="w-fit" :class="scoreColor(item.overall_score)">
              {{ item.verdict }}
            </Badge>
            <div class="mt-auto flex items-center gap-2 pt-2 text-xs text-muted-foreground">
              <img
                v-if="item.author_avatar"
                :src="item.author_avatar"
                :alt="item.author_name"
                class="size-5 rounded-full ring-1 ring-border"
                referrerpolicy="no-referrer"
              >
              <span
                v-else
                class="flex size-5 items-center justify-center rounded-full bg-primary/10 text-[10px] font-semibold text-primary"
                aria-hidden="true"
              >
                {{ item.author_name.charAt(0).toUpperCase() || '·' }}
              </span>
              <span class="truncate">{{ item.author_name }}</span>
              <span class="ml-auto shrink-0">{{ formatTime(item.created_at) }}</span>
            </div>
          </button>
        </div>

        <Empty v-else class="rounded-xl border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <FileTextIcon />
            </EmptyMedia>
            <EmptyTitle>还没有公开项目</EmptyTitle>
            <EmptyDescription>
              分析完成后，可以在详情页选择公开，让更多人看到你的分析结论。
            </EmptyDescription>
          </EmptyHeader>
        </Empty>

        <div v-if="store.loaded && store.items.length > 0 && store.hasMore" class="flex justify-center">
          <Button variant="outline" :disabled="store.loading" @click="store.loadMore">
            {{ store.loading ? '加载中…' : '加载更多' }}
          </Button>
        </div>
      </template>
    </div>
  </PublicLayout>
</template>
