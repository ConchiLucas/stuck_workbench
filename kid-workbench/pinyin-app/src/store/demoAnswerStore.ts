import type { PinyinGeneratedQuiz } from '../api/types'

export function optionLabel(question: PinyinGeneratedQuiz, optionId: string) {
  const index = question.options.findIndex(option => option.id === optionId)
  const option = question.options[index]
  if (!option) return '未选'
  if (question.type === 'shape' || question.type === 'blend') {
    return option.speechText || option.label || `读音 ${index + 1}`
  }
  return option.label || '未选'
}

export function questionPrompt(question: PinyinGeneratedQuiz) {
  if (question.type === 'inword') return question.visual.text || question.speechText || ''
  if (question.type === 'shape') return question.visual.text || ''
  if (question.type === 'blend') return `${question.visual.initial} + ${question.visual.final}`
  return question.speechText || '听音'
}
