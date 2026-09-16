export interface AdditionEquationQuestion {
  id: string
  stem: string
  a: number
  b: number
  options: [number, number, number, number]
  answerIndex: number
}

export interface AdditionEquationType {
  slug: 'addition-equation'
  title: string
  mode: string
  description: string
  learningGoal: string
  rules: string[]
  example: AdditionEquationQuestion
  questions: AdditionEquationQuestion[]
}

export interface QuestionTypeCard {
  id: string
  order: number
  title: string
  description: string
  example: string
  mark: string
  status: 'supported' | 'expandable'
  href?: string
  accent?: 'arithmetic' | 'shape'
}

export interface QuestionTypePack extends QuestionTypeCard {
  accent: 'arithmetic' | 'shape'
  questionIds: string[]
}

export interface QuestionTypeGroup {
  id: 'addition' | 'subtraction' | 'shape'
  title: string
  description: string
  types: QuestionTypeCard[]
}

export type QuestionTypeExample =
  | { kind: 'choice' | 'missing' | 'audio-shape' | 'shape-name' | 'shape-feature'; prompt: string; options: string[]; answer: string }
  | { kind: 'objects'; prompt: string; groups: string[]; options: string[]; answer: string }
  | { kind: 'judgement'; prompt: string; statements: string[]; answer: string }
  | { kind: 'shape-sort'; prompt: string; buckets: { label: string; items: string[] }[] }

export interface QuestionTypeDetail {
  id: string
  groupId: QuestionTypeGroup['id']
  moduleTitle: string
  title: string
  status: QuestionTypeCard['status']
  learningGoal: string
  rules: [string, string, string]
  example: QuestionTypeExample
  practiceHref?: string
}

const questions: AdditionEquationQuestion[] = [
  { id: 'add-01', stem: '3 + 5 = ?', a: 3, b: 5, options: [7, 8, 9, 6], answerIndex: 1 },
  { id: 'add-02', stem: '6 + 7 = ?', a: 6, b: 7, options: [12, 14, 13, 11], answerIndex: 2 },
  { id: 'add-03', stem: '9 + 4 = ?', a: 9, b: 4, options: [14, 12, 11, 13], answerIndex: 3 },
  { id: 'add-04', stem: '8 + 8 = ?', a: 8, b: 8, options: [15, 16, 17, 14], answerIndex: 1 },
  { id: 'add-05', stem: '7 + 5 = ?', a: 7, b: 5, options: [13, 11, 12, 10], answerIndex: 2 },
]

export const additionEquationType: AdditionEquationType = {
  slug: 'addition-equation',
  title: '20 以内加法',
  mode: '看算式选答案',
  description: '观察加法算式，从四个数字中选出正确答案。',
  learningGoal: '理解两个数合起来的数量关系，熟练计算 20 以内加法。',
  rules: ['先看清两个加数', '在心里算出它们的和', '从四个数字中选出答案'],
  example: questions[0],
  questions,
}

export const questionTypePacks: QuestionTypePack[] = [
  { id: 'equation', order: 1, title: '看算式选答案', description: '观察加减算式，选出正确的得数。', example: '3 + 5 = 8', mark: '3+5', status: 'supported', href: '/practice/type/equation', accent: 'arithmetic', questionIds: ['addition-equation', 'subtraction-equation'] },
  { id: 'story', order: 2, title: '看数量图选答案', description: '看图数一数，算出合起来或还剩多少。', example: '3 个 + 2 个 = 5 个', mark: '●+', status: 'supported', href: '/practice/type/story', accent: 'arithmetic', questionIds: ['addition-story', 'subtraction-story'] },
  { id: 'missing', order: 3, title: '补全算式', description: '补上缺少的加数、减数或得数。', example: '3 + □ = 8', mark: '□', status: 'expandable', href: '/practice/type/missing', accent: 'arithmetic', questionIds: ['addition-missing', 'subtraction-missing'] },
  { id: 'judge', order: 4, title: '判断算式对错', description: '判断算式是否成立，或找出算错的一项。', example: '7 + 6 = 12 ?', mark: '✓?', status: 'expandable', href: '/practice/type/judge', accent: 'arithmetic', questionIds: ['addition-judge', 'subtraction-error'] },
  { id: 'shape', order: 5, title: '认识图形', description: '听名称、看外形、按特征和分类认识图形。', example: '△ → 三角形', mark: '△', status: 'supported', href: '/practice/type/shape', accent: 'shape', questionIds: ['shape-find', 'shape-name', 'shape-feature', 'shape-sort'] },
]

