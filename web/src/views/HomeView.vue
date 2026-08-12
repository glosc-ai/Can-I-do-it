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
import PublicLayout from '@/components/layout/PublicLayout.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
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
    title: '描述想法或上传材料',
    description: '可以直接输入几句话，也可以上传 PDF、Word 和 TXT 计划书。',
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

const reportPreview = [
  {
    title: '可行性',
    text: '方向整体成立：目标客群明确，需求真实存在，但冷启动依赖线下地推，需要预留至少 6 个月的现金流。',
  },
  {
    title: '市场',
    text: '社区咖啡市场近三年保持稳定增长，3 公里半径内常住人口约 4.2 万，工作日早间时段需求最集中。',
  },
  {
    title: '风险',
    text: '500 米内已有 3 家连锁品牌；房租占预估营收比例偏高，需在签约前锁定涨幅条款。',
  },
  {
    title: '建议',
    text: '先以外带窗口验证早间高峰需求，再决定是否扩展堂食区域。',
  },
]
</script>

<template>
  <PublicLayout>
    <div class="mx-auto flex max-w-6xl flex-col gap-16 px-4 py-14 sm:px-6 sm:py-20">
      <section class="flex max-w-3xl flex-col items-start gap-6">
        <Badge variant="outline">
          <SparklesIcon data-icon="inline-start" />
          AI 商业可行性分析
        </Badge>
        <div class="flex flex-col gap-4">
          <h1 class="text-4xl font-semibold tracking-tight text-balance sm:text-6xl">
            我能做这个吗
          </h1>
          <p class="max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
            输入一个想法或上传商业计划书，获得市场、竞争、风险与机会的结构化可行性分析，在投入之前先验证方向。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <Button v-if="auth.user" size="lg" as="a" href="/plans">
            开始分析
            <ArrowRightIcon data-icon="inline-end" />
          </Button>
          <Button v-else size="lg" as="a" href="/api/v1/auth/login">
            登录后分析想法
            <ArrowRightIcon data-icon="inline-end" />
          </Button>
          <Button variant="outline" size="lg" as="a" href="#how-it-works">
            了解如何开始
          </Button>
        </div>
      </section>

      <section aria-labelledby="report-preview-title" class="flex flex-col gap-6">
        <div class="flex flex-col gap-2">
          <h2 id="report-preview-title" class="text-2xl font-semibold tracking-tight">
            结构化分析报告
          </h2>
          <p class="text-sm text-muted-foreground">
            每份计划书都会得到一份分维度的可行性报告，以下是一份示例片段。
          </p>
        </div>
        <Card>
          <CardHeader>
            <CardTitle class="flex flex-wrap items-center gap-3">
              社区咖啡店商业计划书
              <Badge>已完成</Badge>
            </CardTitle>
            <CardDescription>community-coffee-plan.pdf · 分析用时 42 秒</CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col">
            <div
              v-for="(section, index) in reportPreview"
              :key="section.title"
              class="flex flex-col gap-1 py-4 first:pt-0 last:pb-0"
            >
              <h3 class="text-sm font-semibold">{{ section.title }}</h3>
              <p class="text-sm leading-6 text-muted-foreground">{{ section.text }}</p>
              <Separator v-if="index < reportPreview.length - 1" class="mt-4" />
            </div>
          </CardContent>
        </Card>
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
              <CardTitle class="flex items-center gap-2">
                <span class="flex size-8 items-center justify-center rounded-lg bg-muted">
                  <component :is="step.icon" class="size-4" />
                </span>
                {{ step.title }}
              </CardTitle>
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
            每份材料都会从九个维度得到结构化反馈，并按权重计算综合分数。
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
    </div>
  </PublicLayout>
</template>
