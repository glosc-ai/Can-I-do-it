<script setup lang="ts">
import AppHeader from './AppHeader.vue'
import { Separator } from '@/components/ui/separator'
</script>

<template>
  <div class="flex min-h-screen flex-col bg-background">
    <AppHeader />
    <main class="flex-1">
      <slot />
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
