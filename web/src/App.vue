<script setup lang="ts">
import { RouterView } from 'vue-router'
import { Toaster } from '@/components/ui/sonner'
import { useTheme } from '@/composables/useTheme'

useTheme()
</script>

<template>
  <RouterView />
  <Toaster rich-colors position="top-right" />
</template>
