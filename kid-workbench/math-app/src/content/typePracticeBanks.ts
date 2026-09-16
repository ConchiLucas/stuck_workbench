export const practiceTypes = ['equation', 'story', 'missing', 'judge', 'shape'] as const
export type PracticeType = (typeof practiceTypes)[number]

export type PracticeQuestion = {
  id: string
  prompt: string
  visual: string
  options: string[]
  answerIndex: number
}

export const typeTitles: Record<PracticeType, string> = {
  equation: '看算式选答案',
  story: '看数量图选答案',
  missing: '补全算式',
  judge: '判断算式对错',
  shape: '认识图形',
}

export const typePracticeBanks: Record<PracticeType, PracticeQuestion[]> = {
  equation: [
    { id: 'eq-1', prompt: '3 + 5 = ?', visual: '3 + 5', options: ['7', '8', '9', '6'], answerIndex: 1 },
    { id: 'eq-2', prompt: '9 − 4 = ?', visual: '9 − 4', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'eq-3', prompt: '6 + 7 = ?', visual: '6 + 7', options: ['12', '13', '14', '11'], answerIndex: 1 },
    { id: 'eq-4', prompt: '12 − 5 = ?', visual: '12 − 5', options: ['6', '7', '8', '9'], answerIndex: 1 },
  ],
  story: [
    { id: 'st-1', prompt: '一共有几颗星星？', visual: '★★★  ★★', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'st-2', prompt: '8 个拿走 3 个，还剩几个？', visual: '●●●●●●●●  −3', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'st-3', prompt: '2 个苹果再放上 2 个，一共几个？', visual: '🍎🍎  🍎🍎', options: ['3', '4', '5', '6'], answerIndex: 1 },
    { id: 'st-4', prompt: '5 只小鸟飞走 1 只，还剩几只？', visual: '🐦🐦🐦🐦🐦  −1', options: ['3', '4', '5', '6'], answerIndex: 1 },
  ],
  missing: [
    { id: 'mi-1', prompt: '3 + □ = 8', visual: '3 + □ = 8', options: ['4', '5', '6', '7'], answerIndex: 1 },
    { id: 'mi-2', prompt: '12 − □ = 7', visual: '12 − □ = 7', options: ['3', '4', '5', '6'], answerIndex: 2 },
    { id: 'mi-3', prompt: '□ + 6 = 10', visual: '□ + 6 = 10', options: ['3', '4', '5', '6'], answerIndex: 1 },
    { id: 'mi-4', prompt: '9 − □ = 2', visual: '9 − □ = 2', options: ['5', '6', '7', '8'], answerIndex: 2 },
  ],
  judge: [
    { id: 'ju-1', prompt: '7 + 6 = 12，对吗？', visual: '7 + 6 = 12', options: ['对', '错'], answerIndex: 1 },
    { id: 'ju-2', prompt: '8 + 8 = 16，对吗？', visual: '8 + 8 = 16', options: ['对', '错'], answerIndex: 0 },
    { id: 'ju-3', prompt: '15 − 6 = 9，对吗？', visual: '15 − 6 = 9', options: ['对', '错'], answerIndex: 0 },
    { id: 'ju-4', prompt: '11 − 3 = 7，对吗？', visual: '11 − 3 = 7', options: ['对', '错'], answerIndex: 1 },
  ],
  shape: [
    { id: 'sh-1', prompt: '这是什么图形？', visual: '△', options: ['圆形', '三角形', '正方形', '长方形'], answerIndex: 1 },
    { id: 'sh-2', prompt: '听到：圆形。哪一个是？', visual: '👂  ○', options: ['○', '△', '□', '◇'], answerIndex: 0 },
    { id: 'sh-3', prompt: '找出没有角的图形', visual: '？', options: ['○', '△', '□', '◇'], answerIndex: 0 },
    { id: 'sh-4', prompt: '这是什么图形？', visual: '□', options: ['圆形', '三角形', '正方形', '长方形'], answerIndex: 2 },
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
