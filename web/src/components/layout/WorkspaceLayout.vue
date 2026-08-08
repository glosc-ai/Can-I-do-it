<script setup lang="ts">
import { FileTextIcon } from '@lucide/vue'
import SidebarLayout from './SidebarLayout.vue'
import type { NavSection } from './nav'

const sections: NavSection[] = [
  {
    title: '工作区',
    items: [{ to: '/plans', label: '我的计划书', icon: FileTextIcon }],
  },
]
</script>

<template>
  <SidebarLayout :sections="sections">
    <slot />
  </SidebarLayout>
</template>
