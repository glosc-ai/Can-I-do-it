<script setup lang="ts">
import { ref } from 'vue'
import { MenuIcon } from '@lucide/vue'
import { RouterLink } from 'vue-router'
import AppHeader from './AppHeader.vue'
import type { NavSection } from './nav'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

defineProps<{ sections: NavSection[] }>()

const open = ref(false)

// 基础链接样式（公共部分）
const linkBase =
  'group relative flex items-center gap-2.5 rounded-md px-3 py-2 text-sm text-muted-foreground transition-all duration-150 hover:bg-accent/60 hover:text-foreground'

// 激活状态：violet 左侧高光 + 背景 tint
const linkActive =
  '[&.router-link-active]:border-l-2 [&.router-link-active]:border-primary [&.router-link-active]:bg-accent/80 [&.router-link-active]:pl-[10px] [&.router-link-active]:font-medium [&.router-link-active]:text-primary'

const linkClass = `${linkBase} ${linkActive}`
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />
    <div class="mx-auto flex max-w-7xl">
      <!-- 桌面端侧边栏 -->
      <aside
        class="hidden min-h-[calc(100vh-57px)] w-60 shrink-0 border-r bg-sidebar px-3 py-6 lg:block"
      >
        <div class="flex flex-col gap-6">
          <div v-for="section in sections" :key="section.title">
            <p class="px-3 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              {{ section.title }}
            </p>
            <nav class="mt-1 grid gap-0.5" :aria-label="section.title">
              <RouterLink
                v-for="item in section.items"
                :key="item.to"
                :to="item.to"
                :class="linkClass"
              >
                <component
                  :is="item.icon"
                  class="size-4 shrink-0 transition-transform duration-150 group-hover:scale-105"
                />
                {{ item.label }}
              </RouterLink>
            </nav>
          </div>
        </div>
      </aside>

      <!-- 主内容区 -->
      <div class="min-w-0 flex-1">
        <!-- 移动端顶部 nav bar -->
        <div class="flex items-center border-b bg-sidebar px-4 py-3 lg:hidden">
          <Button
            variant="outline"
            size="icon-sm"
            aria-label="打开导航"
            class="transition-all duration-150 hover:border-primary/40 hover:text-primary"
            @click="open = true"
          >
            <MenuIcon class="size-4" />
          </Button>
        </div>
        <slot />
      </div>
    </div>

    <!-- 移动端抽屉 -->
    <Sheet v-model:open="open">
      <SheetContent side="left" class="w-72 gap-0 bg-sidebar p-0">
        <SheetHeader class="border-b px-4 py-4">
          <SheetTitle class="text-sm font-semibold">导航</SheetTitle>
        </SheetHeader>
        <div class="flex flex-col gap-6 px-3 py-4">
          <div v-for="section in sections" :key="section.title">
            <p class="px-3 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              {{ section.title }}
            </p>
            <nav class="mt-1 grid gap-0.5" :aria-label="section.title">
              <RouterLink
                v-for="item in section.items"
                :key="item.to"
                :to="item.to"
                :class="linkClass"
                @click="open = false"
              >
                <component
                  :is="item.icon"
                  class="size-4 shrink-0 transition-transform duration-150 group-hover:scale-105"
                />
                {{ item.label }}
              </RouterLink>
            </nav>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  </div>
</template>
