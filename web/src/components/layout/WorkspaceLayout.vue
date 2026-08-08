<script setup lang="ts">
import { ref } from 'vue'
import { FileTextIcon, MenuIcon, MoonIcon, SettingsIcon, ShieldIcon, SunIcon, UsersIcon, XIcon } from '@lucide/vue'
import { RouterLink } from 'vue-router'
import { useTheme } from '@/composables/useTheme'
import { useAuthStore } from '@/features/auth/store'
import AppHeader from './AppHeader.vue'
import { Button } from '@/components/ui/button'

defineProps<{ admin?: boolean }>()
const open = ref(false)
const auth = useAuthStore()
const theme = useTheme()
const navItems = [
  { to: '/plans', label: '我的计划书', icon: FileTextIcon },
]
const adminItems = [
  { to: '/admin/users', label: '用户管理', icon: UsersIcon },
  { to: '/admin/plans', label: '分析总览', icon: ShieldIcon },
  { to: '/admin/settings', label: 'AI 设置', icon: SettingsIcon },
]
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />
    <div class="mx-auto flex max-w-7xl">
      <aside class="hidden min-h-[calc(100vh-57px)] w-60 shrink-0 border-r px-4 py-6 lg:block">
        <div class="flex flex-col gap-6">
          <div>
            <p class="px-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">工作区</p>
            <nav class="mt-2 grid gap-1" aria-label="工作区导航">
              <RouterLink v-for="item in navItems" :key="item.to" :to="item.to" class="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-muted hover:text-foreground [&.router-link-active]:bg-muted [&.router-link-active]:font-medium [&.router-link-active]:text-foreground">
                <component :is="item.icon" class="size-4" />{{ item.label }}
              </RouterLink>
            </nav>
          </div>
          <div v-if="admin && auth.user?.role === 'owner'">
            <p class="px-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">管理</p>
            <nav class="mt-2 grid gap-1" aria-label="管理导航">
              <RouterLink v-for="item in adminItems" :key="item.to" :to="item.to" class="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-muted hover:text-foreground [&.router-link-active]:bg-muted [&.router-link-active]:font-medium [&.router-link-active]:text-foreground">
                <component :is="item.icon" class="size-4" />{{ item.label }}
              </RouterLink>
            </nav>
          </div>
        </div>
      </aside>
      <div class="min-w-0 flex-1">
        <div class="flex items-center justify-between border-b px-4 py-3 lg:hidden">
          <Button variant="outline" size="icon-sm" aria-label="打开导航" @click="open = true"><MenuIcon /></Button>
          <Button variant="ghost" size="icon-sm" :aria-label="`切换主题（当前：${theme.mode}）`" @click="theme.cycleMode"><SunIcon v-if="theme.mode === 'light'" /><MoonIcon v-else /></Button>
        </div>
        <slot />
      </div>
    </div>
    <div v-if="open" class="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true" aria-label="导航菜单">
      <button class="absolute inset-0 bg-foreground/20" aria-label="关闭导航" @click="open = false" />
      <aside class="relative h-full w-72 bg-background p-4 shadow-xl">
        <div class="flex items-center justify-between"><span class="font-medium">导航</span><Button variant="ghost" size="icon-sm" aria-label="关闭导航" @click="open = false"><XIcon /></Button></div>
        <nav class="mt-6 grid gap-1" aria-label="移动端导航">
          <RouterLink v-for="item in navItems" :key="item.to" :to="item.to" class="flex items-center gap-2 rounded-md px-3 py-2 text-sm hover:bg-muted" @click="open = false"><component :is="item.icon" class="size-4" />{{ item.label }}</RouterLink>
          <template v-if="admin && auth.user?.role === 'owner'"><RouterLink v-for="item in adminItems" :key="item.to" :to="item.to" class="flex items-center gap-2 rounded-md px-3 py-2 text-sm hover:bg-muted" @click="open = false"><component :is="item.icon" class="size-4" />{{ item.label }}</RouterLink></template>
        </nav>
      </aside>
    </div>
  </div>
</template>
