<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { CircleAlertIcon } from '@lucide/vue'
import { toast } from '@/lib/message'
import { listAdminUsers, setUserStatus, type AdminUser } from '@/api/admin'
import AdminLayout from '@/components/layout/AdminLayout.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { errorMessage, formatTime } from '@/lib/format'

const users = ref<AdminUser[]>([])
const loading = ref(true)
const loadError = ref('')
const togglingId = ref<number | null>(null)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    users.value = await listAdminUsers()
  } catch (error) {
    loadError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

async function toggleStatus(user: AdminUser) {
  if (togglingId.value !== null) return
  const next = user.status === 'active' ? 'disabled' : 'active'
  togglingId.value = user.id
  try {
    await setUserStatus(user.id, next)
    user.status = next
    toast.success(next === 'active' ? '已启用该用户' : '已禁用该用户')
  } catch (error) {
    toast.error('操作失败', { description: errorMessage(error) })
  } finally {
    togglingId.value = null
  }
}

onMounted(load)
</script>

<template>
  <AdminLayout>
    <div class="flex flex-col gap-6 px-4 py-8 sm:px-6">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">用户管理</h1>
        <p class="text-sm text-muted-foreground">
          控制谁可以登录并使用分析功能。Owner 账号始终受到保护。
        </p>
      </div>

      <Alert v-if="loadError && !loading" variant="destructive">
        <CircleAlertIcon />
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{{ loadError }}</AlertDescription>
        <AlertAction>
          <Button variant="outline" size="sm" @click="load">
            重试
          </Button>
        </AlertAction>
      </Alert>

      <div class="overflow-x-auto rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>用户</TableHead>
              <TableHead>角色</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>最近活跃</TableHead>
              <TableHead class="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody v-if="loading">
            <TableRow v-for="row in 3" :key="row">
              <TableCell><Skeleton class="h-4 w-40" /></TableCell>
              <TableCell><Skeleton class="h-5 w-14" /></TableCell>
              <TableCell><Skeleton class="h-5 w-14" /></TableCell>
              <TableCell><Skeleton class="h-4 w-28" /></TableCell>
              <TableCell />
            </TableRow>
          </TableBody>
          <TableBody v-else>
            <TableRow v-for="user in users" :key="user.id">
              <TableCell>
                <div class="flex items-center gap-3">
                  <img
                    v-if="user.avatar"
                    :src="user.avatar"
                    :alt="user.nickname || user.name"
                    class="size-8 rounded-full"
                    referrerpolicy="no-referrer"
                  >
                  <span
                    v-else
                    class="flex size-8 items-center justify-center rounded-full bg-muted text-xs font-medium"
                    aria-hidden="true"
                  >
                    {{ (user.nickname || user.name || user.email).charAt(0).toUpperCase() }}
                  </span>
                  <div class="flex flex-col">
                    <span class="font-medium">{{ user.nickname || user.name || user.email }}</span>
                    <span class="text-muted-foreground">{{ user.email }}</span>
                  </div>
                </div>
              </TableCell>
              <TableCell>
                <Badge v-if="user.role === 'owner'">Owner</Badge>
                <Badge v-else variant="secondary">成员</Badge>
              </TableCell>
              <TableCell>
                <Badge v-if="user.status === 'active'" variant="outline">正常</Badge>
                <Badge v-else variant="destructive">已禁用</Badge>
              </TableCell>
              <TableCell class="text-muted-foreground">
                {{ formatTime(user.updated_at) }}
              </TableCell>
              <TableCell class="text-right">
                <Badge v-if="user.role === 'owner'" variant="outline">受保护</Badge>
                <Button
                  v-else
                  :variant="user.status === 'active' ? 'destructive' : 'outline'"
                  size="sm"
                  :disabled="togglingId !== null"
                  @click="toggleStatus(user)"
                >
                  <Spinner v-if="togglingId === user.id" data-icon="inline-start" />
                  {{ user.status === 'active' ? '禁用' : '启用' }}
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>
  </AdminLayout>
</template>
