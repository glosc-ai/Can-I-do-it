import { ref } from 'vue'
import { defineStore } from 'pinia'
import { analyzePlan, uploadPlan } from '@/api/plans'

interface IdeaInput {
  idea: string
  context?: string
}

export const useIdeaAnalysisStore = defineStore('idea-analysis', () => {
  const submitting = ref(false)

  async function submit(input: IdeaInput) {
    if (submitting.value) throw new Error('分析正在提交，请稍候。')
    submitting.value = true
    try {
      const idea = input.idea.trim()
      const context = input.context?.trim() || ''
      const title = idea.replace(/\s+/g, ' ').slice(0, 48)
      const source = [
        '【待分析的创业想法】',
        idea,
        context ? `\n\n【补充背景】\n${context}` : '',
        '\n\n【分析要求】\n请围绕这个想法进行完整的市场调查与商业可行性分析。明确区分已知事实、合理假设与需要验证的内容。',
      ].join('')
      const file = new File([source], 'idea-analysis.txt', { type: 'text/plain' })
      const plan = await uploadPlan(file, `想法：${title}`)
      await analyzePlan(plan.id)
      return plan
    } finally {
      submitting.value = false
    }
  }

  return { submitting, submit }
})
