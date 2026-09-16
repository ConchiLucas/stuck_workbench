import type { ScienceExample, ScienceKind } from '@kid-workbench/science-player'

export type QuestionKind = ScienceKind
export type MatchNode = { id: string; label: string; visual?: string; image?: string; icon?: string }

export type QuestionType = {
  slug: QuestionKind
  mark: string
  title: string
  action: string
  description: string
  examples: string[]
}

export type PracticeQuestion = {
  id: string
  kind: QuestionKind
  prompt: string
  visual: string
  icon?: string
  options: string[]
  optionIds?: string[]
  answerId?: string
  correct: number
  success: string
  tip: string
  listLabel?: string
  steps?: string[]
  correctSequence?: string
  correctSequenceIds?: string[]
  sequenceItems?: MatchNode[]
  sequenceDisplayOrder?: string[]
  sequenceStartId?: string
  sequenceLoop?: boolean
  stepIcons?: Record<string, string>
  matchSources?: MatchNode[]
  matchTargets?: MatchNode[]
  matchAnswers?: Record<string, string>
  matchSourceGroup?: string
  matchTargetGroup?: string
  labelToken?: string
  labelCorrect?: string
  labelTargets?: { id: string; label: string; x: number; y: number }[]
  labelTokens?: MatchNode[]
  labelAnswers?: Record<string, string>
  diagramKey?: string
  diagramVersion?: number
}

export const questionTypes: QuestionType[] = [
  { slug: 'choice', mark: '选', title: '选择题', action: '观察后选出答案', description: '从多个候选答案中辨认最符合题意的一项，适合检验事实识记与观察判断。', examples: ['单项选择', '图片选择', '正误判断', '情境选择'] },
  { slug: 'match', mark: '连', title: '连线题', action: '把关系连接起来', description: '把事物与它的环境、功能或用途建立一一对应关系。', examples: ['动物与栖息地', '器官与功能', '材料与用途', '星球与特征'] },
  { slug: 'sequence', mark: '排', title: '排序题', action: '还原变化过程', description: '理解事物随时间变化的先后关系，建立完整过程概念。', examples: ['生命周期', '水循环', '植物生长', '实验步骤'] },
  { slug: 'label', mark: '标', title: '结构标注题', action: '把名称放到正确位置', description: '在结构图上定位部位，再把名称或功能标到对应位置。', examples: ['身体器官', '植物结构', '昆虫身体', '星球位置'] },
]

const plantTargets = [
  { id: 'flower', label: '花', x: 0.78, y: 0.14 },
  { id: 'leaf', label: '叶', x: 0.18, y: 0.48 },
  { id: 'root', label: '根', x: 0.50, y: 0.90 },
]

