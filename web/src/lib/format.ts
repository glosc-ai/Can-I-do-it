export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '请求失败，请稍后重试'
}

export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '-'
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB']
  let value = bytes
  let unit = -1
  do {
    value /= 1024
    unit += 1
  } while (value >= 1024 && unit < units.length - 1)
  return `${value.toFixed(1)} ${units[unit]}`
}

export function formatTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const EXTENSION_LABELS: Record<string, string> = {
  pdf: 'PDF',
  doc: 'Word',
  docx: 'Word',
  txt: 'TXT',
  md: 'Markdown',
  png: '图片',
  jpg: '图片',
  jpeg: '图片',
  webp: '图片',
}

export function fileTypeLabel(filename: string, mimeType?: string): string {
  const extension = filename.split('.').pop()?.toLowerCase() ?? ''
  if (EXTENSION_LABELS[extension]) return EXTENSION_LABELS[extension]
  if (mimeType && mimeType.includes('pdf')) return 'PDF'
  if (mimeType && mimeType.includes('word')) return 'Word'
  if (mimeType && mimeType.startsWith('text/')) return 'TXT'
  if (mimeType && mimeType.startsWith('image/')) return '图片'
  return extension ? extension.toUpperCase() : '文件'
}
