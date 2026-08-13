<script setup lang="ts">
import { Badge } from '@/components/ui/badge'
import { planStatusMeta } from '../status'

defineProps<{ status: string }>()
</script>

<template>
  <Badge
    :variant="planStatusMeta(status).variant"
    :class="planStatusMeta(status).className"
  >
    {{ planStatusMeta(status).label }}
  </Badge>
</template>
