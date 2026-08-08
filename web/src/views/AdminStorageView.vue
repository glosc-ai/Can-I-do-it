<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { CheckCircle2Icon, CircleAlertIcon, CloudIcon, RefreshCwIcon, SaveIcon, Trash2Icon } from '@lucide/vue'
import { toast } from '@/lib/message'
import {
  deleteAdminAsset,
  getStorageSettings,
  listAdminAssets,
  saveStorageSettings,
  testStorageSettings,
  type StorageAsset,
  type StorageSettings,
} from '@/api/admin'
import AdminLayout from '@/components/layout/AdminLayout.vue'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { errorMessage, formatSize, formatTime } from '@/lib/format'

const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const loadError = ref('')
const storage = ref<StorageSettings | null>(null)
const enabled = ref(false)
const endpoint = ref('')
const bucket = ref('')
const publicUrl = ref('')
const region = ref('auto')
const accessKeyId = ref('')
const secretAccessKey = ref('')
const forcePathStyle = ref(false)
const clearCredentials = ref(false)

const assets = ref<StorageAsset[]>([])
const assetsLoading = ref(true)
const assetSource = ref('all')
const deletingId = ref<number | null>(null)

function applySettings(value: StorageSettings) {
  storage.value = value
  enabled.value = value.enabled
  endpoint.value = value.endpoint
  bucket.value = value.bucket
  publicUrl.value = value.public_url
  region.value = value.region || 'auto'
  forcePathStyle.value = value.force_path_style
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    applySettings(await getStorageSettings())
  } catch (error) {
    loadError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

async function loadAssets() {
  assetsLoading.value = true
  try {
    assets.value = (await listAdminAssets({
      source: assetSource.value === 'all' ? undefined : assetSource.value,
      pageSize: 100,
    })).data
  } catch (error) {
    toast.error('素材列表加载失败', { description: errorMessage(error) })
  } finally {
    assetsLoading.value = false
  }
}

async function save() {
  if (saving.value) return
  saving.value = true
  try {
    applySettings(await saveStorageSettings({
      enabled: enabled.value,
      endpoint: endpoint.value.trim(),
      bucket: bucket.value.trim(),
      publicUrl: publicUrl.value.trim() || undefined,
      region: region.value.trim() || 'auto',
      accessKeyId: accessKeyId.value.trim() || undefined,
      secretAccessKey: secretAccessKey.value || undefined,
      forcePathStyle: forcePathStyle.value,
      clearCredentials: clearCredentials.value,
    }))
    accessKeyId.value = ''
    secretAccessKey.value = ''
    clearCredentials.value = false
    toast.success('R2 存储配置已保存')
  } catch (error) {
    toast.error('保存失败', { description: errorMessage(error) })
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (testing.value) return
  testing.value = true
  try {
    await testStorageSettings()
    toast.success('R2 连接正常', { description: '已成功访问配置的 Bucket。' })
  } catch (error) {
    toast.error('R2 连接失败', { description: errorMessage(error) })
  } finally {
    testing.value = false
  }
}

async function removeAsset(asset: StorageAsset) {
  if (deletingId.value !== null || !window.confirm(`确定删除「${asset.name}」吗？对象和记录都会被删除。`)) return
  deletingId.value = asset.id
  try {
    await deleteAdminAsset(asset.id)
    assets.value = assets.value.filter((item) => item.id !== asset.id)
    toast.success('素材已删除')
  } catch (error) {
    toast.error('删除失败', { description: errorMessage(error) })
  } finally {
    deletingId.value = null
  }
}

function sourceLabel(source: StorageAsset['source']) {
  return { upload: '用户上传', ai_generated: 'AI 生成', fetched: '外部获取' }[source]
}

onMounted(() => {
  load()
  loadAssets()
})
</script>

<template>
  <AdminLayout>
    <div class="flex flex-col gap-8 px-4 py-8 sm:px-6">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">R2 存储</h1>
        <p class="text-sm text-muted-foreground">
          保存用户上传、AI 生成和外部获取的资料素材。密钥只在服务端加密保存。
        </p>
      </div>

      <Alert v-if="loadError && !loading" variant="destructive">
        <CircleAlertIcon />
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{{ loadError }}</AlertDescription>
        <AlertAction><Button variant="outline" size="sm" @click="load">重试</Button></AlertAction>
      </Alert>

      <Card v-else>
        <CardHeader>
          <CardTitle class="flex items-center gap-2"><CloudIcon class="size-5" />Cloudflare R2</CardTitle>
          <CardDescription>
            R2 关闭时上传保存到服务器的 UPLOAD_DIR；启用后新对象写入 R2，配置不完整时会拒绝上传以避免落错位置。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form class="flex flex-col gap-5" @submit.prevent="save">
            <label class="flex items-center gap-3 text-sm">
              <input v-model="enabled" type="checkbox" class="size-4 accent-primary">
              <span class="font-medium">启用 R2 作为对象存储</span>
              <Badge v-if="storage?.using_r2" variant="outline">当前使用 R2</Badge>
              <Badge v-else variant="secondary">当前使用本地存储</Badge>
            </label>
            <FieldGroup>
              <Field>
                <FieldLabel for="r2-endpoint">S3 API Endpoint</FieldLabel>
                <Input id="r2-endpoint" v-model="endpoint" type="url" placeholder="https://<account-id>.r2.cloudflarestorage.com" :required="enabled">
                </Input>
                <FieldDescription>Cloudflare Dashboard → R2 → API Tokens 中的 S3 Endpoint。</FieldDescription>
              </Field>
              <div class="grid gap-5 sm:grid-cols-2">
                <Field>
                  <FieldLabel for="r2-bucket">Bucket</FieldLabel>
                  <Input id="r2-bucket" v-model="bucket" placeholder="my-assets" :required="enabled">
                  </Input>
                </Field>
                <Field>
                  <FieldLabel for="r2-region">Region</FieldLabel>
                  <Input id="r2-region" v-model="region" placeholder="auto">
                  </Input>
                </Field>
              </div>
              <Field>
                <FieldLabel for="r2-public-url">Public URL（可选）</FieldLabel>
                <Input id="r2-public-url" v-model="publicUrl" type="url" placeholder="https://cdn.example.com">
                </Input>
                <FieldDescription>留空则生成 15 分钟有效的私有预签名链接。</FieldDescription>
              </Field>
              <div class="grid gap-5 sm:grid-cols-2">
                <Field>
                  <FieldLabel for="r2-access-key">Access Key ID</FieldLabel>
                  <Input id="r2-access-key" v-model="accessKeyId" autocomplete="off" :placeholder="storage?.has_credentials ? '留空则保持现有密钥' : 'R2 API Token Access Key'">
                  </Input>
                </Field>
                <Field>
                  <FieldLabel for="r2-secret-key">Secret Access Key</FieldLabel>
                  <Input id="r2-secret-key" v-model="secretAccessKey" type="password" autocomplete="new-password" :placeholder="storage?.has_credentials ? '留空则保持现有密钥' : 'R2 API Token Secret'">
                  </Input>
                </Field>
              </div>
            </FieldGroup>
            <label class="flex items-center gap-3 text-sm text-muted-foreground">
              <input v-model="forcePathStyle" type="checkbox" class="size-4 accent-primary">
              <span>使用 path-style 请求（仅在兼容 S3 网关要求时开启）</span>
            </label>
            <label v-if="storage?.has_credentials" class="flex items-center gap-3 text-sm text-destructive">
              <input v-model="clearCredentials" type="checkbox" class="size-4 accent-destructive">
              <span>清除已保存的 R2 凭据</span>
            </label>
            <div class="flex flex-wrap gap-2">
              <Button type="submit" :disabled="saving || loading">
                <Spinner v-if="saving" data-icon="inline-start" /><SaveIcon v-else data-icon="inline-start" />
                {{ saving ? '保存中…' : '保存配置' }}
              </Button>
              <Button type="button" variant="outline" :disabled="testing || !storage?.configured" @click="testConnection">
                <Spinner v-if="testing" data-icon="inline-start" /><CheckCircle2Icon v-else data-icon="inline-start" />
                {{ testing ? '测试中…' : '测试连接' }}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <section class="flex flex-col gap-3" aria-labelledby="storage-assets-title">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 id="storage-assets-title" class="text-lg font-semibold tracking-tight">素材对象</h2>
            <p class="text-sm text-muted-foreground">管理员可查看、下载或删除所有用户的存储对象。</p>
          </div>
          <div class="flex items-center gap-2">
            <Select v-model="assetSource" @update:model-value="loadAssets">
              <SelectTrigger class="w-32" aria-label="素材来源"><SelectValue placeholder="全部来源" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部来源</SelectItem>
                <SelectItem value="upload">用户上传</SelectItem>
                <SelectItem value="ai_generated">AI 生成</SelectItem>
                <SelectItem value="fetched">外部获取</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" size="icon-sm" :disabled="assetsLoading" aria-label="刷新素材" @click="loadAssets">
              <RefreshCwIcon :class="assetsLoading ? 'animate-spin' : ''" />
            </Button>
          </div>
        </div>
        <div class="overflow-x-auto rounded-lg border">
          <Table>
            <TableHeader><TableRow><TableHead>名称</TableHead><TableHead>用户 ID</TableHead><TableHead>来源</TableHead><TableHead>大小</TableHead><TableHead>创建时间</TableHead><TableHead class="text-right">操作</TableHead></TableRow></TableHeader>
            <TableBody v-if="assetsLoading"><TableRow v-for="row in 3" :key="row"><TableCell colspan="6"><div class="h-4 w-full animate-pulse rounded bg-muted" /></TableCell></TableRow></TableBody>
            <TableBody v-else-if="assets.length"><TableRow v-for="asset in assets" :key="asset.id"><TableCell><span class="block max-w-56 truncate font-medium">{{ asset.name }}</span><span class="text-xs text-muted-foreground">{{ asset.mime_type }}</span></TableCell><TableCell class="text-muted-foreground">{{ asset.user_id }}</TableCell><TableCell><Badge variant="secondary">{{ sourceLabel(asset.source) }}</Badge></TableCell><TableCell class="text-muted-foreground">{{ formatSize(asset.size_bytes) }}</TableCell><TableCell class="text-muted-foreground">{{ formatTime(asset.created_at) }}</TableCell><TableCell class="text-right"><div class="flex justify-end gap-2"><Button v-if="asset.download_url" as="a" :href="asset.download_url" target="_blank" rel="noopener noreferrer" variant="outline" size="sm">下载</Button><Button variant="ghost" size="icon-sm" :disabled="deletingId !== null" aria-label="删除素材" @click="removeAsset(asset)"><Spinner v-if="deletingId === asset.id" /><Trash2Icon v-else /></Button></div></TableCell></TableRow></TableBody>
            <TableBody v-else><TableRow><TableCell colspan="6" class="py-8 text-center text-muted-foreground">还没有素材对象</TableCell></TableRow></TableBody>
          </Table>
        </div>
      </section>
    </div>
  </AdminLayout>
</template>
