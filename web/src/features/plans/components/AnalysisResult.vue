<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ result: unknown }>()

interface Section {
  title: string
  value: unknown
}

const KNOWN_SECTIONS: Array<{ title: string; keys: string[] }> = [
  { title: '可行性', keys: ['feasibility', 'verdict', 'feasibility_assessment', '可行性'] },
  { title: '市场', keys: ['market', 'market_analysis', '市场'] },
  { title: '竞争', keys: ['competition', 'competitive_landscape', 'competitive_analysis', '竞争'] },
  { title: '风险', keys: ['risk', 'risks', '风险'] },
  { title: '机会', keys: ['opportunity', 'opportunities', '机会'] },
  { title: '建议', keys: ['recommendation', 'recommendations', 'suggestions', 'advice', '建议'] },
]

const IGNORED_KEYS = new Set(['plan_id'])

const sections = computed<Section[]>(() => {
  const result = props.result
  if (!result || typeof result !== 'object' || Array.isArray(result)) return []
  const record = result as Record<string, unknown>
  const picked: Section[] = []
  const used = new Set<string>()

  for (const section of KNOWN_SECTIONS) {
    for (const key of section.keys) {
      if (key in record && !IGNORED_KEYS.has(key)) {
        picked.push({ title: section.title, value: record[key] })
        used.add(key)
        break
      }
    }
  }
  for (const [key, value] of Object.entries(record)) {
    if (!used.has(key) && !IGNORED_KEYS.has(key)) {
      picked.push({ title: key, value })
    }
  }
  return picked
})

const fallbackText = computed(() => {
  const result = props.result
  if (typeof result === 'string') return result
  if (Array.isArray(result)) return ''
  if (result && typeof result === 'object' && sections.value.length === 0) {
    return JSON.stringify(result, null, 2)
  }
  return ''
})

function isRenderableObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function itemText(item: unknown): string {
  if (isRenderableObject(item)) {
    return Object.entries(item)
      .map(([key, value]) => `${key}: ${scalarText(value)}`)
      .join('；')
  }
  return scalarText(item)
}

function scalarText(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <section v-for="section in sections" :key="section.title" class="flex flex-col gap-2">
      <h3 class="text-sm font-semibold tracking-tight">{{ section.title }}</h3>
      <p v-if="typeof section.value === 'string'" class="text-sm leading-7 text-muted-foreground">
        {{ section.value }}
      </p>
      <ul v-else-if="Array.isArray(section.value)" class="list-disc space-y-1 pl-5 text-sm leading-6 text-muted-foreground">
        <li v-for="(item, index) in section.value" :key="index">{{ itemText(item) }}</li>
      </ul>
      <dl v-else-if="isRenderableObject(section.value)" class="flex flex-col gap-1.5">
        <div v-for="(value, key) in section.value" :key="key" class="text-sm leading-6">
          <dt class="inline font-medium">{{ key }}：</dt>
          <dd class="inline text-muted-foreground">{{ itemText(value) }}</dd>
        </div>
      </dl>
      <p v-else class="text-sm leading-7 text-muted-foreground">{{ scalarText(section.value) }}</p>
    </section>

    <ul v-if="Array.isArray(result)" class="list-disc space-y-1 pl-5 text-sm leading-6 text-muted-foreground">
      <li v-for="(item, index) in result" :key="index">{{ itemText(item) }}</li>
    </ul>
    <pre v-else-if="fallbackText" class="overflow-x-auto rounded-lg bg-muted p-4 text-xs leading-6">{{ fallbackText }}</pre>
  </div>
</template>
