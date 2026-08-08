import { computed, ref } from 'vue'

export type ThemeMode = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'can-i-do-it-theme'
const mode = ref<ThemeMode>('system')
let initialized = false

function systemIsDark(): boolean {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
}

function apply(modeValue: ThemeMode): void {
  if (typeof document === 'undefined') return
  const dark = modeValue === 'dark' || (modeValue === 'system' && systemIsDark())
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
}

function initialize(): void {
  if (initialized || typeof window === 'undefined') return
  initialized = true
  const stored = window.localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'system') mode.value = stored
  apply(mode.value)
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (mode.value === 'system') apply(mode.value)
  })
}

function setMode(next: ThemeMode): void {
  mode.value = next
  if (typeof window !== 'undefined') window.localStorage.setItem(STORAGE_KEY, next)
  apply(next)
}

function cycleMode(): void {
  setMode(mode.value === 'system' ? 'light' : mode.value === 'light' ? 'dark' : 'system')
}

export function useTheme() {
  initialize()
  return { mode, isDark: computed(() => mode.value === 'dark' || (mode.value === 'system' && systemIsDark())), setMode, cycleMode }
}
