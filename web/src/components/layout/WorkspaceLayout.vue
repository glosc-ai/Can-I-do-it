<script setup lang="ts">
import { SparklesIcon } from '@lucide/vue'
import SidebarLayout from './SidebarLayout.vue'
import type { NavSection } from './nav'

const sections: NavSection[] = [
  {
    title: '工作区',
    items: [
      { to: '/plans', label: '分析工作台', icon: SparklesIcon },
    ],
  },
]
</script>

<template>
  <SidebarLayout :sections="sections">
    <slot />
  </SidebarLayout>
</template>
