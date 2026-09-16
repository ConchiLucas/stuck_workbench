import { expect, it } from 'vitest'
import type { LogicGeneratedQuiz } from '../api/types'
import { quizToPractice } from './quizAdapter'

it('maps missing visual to an empty sequence', () => {
  const question = quizToPractice({
    instanceId: 'q1',
    type: 'classify',
    stem: '哪个不是动物？',
    targetId: 9,
    visual: undefined as unknown as LogicGeneratedQuiz['visual'],
    options: [{ id: 0, emoji: '🐱' }, { id: 1, label: '树' }],
    answerIndex: 1,
  })
  expect(question.items).toEqual([])
  expect(question.options).toEqual([
    { id: '0', glyph: '🐱', caption: '猫', weight: 1 },
    { id: '1', glyph: '树', caption: '树', weight: 2 },
  ])
  expect(question.emojiOptions).toBe(false)
})

it('keeps emoji glyph and Chinese caption together', () => {
  const question = quizToPractice({
    instanceId: 'q2',
    type: 'compare',
    stem: '哪个更大？',
    targetId: 3,
    visual: {},
    options: [
      { id: 0, emoji: '🚌', label: '公交车' },
      { id: 1, emoji: '🛼' },
      { id: 2, emoji: '🚲' },
      { id: 3, emoji: '🛴' },
    ],
    answerIndex: 0,
  })
  expect(question.options[0]).toMatchObject({ glyph: '🚌', caption: '公交车', weight: 4 })
  expect(question.options[1].caption).toBe('溜冰鞋')
  expect(question.options[0].weight).toBeGreaterThan(question.options[1].weight ?? 0)
})
