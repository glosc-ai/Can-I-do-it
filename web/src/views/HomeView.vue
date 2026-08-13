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
import { Separator } from '@/components/ui/separator'
import { useAuthStore } from '@/features/auth/store'

const auth = useAuthStore()

interface Step {
  icon: FunctionalComponent
  title: string
  description: string
  number: string
}

interface Dimension {
  icon: FunctionalComponent
  title: string
  description: string
  color: string
  iconBg: string
}

const steps: Step[] = [
  {
    number: '01',
    icon: FileUpIcon,
    title: '描述想法或上传材料',
    description: '可以直接输入几句话，也可以上传 PDF、Word 和 TXT 计划书。',
  },
  {
    number: '02',
    icon: SparklesIcon,
    title: '提交 AI 分析',
    description: '一键发起可行性分析，后台异步处理，无需停留在页面等待。',
  },
  {
    number: '03',
    icon: ListChecksIcon,
    title: '查看可行性反馈',
    description: '集中查看结构化的分析结论，快速判断方向是否成立。',
  },
]

const dimensions: Dimension[] = [
  {
    icon: TrendingUpIcon,
    title: '市场',
    description: '市场规模、增长趋势与目标客群是否真实存在。',
    color: 'text-violet-500 dark:text-violet-400',
    iconBg: 'bg-violet-500/10 dark:bg-violet-400/10',
  },
  {
    icon: SwordsIcon,
    title: '竞争',
    description: '竞争格局、差异化空间与进入壁垒。',
    color: 'text-sky-500 dark:text-sky-400',
    iconBg: 'bg-sky-500/10 dark:bg-sky-400/10',
  },
  {
    icon: TriangleAlertIcon,
    title: '风险',
    description: '关键风险、合规隐患与资金压力。',
    color: 'text-amber-500 dark:text-amber-400',
    iconBg: 'bg-amber-500/10 dark:bg-amber-400/10',
  },
  {
    icon: TargetIcon,
    title: '机会',
    description: '机会窗口、切入点与下一步行动建议。',
    color: 'text-emerald-500 dark:text-emerald-400',
    iconBg: 'bg-emerald-500/10 dark:bg-emerald-400/10',
  },
]

const reportPreview = [
  {
    label: '可行性',
    score: 74,
    text: '方向整体成立：目标客群明确，需求真实存在，但冷启动依赖线下地推，需要预留至少 6 个月的现金流。',
    scoreColor: 'text-amber-500 dark:text-amber-400',
  },
  {
    label: '市场',
    score: 82,
    text: '社区咖啡市场近三年保持稳定增长，3 公里半径内常住人口约 4.2 万，工作日早间时段需求最集中。',
    scoreColor: 'text-emerald-500 dark:text-emerald-400',
  },
  {
    label: '风险',
    score: 55,
    text: '500 米内已有 3 家连锁品牌；房租占预估营收比例偏高，需在签约前锁定涨幅条款。',
    scoreColor: 'text-rose-500 dark:text-rose-400',
  },
  {
    label: '建议',
    score: null,
    text: '先以外带窗口验证早间高峰需求，再决定是否扩展堂食区域。',
    scoreColor: '',
  },
]
</script>