export const questionTypeDetails: QuestionTypeDetail[] = [
  { id: 'addition-equation', groupId: 'addition', moduleTitle: '20 以内加法', title: '看算式选答案', status: 'supported', learningGoal: '理解两个数合起来的数量关系，熟练计算 20 以内加法。', rules: ['先看清两个加数', '在心里算出它们的和', '从四个数字中选出答案'], example: { kind: 'choice', prompt: '3 + 5 = ?', options: ['7', '8', '9', '6'], answer: '8' }, practiceHref: '/types/addition-equation/practice' },
  { id: 'addition-story', groupId: 'addition', moduleTitle: '20 以内加法', title: '看数量图选答案', status: 'supported', learningGoal: '把两组具体物品与加法算式联系起来，理解“合起来”的含义。', rules: ['分别数清两组物品', '想一想合起来有多少个', '从数字中选出总数'], example: { kind: 'objects', prompt: '一共有几颗星星？', groups: ['★★★', '★★'], options: ['4', '5', '6', '7'], answer: '5' } },
  { id: 'addition-missing', groupId: 'addition', moduleTitle: '20 以内加法', title: '补全算式', status: 'expandable', learningGoal: '根据加法关系找出缺少的加数或和，建立数之间的联系。', rules: ['先看空格在算式中的位置', '用已知数字推算缺少的数', '把答案放回算式检查'], example: { kind: 'missing', prompt: '3 + □ = 8', options: ['4', '5', '6', '7'], answer: '5' } },
  { id: 'addition-judge', groupId: 'addition', moduleTitle: '20 以内加法', title: '判断算式对错', status: 'expandable', learningGoal: '通过重新计算检验加法算式，形成主动检查答案的习惯。', rules: ['先独立算出正确的和', '和题目右边的数字比较', '判断算式正确还是错误'], example: { kind: 'judgement', prompt: '判断下面的算式', statements: ['7 + 6 = 12'], answer: '错误，7 + 6 = 13' } },
  { id: 'subtraction-equation', groupId: 'subtraction', moduleTitle: '20 以内减法', title: '看算式选答案', status: 'supported', learningGoal: '理解从一个数中去掉一部分的数量关系，熟练计算 20 以内减法。', rules: ['看清被减数和减数', '在心里算出还剩多少', '从四个数字中选出差'], example: { kind: 'choice', prompt: '9 − 4 = ?', options: ['4', '5', '6', '7'], answer: '5' } },
  { id: 'subtraction-story', groupId: 'subtraction', moduleTitle: '20 以内减法', title: '看数量图选答案', status: 'supported', learningGoal: '从物品被拿走的变化中理解减法，找到剩余数量。', rules: ['先数清原来有多少个', '看清拿走了多少个', '数出或算出剩下的数量'], example: { kind: 'objects', prompt: '8 个苹果拿走 3 个，还剩几个？', groups: ['●●●●●', '●●●'], options: ['4', '5', '6', '7'], answer: '5' } },
  { id: 'subtraction-missing', groupId: 'subtraction', moduleTitle: '20 以内减法', title: '补全算式', status: 'expandable', learningGoal: '运用减法各部分之间的关系，推算缺少的数字。', rules: ['确定空格表示哪一部分', '根据已知数字进行推算', '代回原算式检查结果'], example: { kind: 'missing', prompt: '12 − □ = 7', options: ['3', '4', '5', '6'], answer: '5' } },
  { id: 'subtraction-error', groupId: 'subtraction', moduleTitle: '20 以内减法', title: '找出错误算式', status: 'expandable', learningGoal: '逐项检查减法算式，发现并说明计算错误。', rules: ['依次计算每一道算式', '把计算结果与等号右边比较', '指出错误项并说出正确结果'], example: { kind: 'judgement', prompt: '找出计算错误的一项', statements: ['11 − 3 = 8', '14 − 6 = 9', '16 − 7 = 9'], answer: '14 − 6 = 9，正确结果是 8' } },
  { id: 'shape-find', groupId: 'shape', moduleTitle: '认识图形', title: '听名称找图形', status: 'supported', learningGoal: '把听到的图形名称和它的外形准确对应起来。', rules: ['认真听清图形名称', '回想它的边、角或曲线', '从候选图形中找到对应项'], example: { kind: 'audio-shape', prompt: '听到：圆形', options: ['○', '△', '□', '◇'], answer: '○' } },
  { id: 'shape-name', groupId: 'shape', moduleTitle: '认识图形', title: '看图选名称', status: 'supported', learningGoal: '观察图形外形并说出或选出正确名称。', rules: ['仔细观察目标图形', '辨认它的边和角', '选择对应的图形名称'], example: { kind: 'shape-name', prompt: '△', options: ['圆形', '三角形', '正方形', '长方形'], answer: '三角形' } },
  { id: 'shape-feature', groupId: 'shape', moduleTitle: '认识图形', title: '按特征找图形', status: 'expandable', learningGoal: '根据边、角和曲线等特征辨认图形，而不只记住名称。', rules: ['读清题目描述的特征', '逐个观察候选图形', '选择符合全部特征的图形'], example: { kind: 'shape-feature', prompt: '找出没有角的图形', options: ['○', '△', '□', '◇'], answer: '○' } },
  { id: 'shape-sort', groupId: 'shape', moduleTitle: '认识图形', title: '图形分类', status: 'expandable', learningGoal: '发现图形之间的共同特征，并按照明确规则进行分类。', rules: ['先观察所有图形的特点', '确定这次分类使用的规则', '把每个图形放进对应小组'], example: { kind: 'shape-sort', prompt: '按“有没有角”分成两组', buckets: [{ label: '没有角', items: ['○', '◯'] }, { label: '有角', items: ['△', '□', '◇'] }] } },
]

export function getQuestionTypeDetail(id: string) {
  return questionTypeDetails.find((detail) => detail.id === id)
}

export function getQuestionTypePack(typeId: string) {
  return questionTypePacks.find((pack) => pack.id === typeId || pack.questionIds.includes(typeId))
}

export function getPackQuestionDetails(pack: QuestionTypePack) {
  return pack.questionIds
    .map((id) => getQuestionTypeDetail(id))
    .filter((detail): detail is QuestionTypeDetail => Boolean(detail))
}
