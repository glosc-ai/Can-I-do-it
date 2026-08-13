<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  LightbulbIcon,
  LogOutIcon,
  MonitorIcon,
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

const scrolled = ref(false)
const themeSpinKey = ref(0)

function handleScroll() {
  scrolled.value = window.scrollY > 8
}

onMounted(() => window.addEventListener('scroll', handleScroll, { passive: true }))
onUnmounted(() => window.removeEventListener('scroll', handleScroll))

const displayName = computed(
  () => auth.user?.nickname || auth.user?.name || auth.user?.email || '',
)
const avatarInitial = computed(() => displayName.value.trim().charAt(0).toUpperCase() || '·')
const themeMode = computed(() => theme.mode.value)
const themeIcon = computed(() => {
  if (themeMode.value === 'light') return SunIcon
  if (themeMode.value === 'dark') return MoonIcon
  return MonitorIcon
})
const themeLabel = computed(() => {
  if (themeMode.value === 'light') return '亮色'
  if (themeMode.value === 'dark') return '暗色'
  return '跟随系统'
})

function cycleTheme() {
  themeSpinKey.value++
  theme.cycleMode()
}

async function signOut() {
  await auth.logout()
  window.location.href = '/'
}
</script>

<template>
  <header
    class="sticky top-0 z-40 transition-all duration-300"
    :class="[
      scrolled
        ? 'bg-background/95 shadow-sm shadow-border/60 backdrop-blur-md supports-[backdrop-filter]:bg-background/80'
        : 'bg-background/80 backdrop-blur-sm',
    ]"
  >
    <div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">
      <!-- Logo -->
      <a
        class="group flex items-center gap-2.5 font-semibold transition-opacity duration-200 hover:opacity-80"
        href="/"
      >
        <span
          class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm transition-transform duration-200 group-hover:scale-105"
        >
          <LightbulbIcon class="size-4" />
        </span>
        <span class="tracking-tight">Can I Do It</span>
      </a>

      <!-- 右侧导航 -->
      <nav class="flex items-center gap-0.5" aria-label="主导航">
        <template v-if="auth.user">
          <Button
            variant="ghost"
            size="sm"
            as="a"
            href="/plans"
            class="gap-1.5 transition-all duration-200 hover:text-primary"
            aria-label="分析工作台"
          >
            <LightbulbIcon class="size-4" />
            <span class="hidden md:inline">分析工作台</span>
          </Button>
          <Button
            v-if="auth.user.role === 'owner'"
            variant="ghost"
            size="sm"
            as="a"
            href="/admin/users"
            class="gap-1.5 transition-all duration-200 hover:text-primary"
            aria-label="后台"
          >
            <ShieldIcon class="size-4" />
            <span class="hidden md:inline">后台</span>
          </Button>
        </template>

        <!-- 主题切换按钮 -->
        <Button
          variant="ghost"
          size="icon-sm"
          :aria-label="`切换主题（当前：${themeLabel}）`"
          class="relative overflow-hidden transition-all duration-200 hover:text-primary"
          @click="cycleTheme"
        >
          <component
            :is="themeIcon"
            :key="themeSpinKey"
            class="size-4 animate-spin-once"
          />
        </Button>

        <!-- 未登录 -->
        <Button
          v-if="!auth.user"
          size="sm"
          as="a"
          href="/api/v1/auth/login"
          class="ml-1 transition-all duration-200 hover:scale-[1.02]"
        >
          登录
        </Button>

        <!-- 已登录：用户菜单 -->
        <DropdownMenu v-else>
          <DropdownMenuTrigger as-child>
            <Button
              variant="ghost"
              size="sm"
              class="ml-1 gap-2 transition-all duration-200"
              :aria-label="`账户菜单（${displayName}）`"
            >
              <span class="relative">
                <img
                  v-if="auth.user.avatar"
                  :src="auth.user.avatar"
                  :alt="displayName"
                  class="size-6 rounded-full ring-1 ring-border transition-all duration-200 hover:ring-primary/60"
                  referrerpolicy="no-referrer"
                >
                <span
                  v-else
                  class="flex size-6 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary ring-1 ring-primary/20"
                  aria-hidden="true"
                >
                  {{ avatarInitial }}
                </span>
              </span>
              <span class="hidden max-w-32 truncate text-sm sm:inline">{{ displayName }}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-56 animate-fade-in">
            <DropdownMenuLabel class="flex flex-col gap-0.5">
              <span class="font-medium">{{ displayName }}</span>
              <span v-if="auth.user.email" class="text-xs font-normal text-muted-foreground">
                {{ auth.user.email }}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem class="gap-2 text-destructive focus:text-destructive" @select="signOut">
              <LogOutIcon class="size-4" />
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </nav>
    </div>
    <Separator class="opacity-60" />
  </header>
</template>
