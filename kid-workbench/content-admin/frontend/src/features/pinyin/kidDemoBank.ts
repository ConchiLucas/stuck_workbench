import type { PinyinGeneratedQuiz } from '../../api/pinyinTypes'

export const demoTypes = ['listen', 'inword', 'shape', 'blend'] as const
export type DemoType = (typeof demoTypes)[number]

export const questionTypes: Array<{ key: DemoType; title: string }> = [
  { key: 'listen', title: '听音选字母' },
  { key: 'inword', title: '字中找拼音' },
  { key: 'shape', title: '看形认读' },
  { key: 'blend', title: '声韵拼读' },
]

export function typeTitle(type: DemoType) {
  return questionTypes.find((item) => item.key === type)?.title ?? type
}

const stemByType: Record<DemoType, string> = {
  listen: '听一听，选出你听到的拼音',
  inword: '听一听，这个字里藏着哪个拼音？',
  shape: '看一看，选出四线格里拼音的读音',
  blend: '把声母和韵母拼在一起',
}

export function practiceStem(type: string, stem?: string) {
  if (stem && /[\u4e00-\u9fff]/.test(stem)) return stem
  return stemByType[type as DemoType] ?? stem ?? ''
}

export function pickKey(type: DemoType, n: number) {
  return `${type}:${n}`
}

export function optionLabel(question: PinyinGeneratedQuiz, index: number) {
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