export const practiceQuestions: PracticeQuestion[] = [
  { id: 'duck-feet', kind: 'choice', prompt: '哪种动物的脚掌适合在水里游泳？', visual: '', icon: 'duck', options: ['猫', '鸭子', '兔子'], optionIds: ['cat', 'duck', 'rabbit'], answerId: 'duck', correct: 1, success: '找对了！鸭子的脚趾间有蹼，划水更有力。', tip: '先看脚趾之间，有没有像小扇子一样连起来。' },
  { id: 'plant-sun', kind: 'choice', prompt: '在大自然里，绿叶制造养分主要靠哪一种光？', visual: '', icon: 'sun', options: ['灯光', '太阳', '萤火虫'], optionIds: ['lamp', 'sun', 'firefly'], answerId: 'sun', correct: 1, success: '找对了！太阳光帮植物制造养分，才能慢慢长高。', tip: '想想叶子白天总是朝着哪里。' },
  { id: 'ice-heat', kind: 'choice', prompt: '冰变成水，通常是因为什么？', visual: '', icon: 'ice', options: ['变热了', '变冷了', '变重了'], optionIds: ['heat', 'cold', 'heavy'], answerId: 'heat', correct: 0, success: '找对了！冰遇热就会融化成水。', tip: '把冰块放在手上，会慢慢变成什么？' },
  { id: 'fish-gills', kind: 'choice', prompt: '哪种动物用鳃在水里呼吸？', visual: '', icon: 'fish', options: ['猫', '金鱼', '兔子'], optionIds: ['cat', 'goldfish', 'rabbit'], answerId: 'goldfish', correct: 1, success: '找对了！金鱼用鳃在水里呼吸。', tip: '看看谁一直生活在水里。' },

  { id: 'animal-homes', kind: 'match', prompt: '把动物和它们的生活环境连在一起。', visual: '', options: [], correct: 0, success: '连对了！动物要住在适合自己的地方。', tip: '看看皮毛、皮肤和脚掌，再想它们适合哪里。', matchSourceGroup: '动物', matchTargetGroup: '环境', matchSources: [{ id: 'frog', label: '青蛙', icon: 'frog' }, { id: 'bear', label: '北极熊', icon: 'bear' }, { id: 'camel', label: '骆驼', icon: 'camel' }], matchTargets: [{ id: 'forest', label: '热带雨林', image: '/habitats/habitat-rainforest.png' }, { id: 'ice', label: '冰原', image: '/habitats/habitat-ice.png' }, { id: 'desert', label: '沙漠', image: '/habitats/habitat-desert.png' }], matchAnswers: { frog: 'forest', bear: 'ice', camel: 'desert' } },
  { id: 'organ-jobs', kind: 'match', prompt: '把器官和它们的本领连在一起。', visual: '', options: [], correct: 0, success: '连对了！眼睛用来看，耳朵用来听，鼻子用来闻。', tip: '先想这个器官帮我们做什么。', matchSourceGroup: '器官', matchTargetGroup: '本领', matchSources: [{ id: 'ear', label: '耳朵', icon: 'ear' }, { id: 'eye', label: '眼睛', icon: 'eye' }, { id: 'nose', label: '鼻子', icon: 'nose' }], matchTargets: [{ id: 'see', label: '看', icon: 'see' }, { id: 'hear', label: '听', icon: 'hear' }, { id: 'smell', label: '闻', icon: 'smell' }], matchAnswers: { eye: 'see', ear: 'hear', nose: 'smell' } },
  { id: 'who-eats-what', kind: 'match', prompt: '把下面三种动物和它们最爱吃的食物连起来。', visual: '', options: [], correct: 0, success: '连对了！熊猫吃竹子，蜜蜂采花蜜，长颈鹿吃树叶。', tip: '想想它们平时用什么喂饱自己。', matchSourceGroup: '动物', matchTargetGroup: '食物', matchSources: [{ id: 'bee', label: '蜜蜂', icon: 'bee' }, { id: 'panda', label: '熊猫', icon: 'panda' }, { id: 'giraffe', label: '长颈鹿', icon: 'giraffe' }], matchTargets: [{ id: 'bamboo', label: '竹子', icon: 'bamboo' }, { id: 'nectar', label: '花蜜', icon: 'nectar' }, { id: 'leaves', label: '树叶', icon: 'leaf' }], matchAnswers: { panda: 'bamboo', bee: 'nectar', giraffe: 'leaves' } },

  { id: 'plant-grow', kind: 'sequence', prompt: '把植物的生长过程排成正确顺序。', visual: '', options: [], correct: 0, success: '顺序正确！种子吸水后先发芽，再长成幼苗。', tip: '', listLabel: '植物生长排序', sequenceItems: [{ id: 'sprout', label: '发芽', icon: 'sprout' }, { id: 'seed', label: '种子', icon: 'seed' }, { id: 'seedling', label: '幼苗', icon: 'seedling' }], sequenceDisplayOrder: ['sprout', 'seed', 'seedling'], correctSequenceIds: ['seed', 'sprout', 'seedling'], sequenceStartId: 'seed', steps: ['发芽', '种子', '幼苗'], correctSequence: '种子发芽幼苗' },
  { id: 'butterfly-grow', kind: 'sequence', prompt: '把蝴蝶从卵到成虫的主要阶段排好（本题不包含蛹）。', visual: '', options: [], correct: 0, success: '顺序正确！卵先变成毛毛虫，再变成蝴蝶。', tip: '', listLabel: '蝴蝶生长排序', sequenceItems: [{ id: 'caterpillar', label: '毛毛虫', icon: 'caterpillar' }, { id: 'egg', label: '卵', icon: 'egg' }, { id: 'butterfly', label: '蝴蝶', icon: 'butterfly' }], sequenceDisplayOrder: ['caterpillar', 'egg', 'butterfly'], correctSequenceIds: ['egg', 'caterpillar', 'butterfly'], sequenceStartId: 'egg', steps: ['毛毛虫', '卵', '蝴蝶'], correctSequence: '卵毛毛虫蝴蝶' },
  { id: 'water-change', kind: 'sequence', prompt: '冰块受热之后会怎样变化？从固态开始排。', visual: '', options: [], correct: 0, success: '顺序正确！冰遇热先变成水，再变成水汽。', tip: '', listLabel: '水的变化排序', sequenceItems: [{ id: 'water', label: '水', icon: 'water' }, { id: 'ice', label: '冰', icon: 'ice' }, { id: 'vapor', label: '汽', icon: 'vapor' }], sequenceDisplayOrder: ['water', 'ice', 'vapor'], correctSequenceIds: ['ice', 'water', 'vapor'], sequenceStartId: 'ice', steps: ['水', '冰', '汽'], correctSequence: '冰水汽' },

  { id: 'plant-root', kind: 'label', prompt: '把“根”标到植物的正确位置。', visual: '', options: [], correct: 0, success: '标对了！根藏在土壤下，负责吸收水分。', tip: '看一看植物最下面、埋在土里的部分。', labelToken: '根', labelCorrect: 'root', labelTargets: plantTargets, labelTokens: [{ id: 'root', label: '根' }], labelAnswers: { root: 'root' }, diagramKey: 'plant-structure', diagramVersion: 1 },
  { id: 'plant-leaf', kind: 'label', prompt: '把“叶”标到植物的正确位置。', visual: '', options: [], correct: 0, success: '标对了！叶子长在茎上，用来接收阳光。', tip: '看看植物身体两边伸出来的绿色部分。', labelToken: '叶', labelCorrect: 'leaf', labelTargets: plantTargets, labelTokens: [{ id: 'leaf', label: '叶' }], labelAnswers: { leaf: 'leaf' }, diagramKey: 'plant-structure', diagramVersion: 1 },
  { id: 'plant-flower', kind: 'label', prompt: '把“花”标到植物的正确位置。', visual: '', options: [], correct: 0, success: '标对了！花开在最上面，能结出种子。', tip: '看看植物最高、最鲜艳的部分。', labelToken: '花', labelCorrect: 'flower', labelTargets: plantTargets, labelTokens: [{ id: 'flower', label: '花' }], labelAnswers: { flower: 'flower' }, diagramKey: 'plant-structure', diagramVersion: 1 },
]

