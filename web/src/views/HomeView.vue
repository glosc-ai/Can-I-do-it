<script setup lang="ts">
import type { FunctionalComponent } from 'vue'
import {
  ArrowRightIcon,
  FileUpIcon,
  ListChecksIcon,
  SparklesIcon,
  SwordsIcon,
  TargetIcon,
  TrendingUpIcon,
  TriangleAlertIcon,
} from '@lucide/vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { useAuthStore } from '@/features/auth/store'

const auth = useAuthStore()

interface Step {
  icon: FunctionalComponent
  title: string
  description: string
}

const steps: Step[] = [
  {
    icon: FileUpIcon,
    title: '上传计划书',
    description: '支持 PDF、Word 和 TXT 文档，每次上传都会保留版本记录。',
  },
  {
    icon: SparklesIcon,
    title: '提交 AI 分析',
    description: '一键发起可行性分析，后台异步处理，无需停留在页面等待。',
  },
  {
    icon: ListChecksIcon,
    title: '查看可行性反馈',
    description: '集中查看结构化的分析结论，快速判断方向是否成立。',
  },
]

const dimensions: Step[] = [
  {
    icon: TrendingUpIcon,
    title: '市场',
    description: '市场规模、增长趋势与目标客群是否真实存在。',
  },
  {
    icon: SwordsIcon,
    title: '竞争',
    description: '竞争格局、差异化空间与进入壁垒。',
  },
  {
    icon: TriangleAlertIcon,
    title: '风险',
    description: '关键风险、合规隐患与资金压力。',
  },
  {
    icon: TargetIcon,
    title: '机会',
    description: '机会窗口、切入点与下一步行动建议。',
  },
]
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />

    <main class="mx-auto flex max-w-6xl flex-col gap-16 px-4 py-14 sm:px-6 sm:py-20">
      <section class="flex max-w-3xl flex-col items-start gap-6">
        <Badge variant="outline">
          <SparklesIcon data-icon="inline-start" />
          AI 商业可行性分析
        </Badge>
        <div class="flex flex-col gap-4">
          <h1 class="text-4xl font-semibold tracking-tight text-balance sm:text-6xl">
            Can I Do It
          </h1>
          <p class="max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
            上传商业计划书，获得市场、竞争、风险与机会的结构化可行性分析，在投入之前先验证想法。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <Button v-if="auth.user" size="lg" as="a" href="/plans">
            进入我的计划书
            <ArrowRightIcon data-icon="inline-end" />
          </Button>
          <Button v-else size="lg" as="a" href="/api/v1/auth/login">
            使用 SSO 登录
            <ArrowRightIcon data-icon="inline-end" />
          </Button>
          <Button variant="outline" size="lg" as="a" href="#how-it-works">
            了解如何开始
          </Button>
        </div>
      </section>

      <section id="how-it-works" class="flex flex-col gap-6" aria-labelledby="how-it-works-title">
        <div class="flex flex-col gap-2">
          <h2 id="how-it-works-title" class="text-2xl font-semibold tracking-tight">
            三步开始
          </h2>
          <p class="text-sm text-muted-foreground">
            从上传到结论，整个过程只需要几分钟。
          </p>
        </div>
        <div class="grid gap-4 sm:grid-cols-2 md:grid-cols-3">
          <Card v-for="step in steps" :key="step.title">
            <CardHeader>
              <CardTitle>{{ step.title }}</CardTitle>
              <CardAction>
                <span class="flex size-8 items-center justify-center rounded-lg bg-muted">
                  <component :is="step.icon" class="size-4" />
                </span>
              </CardAction>
              <CardDescription>{{ step.description }}</CardDescription>
            </CardHeader>
          </Card>
        </div>
      </section>

      <section id="dimensions" class="flex flex-col gap-6" aria-labelledby="dimensions-title">
        <div class="flex flex-col gap-2">
          <h2 id="dimensions-title" class="text-2xl font-semibold tracking-tight">
            分析维度
          </h2>
          <p class="text-sm text-muted-foreground">
            每份计划书都会从四个维度得到结构化反馈。
          </p>
        </div>
        <div class="grid gap-6 sm:grid-cols-2 md:grid-cols-4">
          <div v-for="dimension in dimensions" :key="dimension.title" class="flex flex-col gap-2">
            <span class="flex size-9 items-center justify-center rounded-lg bg-muted">
              <component :is="dimension.icon" class="size-4" />
            </span>
            <h3 class="text-sm font-medium">{{ dimension.title }}</h3>
            <p class="text-sm text-muted-foreground">{{ dimension.description }}</p>
          </div>
        </div>
      </section>
    </main>

    <Separator />
    <footer>
      <div
        class="mx-auto flex max-w-6xl flex-col gap-2 px-4 py-8 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-6"
      >
        <span>Can I Do It</span>
        <span>先验证，再投入。</span>
      </div>
    </footer>
  </div>
</template>
