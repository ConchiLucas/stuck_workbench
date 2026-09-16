import { glyphCaption, glyphWeight } from './logicNames'

export const practiceTypes = ['pattern', 'classify', 'order', 'shape_reason', 'diff', 'compare'] as const
export type PracticeType = (typeof practiceTypes)[number]
export type PlayMode = 'choice' | 'order' | 'classify'

export type PracticeOption = {
  id: string
  glyph: string
  caption: string
  weight: number
}

export type PracticeQuestion = {
  id: string
  prompt: string
  items: string[]
  options: PracticeOption[]
  answerIndex: number
  emojiOptions: boolean
  play: PlayMode
}

export const typeTitles: Record<PracticeType, string> = {
  pattern: '找规律',
  classify: '分类',
  order: '排序',
  shape_reason: '图形推理',
  diff: '找不同',
  compare: '比较',
}

export const typeHints: Partial<Record<PracticeType, string>> = {
  classify: '点出不一样的那个',
  diff: '找出不一样的那个',
}

export function asTiles(glyphs: string[]): PracticeOption[] {
  return glyphs.map((glyph) => ({
    id: glyph,
    glyph,
    caption: glyphCaption(glyph),
    weight: glyphWeight(glyph),
  }))
}

export const typePracticeBanks: Record<PracticeType, PracticeQuestion[]> = {
  pattern: [
    { id: 'pt-1', prompt: '下一个是哪个？', items: ['🔴', '🔵', '🔴', '🔵'], options: asTiles(['🟢', '🟡', '🔴', '⬛']), answerIndex: 2, emojiOptions: true, play: 'choice' },
    { id: 'pt-2', prompt: '下一个是哪个？', items: ['⬛', '⬜', '⬛', '⬜'], options: asTiles(['⬜', '🔺', '⬛', '🔵']), answerIndex: 2, emojiOptions: true, play: 'choice' },
    { id: 'pt-3', prompt: '下一个是哪个？', items: ['🍎', '🍎', '🍌', '🍌'], options: asTiles(['🍎', '🍌', '🍇', '🍊']), answerIndex: 1, emojiOptions: true, play: 'choice' },
    { id: 'pt-4', prompt: '接下来是几？', items: ['1️⃣', '2️⃣', '3️⃣', '4️⃣'], options: asTiles(['6️⃣', '5️⃣', '3️⃣', '0️⃣']), answerIndex: 1, emojiOptions: false, play: 'choice' },
  ],
  classify: [
    { id: 'cl-1', prompt: '哪个不是水果？', items: [], options: asTiles(['🍎', '🍌', '🍊', '🚗']), answerIndex: 3, emojiOptions: true, play: 'classify' },
    { id: 'cl-2', prompt: '哪个不会飞？', items: [], options: asTiles(['🐦', '🦋', '✈️', '🐕']), answerIndex: 3, emojiOptions: true, play: 'classify' },
    { id: 'cl-3', prompt: '哪个不是动物？', items: [], options: asTiles(['🐱', '🐶', '🐰', '🌳']), answerIndex: 3, emojiOptions: true, play: 'classify' },
    { id: 'cl-4', prompt: '哪个不能吃？', items: [], options: asTiles(['🍞', '🍎', '🧀', '👟']), answerIndex: 3, emojiOptions: true, play: 'classify' },
  ],
  order: [
    { id: 'od-1', prompt: '从小到大怎么排？', items: ['1️⃣', '2️⃣', '3️⃣'], options: asTiles(['3️⃣', '1️⃣', '2️⃣']), answerIndex: 0, emojiOptions: true, play: 'order' },
    { id: 'od-2', prompt: '一天的正确顺序是？', items: ['🌅', '☀️', '🌙'], options: asTiles(['🌙', '🌅', '☀️']), answerIndex: 0, emojiOptions: true, play: 'order' },
    { id: 'od-3', prompt: '小树长大的顺序是？', items: ['🌱', '🌿', '🌳'], options: asTiles(['🌳', '🌱', '🌿']), answerIndex: 0, emojiOptions: true, play: 'order' },
    { id: 'od-4', prompt: '小鸡出生的顺序是？', items: ['🥚', '🐣', '🐥'], options: asTiles(['🐥', '🥚', '🐣']), answerIndex: 0, emojiOptions: true, play: 'order' },
  ],
  shape_reason: [
    { id: 'sr-1', prompt: '下一个是哪个？', items: ['🔵', '⬛', '🔵', '⬛'], options: asTiles(['🔺', '⭐', '🔵', '⬛']), answerIndex: 2, emojiOptions: true, play: 'choice' },
    { id: 'sr-2', prompt: '三角形越来越多，下一个？', items: ['🔺', '🔺🔺', '🔺🔺🔺'], options: asTiles(['🔺', '🔺🔺🔺🔺', '⬛', '🔵']), answerIndex: 1, emojiOptions: false, play: 'choice' },
    { id: 'sr-3', prompt: '点子越来越多，下一个？', items: ['•', '••', '•••'], options: asTiles(['•', '••', '••••', '•••••']), answerIndex: 2, emojiOptions: false, play: 'choice' },
    { id: 'sr-4', prompt: '下一个是哪个？', items: ['⭐', '⬛', '⭐', '⬛'], options: asTiles(['⬛', '⭐', '🔵', '❤️']), answerIndex: 1, emojiOptions: true, play: 'choice' },
  ],
  diff: [
    { id: 'df-1', prompt: '哪个和其他不一样？', items: [], options: asTiles(['🍎', '🍌', '🍊', '🍐']), answerIndex: 3, emojiOptions: true, play: 'choice' },
    { id: 'df-2', prompt: '哪个表情不一样？', items: [], options: asTiles(['😊', '😄', '😁', '😢']), answerIndex: 3, emojiOptions: true, play: 'choice' },
    { id: 'df-3', prompt: '哪个形状不一样？', items: [], options: asTiles(['⬛', '⬜', '◼️', '🔵']), answerIndex: 3, emojiOptions: true, play: 'choice' },
    { id: 'df-4', prompt: '哪个方向不一样？', items: [], options: asTiles(['👉', '👆', '👇', '👈']), answerIndex: 3, emojiOptions: true, play: 'choice' },
  ],
  compare: [
    { id: 'cm-1', prompt: '哪个更高？', items: [], options: asTiles(['🐭', '🦒', '🐜', '🐣']), answerIndex: 1, emojiOptions: true, play: 'choice' },
    { id: 'cm-2', prompt: '哪个更大？', items: [], options: asTiles(['🐱', '🐰', '🐘', '🐦']), answerIndex: 2, emojiOptions: true, play: 'choice' },
    { id: 'cm-3', prompt: '哪个更快？', items: [], options: asTiles(['🐢', '🐌', '🚀', '🚶']), answerIndex: 2, emojiOptions: true, play: 'choice' },
    { id: 'cm-4', prompt: '哪个更重？', items: [], options: asTiles(['🪶', '🎈', '🐘', '🍃']), answerIndex: 2, emojiOptions: true, play: 'choice' },
  ],
}

export function isPracticeType(value: string | undefined): value is PracticeType {
  return practiceTypes.includes(value as PracticeType)
}

export function practiceHref(type: PracticeType, n = 1) {
  return n <= 1 ? `/practice/type/${type}` : `/practice/type/${type}/${n}`
}

export function practiceResultHref(type: PracticeType) {
  return `/practice/type/${type}/result`
}

export function shuffleTiles(items: string[], seed: string) {
  const copy = [...items]
  let hash = 0
  for (const char of seed) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  for (let i = copy.length - 1; i > 0; i -= 1) {
    hash = Math.imul(hash, 1664525) + 1013904223 >>> 0
    const j = hash % (i + 1)
    ;[copy[i], copy[j]] = [copy[j], copy[i]]
  }
  return copy
}
