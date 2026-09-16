import type { ScienceGeneratedQuiz } from '../api/types'
import type { PracticeQuestion } from './questionTypes'

export function quizToPractice(question: ScienceGeneratedQuiz): PracticeQuestion {
  const options = question.options.map((option) => option.label ?? '')
  const optionIds = question.options.map((option) => String(option.id))
  const answerId = optionIds[question.answerIndex] ?? String(question.answerIndex)
  return {
    id: question.instanceId,
    kind: 'choice',
    prompt: question.stem,
    visual: question.visual.emoji || question.visual.text || '',
    icon: question.visual.kind === 'icon' ? (question.visual.text || question.visual.key) : undefined,
    options,
    optionIds,
    answerId,
    correct: question.answerIndex,
    success: '找对了！',
    tip: '再观察一下。',
  }
}
