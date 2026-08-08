<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { CircleAlertIcon, KeyRoundIcon, SaveIcon } from '@lucide/vue'
import { toast } from '@/lib/message'
import { getAISettings, saveAISettings } from '@/api/admin'
import AdminLayout from '@/components/layout/AdminLayout.vue'
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
import { errorMessage } from '@/lib/format'

const endpoint = ref('')
const model = ref('')
const apiKey = ref('')
const hasApiKey = ref(false)
const loading = ref(true)
const loadError = ref('')
const saving = ref(false)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const settings = await getAISettings()
    endpoint.value = settings.endpoint
    model.value = settings.model
    hasApiKey.value = settings.has_api_key
  } catch (error) {
    loadError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (saving.value) return
  saving.value = true
  try {
    await saveAISettings({
      endpoint: endpoint.value.trim(),
      model: model.value.trim(),
      apiKey: apiKey.value || undefined,
    })
    apiKey.value = ''
    toast.success('AI 服务配置已保存')
    await load()
  } catch (error) {
    toast.error('保存失败', { description: errorMessage(error) })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <AdminLayout>
    <div class="flex flex-col gap-6 px-4 py-8 sm:px-6">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">AI 设置</h1>
        <p class="text-sm text-muted-foreground">
          配置用于可行性分析的 OpenAI 兼容接口，保存后即时生效。
        </p>
      </div>

      <Alert v-if="loadError && !loading" variant="destructive">
        <CircleAlertIcon />
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{{ loadError }}</AlertDescription>
        <AlertAction>
          <Button variant="outline" size="sm" @click="load">重试</Button>
        </AlertAction>
      </Alert>

      <Card v-else-if="loading">
        <CardHeader>
          <Skeleton class="h-5 w-24" />
          <Skeleton class="h-4 w-56" />
        </CardHeader>
        <CardContent class="flex flex-col gap-3">
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-9 w-full" />
        </CardContent>
      </Card>

      <Card v-else>
        <CardHeader>
          <CardTitle>OpenAI 兼容服务</CardTitle>
          <CardDescription>
            分析任务通过该服务执行；API Key 在服务端加密存储。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form @submit.prevent="save">
            <FieldGroup>
              <Field>
                <FieldLabel for="ai-endpoint">服务地址</FieldLabel>
                <Input
                  id="ai-endpoint"
                  v-model="endpoint"
                  type="url"
                  required
                  placeholder="https://api.openai.com/v1"
                />
                <FieldDescription>
                  需兼容 OpenAI Chat Completions 接口，无需以 /chat/completions 结尾。
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
                <FieldLabel for="ai-key" class="flex items-center gap-2">
                  API Key
                  <Badge v-if="hasApiKey" variant="outline">已配置</Badge>
                  <Badge v-else variant="destructive">未配置</Badge>
                </FieldLabel>
                <Input
                  id="ai-key"
                  v-model="apiKey"
                  type="password"
                  autocomplete="new-password"
                  :placeholder="hasApiKey ? '留空则保持现有密钥' : '请输入 API Key'"
                />
                <FieldDescription>
                  密钥不会在页面回显；未配置密钥时分析任务会失败。
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

      <Alert v-if="!loading && !loadError && !hasApiKey">
        <KeyRoundIcon />
        <AlertTitle>尚未配置 API Key</AlertTitle>
        <AlertDescription>
          保存 endpoint、model 和 API Key 后，分析任务才能正常执行。
        </AlertDescription>
      </Alert>
    </div>
  </AdminLayout>
</template>
