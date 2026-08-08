<script setup lang="ts">
import { computed } from 'vue'
import { FileTextIcon, LightbulbIcon, LogOutIcon, MoonIcon, ShieldIcon, SunIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { useAuthStore } from '@/features/auth/store'
import { useTheme } from '@/composables/useTheme'

const auth = useAuthStore()
const theme = useTheme()

const displayName = computed(
  () => auth.user?.nickname || auth.user?.name || auth.user?.email || '',
)

async function signOut() {
  await auth.logout()
  window.location.href = '/api/v1/auth/login'
}
</script>

<template>
  <header class="sticky top-0 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
    <div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">
      <a class="flex items-center gap-2 font-medium" href="/">
        <span class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <LightbulbIcon class="size-4" />
        </span>
        <span>Can I Do It</span>
      </a>

      <nav class="flex items-center gap-1" aria-label="主导航">
        <Button variant="ghost" size="icon-sm" :aria-label="`切换主题（当前：${theme.mode}）`" @click="theme.cycleMode">
          <SunIcon v-if="theme.mode === 'light'" />
          <MoonIcon v-else />
        </Button>
        <Button variant="ghost" size="sm" as="a" href="/plans">
          <FileTextIcon data-icon="inline-start" />
          计划书
        </Button>
        <Button v-if="auth.user?.role === 'owner'" variant="ghost" size="sm" as="a" href="/admin">
          <ShieldIcon data-icon="inline-start" />
          后台
        </Button>

        <Button v-if="!auth.user" size="sm" as="a" href="/api/v1/auth/login">
          登录
        </Button>
        <template v-else>
          <span class="px-2 text-sm text-muted-foreground">{{ displayName }}</span>
          <Button variant="ghost" size="icon" aria-label="退出登录" @click="signOut">
            <LogOutIcon />
          </Button>
        </template>
      </nav>
    </div>
    <Separator />
  </header>
</template>
