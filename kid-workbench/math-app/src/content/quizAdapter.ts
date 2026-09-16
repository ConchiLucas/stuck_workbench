import type { MathGeneratedQuiz } from '../api/types'
import type { PracticeQuestion } from './typePracticeBanks'

const shapeGlyph: Record<string, string> = {
  circle: '○', square: '□', rect: '▭', triangle: '△',
  oval: '⬭', trapezoid: '⏢', rhombus: '◇', star: '★',
}

function storyVisual(question: MathGeneratedQuiz) {
  const emoji = question.visual.emoji || '●'
  const left = emoji.repeat(Math.max(0, question.visual.a ?? 0))
  const right = emoji.repeat(Math.max(0, question.visual.b ?? 0))
  if (question.visual.kind === 'sub') return `${left}  −${question.visual.b ?? 0}`
  return `${left}  ${right}`
}

export function isRedundantVisual(prompt: string, visual: string) {
  const compact = (value: string) => value.replace(/\s+/g, '').replace(/[？?]/g, '')
  const p = compact(prompt)
  const v = compact(visual)
  if (!v) return true
  return p === v || p.includes(v) || v.includes(p)
}

export function quizToPractice(question: MathGeneratedQuiz): PracticeQuestion {
  let visual = question.visual.text || question.stem
  if (question.visual.kind === 'shape' && question.visual.text) {
    visual = shapeGlyph[question.visual.text] || question.visual.text
  } else if (question.type === 'story') {
    visual = storyVisual(question)
  } else if (question.visual.a != null && question.visual.b != null && question.type === 'equation') {
    const sign = question.visual.kind === 'sub' ? '−' : '+'
    visual = `${question.visual.a} ${sign} ${question.visual.b}`
  }
  return {
    id: question.instanceId,
    prompt: question.stem,
    visual,
    options: question.options.map((option) => option.label ?? ''),
    answerIndex: question.answerIndex,
  }
}
