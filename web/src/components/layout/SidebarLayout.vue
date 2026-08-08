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

const linkClass =
  'flex items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground [&.router-link-active]:bg-muted [&.router-link-active]:font-medium [&.router-link-active]:text-foreground'
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />
    <div class="mx-auto flex max-w-7xl">
      <aside class="hidden min-h-[calc(100vh-57px)] w-60 shrink-0 border-r px-4 py-6 lg:block">
        <div class="flex flex-col gap-6">
          <div v-for="section in sections" :key="section.title">
            <p class="px-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              {{ section.title }}
            </p>
            <nav class="mt-2 grid gap-1" :aria-label="section.title">
              <RouterLink
                v-for="item in section.items"
                :key="item.to"
                :to="item.to"
                :class="linkClass"
              >
                <component :is="item.icon" class="size-4" />
                {{ item.label }}
              </RouterLink>
            </nav>
          </div>
        </div>
      </aside>

      <div class="min-w-0 flex-1">
        <div class="flex items-center border-b px-4 py-3 lg:hidden">
          <Button variant="outline" size="icon-sm" aria-label="打开导航" @click="open = true">
            <MenuIcon />
          </Button>
        </div>
        <slot />
      </div>
    </div>

    <Sheet v-model:open="open">
      <SheetContent side="left" class="w-72 gap-6 p-4">
        <SheetHeader class="p-0">
          <SheetTitle>导航</SheetTitle>
        </SheetHeader>
        <div class="flex flex-col gap-6">
          <div v-for="section in sections" :key="section.title">
            <p class="px-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              {{ section.title }}
            </p>
            <nav class="mt-2 grid gap-1" :aria-label="section.title">
              <RouterLink
                v-for="item in section.items"
                :key="item.to"
                :to="item.to"
                :class="linkClass"
                @click="open = false"
              >
                <component :is="item.icon" class="size-4" />
                {{ item.label }}
              </RouterLink>
            </nav>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  </div>
</template>
