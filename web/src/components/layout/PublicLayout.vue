<script setup lang="ts">
import { SparklesIcon } from '@lucide/vue'
import AppHeader from './AppHeader.vue'
import { Separator } from '@/components/ui/separator'
</script>

<template>
  <div class="flex min-h-screen flex-col bg-background">
    <AppHeader />
    <main class="flex-1">
      <slot />
    </main>
    <Separator class="opacity-60" />
    <footer class="bg-muted/20">
      <div
        class="mx-auto flex max-w-6xl flex-col gap-2 px-4 py-8 sm:flex-row sm:items-center sm:justify-between sm:px-6"
      >
        <span class="flex items-center gap-1.5 text-sm font-medium text-muted-foreground">
          <span class="flex size-5 items-center justify-center rounded bg-primary/10 text-primary">
            <SparklesIcon class="size-3" />
          </span>
          Can I Do It
        </span>
        <span class="text-sm text-muted-foreground">先验证，再投入。</span>
      </div>
    </footer>
  </div>
</template>
