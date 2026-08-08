<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { CircleAlertIcon, SaveIcon } from '@lucide/vue'
import { toast } from 'vue-sonner'
import type { User } from '@/api/auth'
import { request } from '@/api/client'
import AppHeader from '@/components/layout/AppHeader.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
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

const users = ref<User[]>([])
const loading = ref(true)
const loadError = ref('')
const togglingId = ref<number | null>(null)

const endpoint = ref('')
const model = ref('')
const apiKey = ref('')
const saving = ref(false)

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '请求失败，请稍后重试'
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [usersResponse, settingsResponse] = await Promise.all([
      request<{ data: User[] }>('/admin/users'),
      request<{ data: { endpoint: string; model: string } }>('/admin/settings/ai'),
    ])
    users.value = usersResponse.data
    endpoint.value = settingsResponse.data.endpoint
    model.value = settingsResponse.data.model
  } catch (error) {
    loadError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

async function toggleStatus(user: User) {
  if (togglingId.value !== null) return
  const next = user.status === 'active' ? 'disabled' : 'active'
  togglingId.value = user.id
  try {
    await request(`/admin/users/${user.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ status: next }),
    })
    user.status = next
    toast.success(next === 'active' ? '已启用该用户' : '已禁用该用户')
  } catch (error) {
    toast.error('操作失败', { description: errorMessage(error) })
  } finally {
    togglingId.value = null
  }
}

async function saveSettings() {
  if (saving.value) return
  saving.value = true
  try {
    await request('/admin/settings/ai', {
      method: 'PATCH',
      body: JSON.stringify({
        endpoint: endpoint.value.trim(),
        model: model.value.trim(),
        apiKey: apiKey.value,
      }),
    })
    apiKey.value = ''
    toast.success('AI 服务配置已保存')
  } catch (error) {
    toast.error('保存失败', { description: errorMessage(error) })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />

    <main class="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-10 sm:px-6">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">管理后台</h1>
        <p class="text-sm text-muted-foreground">
          管理用户访问权限与 AI 分析服务。
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

      <div v-else-if="loading" class="flex flex-col gap-8">
        <section class="flex flex-col gap-3">
          <Skeleton class="h-6 w-24" />
          <div class="flex flex-col gap-2 rounded-lg border p-4">
            <Skeleton v-for="row in 4" :key="row" class="h-9 w-full" />
          </div>
        </section>
        <section class="flex flex-col gap-3">
          <Skeleton class="h-6 w-20" />
          <div class="flex flex-col gap-3 rounded-lg border p-4">
            <Skeleton class="h-8 w-full" />
            <Skeleton class="h-8 w-full" />
            <Skeleton class="h-8 w-2/3" />
          </div>
        </section>
      </div>

      <template v-else>
        <section class="flex flex-col gap-3" aria-labelledby="users-title">
          <div class="flex flex-col gap-1">
            <h2 id="users-title" class="text-lg font-semibold tracking-tight">
              用户管理
            </h2>
            <p class="text-sm text-muted-foreground">
              控制谁可以登录并使用分析功能。
            </p>
          </div>

          <div class="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>用户</TableHead>
                  <TableHead>角色</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead class="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="user in users" :key="user.id">
                  <TableCell>
                    <div class="flex flex-col">
                      <span class="font-medium">
                        {{ user.nickname || user.name || user.email }}
                      </span>
                      <span class="text-muted-foreground">{{ user.email }}</span>
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
                  <TableCell class="text-right">
                    <Button
                      v-if="user.role !== 'owner'"
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
        </section>

        <section aria-labelledby="ai-settings-title">
          <Card>
            <CardHeader>
              <CardTitle id="ai-settings-title">AI 服务</CardTitle>
              <CardDescription>
                配置用于可行性分析的 OpenAI 兼容接口，保存后即时生效。
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form @submit.prevent="saveSettings">
                <FieldGroup>
                  <Field>
                    <FieldLabel for="ai-endpoint">OpenAI-compatible URL</FieldLabel>
                    <Input
                      id="ai-endpoint"
                      v-model="endpoint"
                      type="url"
                      required
                      placeholder="https://api.openai.com/v1"
                    />
                    <FieldDescription>
                      服务地址需兼容 OpenAI Chat Completions 接口。
                    </FieldDescription>
                  </Field>
                  <Field>
                    <FieldLabel for="ai-model">模型</FieldLabel>
                    <Input
                      id="ai-model"
                      v-model="model"
                      required
                      placeholder="gpt-4o-mini"
                    />
                  </Field>
                  <Field>
                    <FieldLabel for="ai-key">API Key</FieldLabel>
                    <Input
                      id="ai-key"
                      v-model="apiKey"
                      type="password"
                      autocomplete="off"
                      placeholder="留空则保持现有密钥"
                    />
                    <FieldDescription>
                      密钥在服务端加密存储，保存后不会在页面回显。
                    </FieldDescription>
                  </Field>
                  <div>
                    <Button type="submit" :disabled="saving">
                      <Spinner v-if="saving" data-icon="inline-start" />
                      <SaveIcon v-else data-icon="inline-start" />
                      {{ saving ? '保存中…' : '保存配置' }}
                    </Button>
                  </div>
                </FieldGroup>
              </form>
            </CardContent>
          </Card>
        </section>
      </template>
    </main>
  </div>
</template>