<template>
  <PublicLayout>
    <div class="mx-auto flex max-w-6xl flex-col gap-24 px-4 py-16 sm:px-6 sm:py-24">

      <!-- ─── Hero ─────────────────────────────────── -->
      <section class="relative flex flex-col items-start gap-8">
        <!-- 背景光晕 -->
        <div
          class="pointer-events-none absolute -left-24 -top-24 h-72 w-72 rounded-full bg-primary/8 blur-3xl dark:bg-primary/12"
          aria-hidden="true"
        />

        <div class="animate-fade-up">
          <Badge
            variant="outline"
            class="gap-1.5 border-primary/30 bg-primary/5 text-primary"
          >
            <SparklesIcon class="size-3 shrink-0" />
            <span class="shimmer-text font-medium">AI 商业可行性分析</span>
          </Badge>
        </div>

        <div class="flex flex-col gap-5 animate-fade-up delay-100">
          <h1 class="gradient-heading max-w-2xl text-5xl font-bold tracking-tight text-balance leading-[1.1] sm:text-7xl">
            我能做<br>这个吗
          </h1>
          <p class="max-w-xl text-base leading-7 text-muted-foreground sm:text-lg">
            输入一个想法或上传商业计划书，获得市场、竞争、风险与机会的结构化可行性分析，在投入之前先验证方向。
          </p>
        </div>

        <div class="flex flex-wrap gap-3 animate-fade-up delay-200">
          <Button
            v-if="auth.user"
            size="lg"
            as="a"
            href="/plans"
            class="gap-2 shadow-lg shadow-primary/20 transition-all duration-200 hover:scale-[1.03] hover:shadow-primary/30"
          >
            开始分析
            <ArrowRightIcon class="size-4 transition-transform duration-200 group-hover:translate-x-0.5" />
          </Button>
          <Button
            v-else
            size="lg"
            as="a"
            href="/api/v1/auth/login"
            class="gap-2 shadow-lg shadow-primary/20 transition-all duration-200 hover:scale-[1.03] hover:shadow-primary/30"
          >
            登录后开始分析
            <ArrowRightIcon class="size-4" />
          </Button>
          <Button
            variant="outline"
            size="lg"
            as="a"
            href="#how-it-works"
            class="transition-all duration-200 hover:border-primary/40 hover:text-primary"
          >
            了解如何开始
          </Button>
        </div>
      </section>

      <!-- ─── 报告预览 ───────────────────────────────── -->
      <section
        aria-labelledby="report-preview-title"
        class="flex flex-col gap-6 animate-fade-up delay-300"
      >
        <div class="flex flex-col gap-1.5">
          <h2 id="report-preview-title" class="text-2xl font-bold tracking-tight">
            结构化分析报告
          </h2>
          <p class="text-sm text-muted-foreground">
            每份计划书都会得到一份分维度的可行性报告，以下是一份示例片段。
          </p>
        </div>

        <div class="overflow-hidden rounded-2xl border bg-card shadow-sm transition-all duration-300 hover:shadow-md">
          <!-- 卡片头 -->
          <div class="flex flex-wrap items-center justify-between gap-3 border-b bg-muted/30 px-6 py-4">
            <div class="flex flex-wrap items-center gap-3">
              <span class="font-semibold">社区咖啡店商业计划书</span>
              <Badge class="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20">
                已完成
              </Badge>
            </div>
            <span class="text-xs text-muted-foreground">community-coffee-plan.pdf · 分析用时 42 秒</span>
          </div>
          <!-- 报告条目 -->
          <div class="divide-y">
            <div
              v-for="section in reportPreview"
              :key="section.label"
              class="flex items-start gap-4 px-6 py-4 transition-colors duration-150 hover:bg-muted/20"
            >
              <div class="flex min-w-[3rem] flex-col items-start gap-0.5 pt-0.5">
                <span class="text-xs font-semibold text-muted-foreground/70 uppercase tracking-wide">
                  {{ section.label }}
                </span>
                <span v-if="section.score !== null" class="text-lg font-bold tabular-nums" :class="section.scoreColor">
                  {{ section.score }}
                </span>
              </div>
              <p class="flex-1 text-sm leading-6 text-muted-foreground">{{ section.text }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- ─── 三步流程 ──────────────────────────────── -->
      <section id="how-it-works" class="flex flex-col gap-8" aria-labelledby="how-it-works-title">
        <div class="flex flex-col gap-1.5">
          <h2 id="how-it-works-title" class="text-2xl font-bold tracking-tight">
            三步开始
          </h2>
          <p class="text-sm text-muted-foreground">
            从上传到结论，整个过程只需要几分钟。
          </p>
        </div>

        <div class="grid gap-4 sm:grid-cols-2 md:grid-cols-3">
          <div
            v-for="(step, index) in steps"
            :key="step.title"
            class="group relative flex flex-col gap-4 rounded-2xl border bg-card p-6 shadow-sm transition-all duration-200 hover:-translate-y-1 hover:shadow-md hover:border-primary/30"
            :class="`animate-fade-up delay-${(index + 1) * 100}`"
          >
            <!-- 序号 -->
            <span class="text-4xl font-black tracking-tighter text-primary/10 select-none leading-none">
              {{ step.number }}
            </span>
            <!-- 图标 -->
            <span class="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary transition-transform duration-200 group-hover:scale-105">
              <component :is="step.icon" class="size-5" />
            </span>
            <div class="flex flex-col gap-1.5">
              <h3 class="font-semibold leading-tight">{{ step.title }}</h3>
              <p class="text-sm leading-6 text-muted-foreground">{{ step.description }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- ─── 分析维度 ──────────────────────────────── -->
      <section id="dimensions" class="flex flex-col gap-8" aria-labelledby="dimensions-title">
        <div class="flex flex-col gap-1.5">
          <h2 id="dimensions-title" class="text-2xl font-bold tracking-tight">
            九维分析
          </h2>
          <p class="text-sm text-muted-foreground">
            每份材料都会从九个维度得到结构化反馈，并按权重计算综合分数。
          </p>
        </div>

        <div class="grid gap-4 sm:grid-cols-2 md:grid-cols-4">
          <div
            v-for="(dimension, index) in dimensions"
            :key="dimension.title"
            class="group flex flex-col gap-3 rounded-2xl border bg-card p-5 shadow-sm transition-all duration-200 hover:-translate-y-1 hover:shadow-md"
            :class="`animate-fade-up delay-${(index + 1) * 100}`"
          >
            <span
              class="flex size-10 items-center justify-center rounded-xl transition-transform duration-200 group-hover:scale-105"
              :class="dimension.iconBg"
            >
              <component :is="dimension.icon" class="size-5" :class="dimension.color" />
            </span>
            <div class="flex flex-col gap-1">
              <h3 class="font-semibold" :class="dimension.color">{{ dimension.title }}</h3>
              <p class="text-sm leading-6 text-muted-foreground">{{ dimension.description }}</p>
            </div>
          </div>
        </div>

        <!-- 九维说明 -->
        <div class="rounded-2xl border border-dashed bg-muted/20 px-6 py-5">
          <p class="text-sm leading-7 text-muted-foreground">
            完整的九个维度还包括：
            <span class="font-medium text-foreground">需求验证、商业模式、获客路径、团队能力、财务预测、合规风险</span>。
            每个维度都会给出评分（0–100）、权重与置信度，形成加权综合分数。
          </p>
        </div>
      </section>

    </div>
  </PublicLayout>
</template>
