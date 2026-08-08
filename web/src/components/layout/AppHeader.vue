<script setup lang="ts">
import { computed } from 'vue'
import {
  FileTextIcon,
  LightbulbIcon,
  LogOutIcon,
  MoonIcon,
  ShieldIcon,
  SunIcon,
} from '@lucide/vue'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Separator } from '@/components/ui/separator'
import { useAuthStore } from '@/features/auth/store'
import { useTheme } from '@/composables/useTheme'

const auth = useAuthStore()
const theme = useTheme()

const displayName = computed(
  () => auth.user?.nickname || auth.user?.name || auth.user?.email || '',
)
const avatarInitial = computed(() => displayName.value.trim().charAt(0).toUpperCase() || '·')
const themeMode = computed(() => theme.mode.value)

async function signOut() {
  await auth.logout()
  window.location.href = '/'
}
</script>

<template>
  <header class="sticky top-0 z-40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
    <div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">
      <a class="flex items-center gap-2 font-medium" href="/">
        <span class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <LightbulbIcon class="size-4" />
        </span>
        <span>Can I Do It</span>
      </a>

      <nav class="flex items-center gap-1" aria-label="主导航">
        <template v-if="auth.user">
          <Button variant="ghost" size="sm" as="a" href="/plans">
            <FileTextIcon data-icon="inline-start" />
            计划书
          </Button>
          <Button v-if="auth.user.role === 'owner'" variant="ghost" size="sm" as="a" href="/admin/users">
            <ShieldIcon data-icon="inline-start" />
            后台
          </Button>
        </template>

        <Button
          variant="ghost"
          size="icon-sm"
          :aria-label="`切换主题（当前：${themeMode}）`"
          @click="theme.cycleMode"
        >
          <SunIcon v-if="themeMode === 'light'" />
          <MoonIcon v-else />
        </Button>

        <Button v-if="!auth.user" size="sm" as="a" href="/api/v1/auth/login">
          登录
        </Button>
        <DropdownMenu v-else>
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="sm" class="gap-2" :aria-label="`账户菜单（${displayName}）`">
              <img
                v-if="auth.user.avatar"
                :src="auth.user.avatar"
                :alt="displayName"
                class="size-5 rounded-full"
                referrerpolicy="no-referrer"
              >
              <span
                v-else
                class="flex size-5 items-center justify-center rounded-full bg-muted text-xs font-medium"
                aria-hidden="true"
              >
                {{ avatarInitial }}
              </span>
              <span class="hidden max-w-32 truncate sm:inline">{{ displayName }}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-56">
            <DropdownMenuLabel class="flex flex-col">
              <span>{{ displayName }}</span>
              <span v-if="auth.user.email" class="text-xs font-normal text-muted-foreground">
                {{ auth.user.email }}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem @select="signOut">
              <LogOutIcon />
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </nav>
    </div>
    <Separator />
  </header>
</template>