export function toScienceExample(question: PracticeQuestion): ScienceExample {
  if (question.kind === 'choice') {
    const options = question.options.map((label, index) => ({ id: question.optionIds?.[index] ?? `opt-${index}`, label, icon: undefined as string | undefined }))
    return { kind: 'choice', prompt: question.prompt, visual: question.visual, icon: question.icon, options, answerId: question.answerId ?? options[question.correct]?.id, explanation: question.success.replace(/^.*?！/, ''), tip: question.tip }
  }
  if (question.kind === 'match') {
    return {
      kind: 'match', prompt: question.prompt, explanation: question.success.replace(/^.*?！/, ''), tip: question.tip,
      matchSourceGroup: question.matchSourceGroup, matchTargetGroup: question.matchTargetGroup,
      matchSources: (question.matchSources ?? []).map((item) => ({ id: item.id, label: item.label, icon: item.icon, imageUrl: item.image })),
      matchTargets: (question.matchTargets ?? []).map((item) => ({ id: item.id, label: item.label, icon: item.icon, imageUrl: item.image })),
      matchAnswers: question.matchAnswers,
    }
  }
  if (question.kind === 'sequence') {
    return {
      kind: 'sequence', prompt: question.prompt, explanation: question.success.replace(/^.*?！/, ''), tip: question.tip,
      sequenceItems: question.sequenceItems?.map((item) => ({ id: item.id, label: item.label, icon: item.icon })),
      sequenceDisplayOrder: question.sequenceDisplayOrder,
      correctSequence: question.correctSequenceIds,
      sequenceStartId: question.sequenceStartId,
      sequenceLoop: question.sequenceLoop,
      listLabel: question.listLabel,
    }
  }
  return {
    kind: 'label', prompt: question.prompt, explanation: question.success.replace(/^.*?！/, ''), tip: question.tip,
    diagramKey: question.diagramKey ?? 'plant-structure', diagramVersion: question.diagramVersion ?? 1,
    labelTargets: question.labelTargets, labelTokens: question.labelTokens?.map((item) => ({ id: item.id, label: item.label })),
    labelAnswers: question.labelAnswers,
  }
}

export function findQuestionType(slug?: string) {
  return questionTypes.find((item) => item.slug === slug)
}

export function questionsFor(slug?: string) {
  return practiceQuestions.filter((item) => item.kind === slug)
}

export function questionPath(kind: string, question: PracticeQuestion, bank: PracticeQuestion[] = questionsFor(kind)) {
  const index = bank.findIndex((item) => item.id === question.id)
  if (index <= 0) return `/question-types/${kind}`
  return `/question-types/${kind}/${question.id}`
}
