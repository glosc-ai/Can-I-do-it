<script setup lang="ts">
import { ArrowLeftIcon, CloudIcon, FileSearchIcon, SettingsIcon, UsersIcon } from '@lucide/vue'
import SidebarLayout from './SidebarLayout.vue'
import type { NavSection } from './nav'

const sections: NavSection[] = [
  {
    title: '管理后台',
    items: [
      { to: '/admin/users', label: '用户管理', icon: UsersIcon },
      { to: '/admin/plans', label: '计划与分析', icon: FileSearchIcon },
      { to: '/admin/settings', label: 'AI 设置', icon: SettingsIcon },
      { to: '/admin/storage', label: 'R2 存储', icon: CloudIcon },
    ],
  },
  {
    title: '返回',
    items: [{ to: '/plans', label: '我的工作区', icon: ArrowLeftIcon }],
  },
]
</script>

<template>
  <SidebarLayout :sections="sections">
    <slot />
  </SidebarLayout>
</template>
