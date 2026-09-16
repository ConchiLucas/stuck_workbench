import type { LogicGeneratedQuiz } from '../api/types'
import { glyphCaption, glyphWeight } from './logicNames'
import type { LogicExample, LogicKind, LogicObject } from '@kid-workbench/logic-player'
import type { PlayMode, PracticeOption, PracticeQuestion, PracticeType } from './typePracticeBanks'

function isEmojiGlyph(glyph: string) {
  return Boolean(glyph) && !/[\u4e00-\u9fff]/.test(glyph) && !/^[a-z0-9_-]+$/i.test(glyph)
}

function asObject(id: string, glyph: string, caption?: string, scale?: number): LogicObject {
  const label = caption && caption !== glyph ? caption : (isEmojiGlyph(glyph) ? glyph : (glyphCaption(glyph) || glyph))
  return {
    id,
    glyph,
    caption: label,
    scale: scale ?? glyphWeight(glyph),
  }
}

export function quizToExample(question: LogicGeneratedQuiz): LogicExample {
  const raw = question as LogicGeneratedQuiz & { example?: LogicExample }
  if (raw.example?.kind && raw.example.objects?.length) {
    return raw.example
  }
  const kind = question.type as LogicKind
  if (kind === 'order' && (question.visual?.items?.length ?? 0) >= 3) {
    const items = question.visual.items ?? []
    const objects = items.map((item, index) => asObject(`item-${index}`, item))
    const ids = objects.map((o) => o.id)
    return {
      kind,
      prompt: question.stem,
      rule: { type: 'order', dimension: 'given', direction: 'asc', explain: question.stem },
      objects,
      correctSequence: ids,
      displayOrder: ids,
    }
  }
  const objects = question.options.map((option, index) => {
    const glyph = option.emoji || option.label || String(index)
    return asObject(String(option.id ?? index), glyph, option.label)
  })
  const optionIds = objects.map((o) => o.id)
  const answerId = objects[question.answerIndex]?.id ?? optionIds[0]
  const seq = (question.visual?.items ?? []).map((item, index) => {
    const match = objects.find((o) => o.glyph === item || o.caption === item)
    if (match) return match.id
    const extra = asObject(`seq-${index}`, item)
    objects.push(extra)
    return extra.id
  })
  return {
    kind,
    prompt: question.stem,
    rule: { type: kind, dimension: 'given', explain: question.stem },
    objects,
    sequence: seq,
    options: optionIds,
    answerId,
  }
}

export function fallbackExamples(type: PracticeType): LogicExample[] {
  return banks[type]
}

const red = '#e11d48'
const blue = '#2563eb'
const green = '#16a34a'
const yellow = '#eab308'

const banks: Record<PracticeType, LogicExample[]> = {
  pattern: [
    {
      kind: 'pattern', prompt: '下一个是哪个？',
      rule: { type: 'AB', period: 2, dimension: 'color', explain: '红蓝交替' },
      objects: [
        asObject('r1', 'circle', '红', 2), asObject('b1', 'circle', '蓝', 2),
        asObject('r2', 'circle', '红', 2), asObject('b2', 'circle', '蓝', 2),
        asObject('g1', 'circle', '绿', 2), asObject('y1', 'circle', '黄', 2),
        asObject('k1', 'square', '黑', 2),
      ].map((o, i) => ({ ...o, fill: [red, blue, red, blue, green, yellow, '#111'][i] })),
      sequence: ['r1', 'b1', 'r2', 'b2'], options: ['g1', 'y1', 'r1', 'k1'], answerId: 'r1',
    },
    {
      kind: 'pattern', prompt: '下一个是哪个？',
      rule: { type: 'AB', period: 2, dimension: 'fill', explain: '黑白交替' },
      objects: [
        { id: 'k', caption: '黑', glyph: 'square', fill: '#111', scale: 2 },
        { id: 'w', caption: '白', glyph: 'square', fill: '#f8fafc', scale: 2 },
        { id: 't', caption: '三角', glyph: 'triangle', fill: blue, scale: 2 },
        { id: 'c', caption: '蓝', glyph: 'circle', fill: blue, scale: 2 },
      ],
      sequence: ['k', 'w', 'k', 'w'], options: ['w', 't', 'k', 'c'], answerId: 'k',
    },
    {
      kind: 'pattern', prompt: '下一个是哪个？',
      rule: { type: 'AABB', period: 2, dimension: 'kind', explain: '两个苹果后两个香蕉，下一格补香蕉' },
      objects: [
        { id: 'a1', caption: '苹果', glyph: 'apple', fill: red, scale: 2, attrs: { kind: 'apple' } },
        { id: 'a2', caption: '苹果', glyph: 'apple', fill: red, scale: 2, attrs: { kind: 'apple' } },
        { id: 'b1', caption: '香蕉', glyph: 'banana', fill: yellow, scale: 2, attrs: { kind: 'banana' } },
        { id: 'b2', caption: '香蕉', glyph: 'banana', fill: yellow, scale: 2, attrs: { kind: 'banana' } },
        { id: 'g', caption: '葡萄', glyph: 'grape', fill: '#7c3aed', scale: 2, attrs: { kind: 'grape' } },
        { id: 'o', caption: '橙', glyph: 'circle', fill: '#f97316', scale: 2, attrs: { kind: 'orange' } },
      ],
      sequence: ['a1', 'a2', 'b1'], options: ['a1', 'b2', 'g', 'o'], answerId: 'b2',
    },
    {
      kind: 'pattern', prompt: '接下来是几？',
      rule: { type: 'count', dimension: 'n', explain: '1 到 5' },
      objects: [
        { id: 'n1', caption: '1', glyph: 'one', fill: blue, scale: 2 },
        { id: 'n2', caption: '2', glyph: 'two', fill: blue, scale: 2 },
        { id: 'n3', caption: '3', glyph: 'three', fill: blue, scale: 2 },
        { id: 'n4', caption: '4', glyph: 'four', fill: blue, scale: 2 },
        { id: 'n5', caption: '5', glyph: 'circle', fill: blue, scale: 2, attrs: { n: '5' } },
        { id: 'n6', caption: '6', glyph: 'star', fill: blue, scale: 2 },
        { id: 'n0', caption: '0', glyph: 'circle', fill: '#94a3b8', scale: 1 },
      ],
      sequence: ['n1', 'n2', 'n3', 'n4'], options: ['n6', 'n5', 'n3', 'n0'], answerId: 'n5',
    },
  ],
  classify: [
    {
      kind: 'classify', prompt: '哪个不是水果？',
      rule: { type: 'odd-one-out', dimension: 'kind', inGroup: 'fruit', explain: '水果' },
      objects: [
        { id: 'a', caption: '苹果', glyph: 'apple', fill: red, scale: 2, attrs: { category: 'fruit' } },
        { id: 'b', caption: '香蕉', glyph: 'banana', fill: yellow, scale: 2, attrs: { category: 'fruit' } },
        { id: 'o', caption: '橙', glyph: 'circle', fill: '#f97316', scale: 2, attrs: { category: 'fruit' } },
        { id: 'c', caption: '汽车', glyph: 'car', fill: red, scale: 2, attrs: { category: 'vehicle' } },
      ],
      options: ['a', 'b', 'o', 'c'], answerId: 'c',
    },
    {
      kind: 'classify', prompt: '哪个不会飞？',
      rule: { type: 'odd-one-out', dimension: 'fly', inGroup: 'fly', explain: '会飞' },
      objects: [
        { id: 'bird', caption: '鸟', glyph: 'bird', fill: '#0ea5e9', scale: 2, attrs: { category: 'fly' } },
        { id: 'fly', caption: '蝴蝶', glyph: 'flower', fill: '#ec4899', scale: 2, attrs: { category: 'fly' } },
        { id: 'plane', caption: '飞机', glyph: 'arrow', fill: '#64748b', scale: 2, attrs: { category: 'fly' } },
        { id: 'dog', caption: '狗', glyph: 'dog', fill: '#92400e', scale: 2, attrs: { category: 'walk' } },
      ],
      options: ['bird', 'fly', 'plane', 'dog'], answerId: 'dog',
    },
    {
      kind: 'classify', prompt: '哪个不是动物？',
      rule: { type: 'odd-one-out', dimension: 'kingdom', inGroup: 'animal', explain: '动物' },
      objects: [
        { id: 'cat', caption: '猫', glyph: 'cat', fill: '#f59e0b', scale: 2, attrs: { category: 'animal' } },
        { id: 'dog', caption: '狗', glyph: 'dog', fill: '#92400e', scale: 2, attrs: { category: 'animal' } },
        { id: 'rabbit', caption: '兔', glyph: 'rabbit', fill: '#a3a3a3', scale: 2, attrs: { category: 'animal' } },
        { id: 'tree', caption: '树', glyph: 'tree', fill: green, scale: 2, attrs: { category: 'plant' } },
      ],
      options: ['cat', 'dog', 'rabbit', 'tree'], answerId: 'tree',
    },
    {
      kind: 'classify', prompt: '哪个不能吃？',
      rule: { type: 'odd-one-out', dimension: 'edible', inGroup: 'food', explain: '食物' },
      objects: [
        { id: 'bread', caption: '面包', glyph: 'square', fill: '#d97706', scale: 2, attrs: { category: 'food' } },
        { id: 'apple', caption: '苹果', glyph: 'apple', fill: red, scale: 2, attrs: { category: 'food' } },
        { id: 'cheese', caption: '奶酪', glyph: 'triangle', fill: yellow, scale: 2, attrs: { category: 'food' } },
        { id: 'shoe', caption: '鞋', glyph: 'skate', fill: '#db2777', scale: 2, attrs: { category: 'wear' } },
      ],
      options: ['bread', 'apple', 'cheese', 'shoe'], answerId: 'shoe',
    },
  ],
  order: [
    {
      kind: 'order', prompt: '从小到大怎么排？',
      rule: { type: 'order', dimension: 'count', direction: 'asc', explain: '1 2 3' },
      objects: [
        { id: 'o1', caption: '1', glyph: 'one', fill: blue, scale: 2 },
        { id: 'o2', caption: '2', glyph: 'two', fill: blue, scale: 2 },
        { id: 'o3', caption: '3', glyph: 'three', fill: blue, scale: 2 },
      ],
      correctSequence: ['o1', 'o2', 'o3'], displayOrder: ['o3', 'o1', 'o2'],
    },
    {
      kind: 'order', prompt: '一天的正确顺序是？',
      rule: { type: 'order', dimension: 'time', direction: 'asc', explain: '早晚' },
      objects: [
        { id: 'am', caption: '晨', glyph: 'sun', fill: yellow, scale: 2 },
        { id: 'noon', caption: '午', glyph: 'sun', fill: '#f97316', scale: 2 },
        { id: 'pm', caption: '夜', glyph: 'circle', fill: '#1e3a8a', scale: 2 },
      ],
      correctSequence: ['am', 'noon', 'pm'], displayOrder: ['pm', 'am', 'noon'],
    },
    {
      kind: 'order', prompt: '小树长大的顺序是？',
      rule: { type: 'order', dimension: 'grow', direction: 'asc', explain: '生长' },
      objects: [
        { id: 'seed', caption: '芽', glyph: 'leaf', fill: '#86efac', scale: 1 },
        { id: 'bush', caption: '苗', glyph: 'leaf', fill: green, scale: 2 },
        { id: 'tree', caption: '树', glyph: 'tree', fill: green, scale: 3 },
      ],
      correctSequence: ['seed', 'bush', 'tree'], displayOrder: ['tree', 'seed', 'bush'],
    },
    {
      kind: 'order', prompt: '小鸡出生的顺序是？',
      rule: { type: 'order', dimension: 'hatch', direction: 'asc', explain: '孵化' },
      objects: [
        { id: 'egg', caption: '蛋', glyph: 'circle', fill: '#f8fafc', scale: 2 },
        { id: 'hatch', caption: '出壳', glyph: 'circle', fill: yellow, scale: 2 },
        { id: 'chick', caption: '小鸡', glyph: 'bird', fill: yellow, scale: 2 },
      ],
      correctSequence: ['egg', 'hatch', 'chick'], displayOrder: ['chick', 'egg', 'hatch'],
    },
  ],
  shape_reason: [
    {
      kind: 'shape_reason', prompt: '下一个是哪个？',
      rule: { type: 'AB', dimension: 'shape', explain: '圆方交替' },
      objects: [
        { id: 'c', caption: '圆', glyph: 'circle', fill: blue, scale: 2 },
        { id: 's', caption: '方', glyph: 'square', fill: '#111', scale: 2 },
        { id: 't', caption: '三角', glyph: 'triangle', fill: red, scale: 2 },
        { id: 'st', caption: '星', glyph: 'star', fill: yellow, scale: 2 },
      ],
      sequence: ['c', 's', 'c', 's'], options: ['t', 'st', 'c', 's'], answerId: 'c',
    },
    {
      kind: 'shape_reason', prompt: '三角形越来越多，下一个？',
      rule: { type: 'count', dimension: 'count', explain: '数量加一' },
      objects: [
        { id: 't1', caption: '一个', glyph: 'triangle', fill: red, scale: 2, count: 1 },
        { id: 't2', caption: '两个', glyph: 'triangle', fill: red, scale: 2, count: 2 },
        { id: 't3', caption: '三个', glyph: 'triangle', fill: red, scale: 2, count: 3 },
        { id: 't4', caption: '四个', glyph: 'triangle', fill: red, scale: 2, count: 4 },
        { id: 's', caption: '方', glyph: 'square', fill: '#111', scale: 2 },
        { id: 'c', caption: '圆', glyph: 'circle', fill: blue, scale: 2 },
      ],
      sequence: ['t1', 't2', 't3'], options: ['t1', 't4', 's', 'c'], answerId: 't4',
    },
    {
      kind: 'shape_reason', prompt: '点子越来越多，下一个？',
      rule: { type: 'count', dimension: 'count', explain: '点加一' },
      objects: [
        { id: 'd1', caption: '一个点', glyph: 'circle', fill: blue, scale: 2, count: 1 },
        { id: 'd2', caption: '两个点', glyph: 'circle', fill: blue, scale: 2, count: 2 },
        { id: 'd3', caption: '三个点', glyph: 'circle', fill: blue, scale: 2, count: 3 },
        { id: 'd4', caption: '四个点', glyph: 'circle', fill: blue, scale: 2, count: 4 },
        { id: 'd5', caption: '五个点', glyph: 'star', fill: blue, scale: 2 },
      ],
      sequence: ['d1', 'd2', 'd3'], options: ['d1', 'd2', 'd4', 'd5'], answerId: 'd4',
    },
    {
      kind: 'shape_reason', prompt: '下一个是哪个？',
      rule: { type: 'AB', dimension: 'shape', explain: '星方交替' },
      objects: [
        { id: 'st', caption: '星', glyph: 'star', fill: yellow, scale: 2 },
        { id: 's', caption: '方', glyph: 'square', fill: '#111', scale: 2 },
        { id: 'c', caption: '圆', glyph: 'circle', fill: blue, scale: 2 },
        { id: 'h', caption: '心', glyph: 'heart', fill: red, scale: 2 },
      ],
      sequence: ['st', 's', 'st', 's'], options: ['s', 'st', 'c', 'h'], answerId: 'st',
    },
  ],
  diff: [
    {
      kind: 'diff', prompt: '哪个和其他不一样？',
      rule: { type: 'unique-odd', dimension: 'color', explain: '三红一绿' },
      objects: [
        { id: 'a', caption: '红苹果', glyph: 'apple', fill: red, scale: 2, attrs: { color: 'red' } },
        { id: 'h', caption: '红心', glyph: 'heart', fill: red, scale: 2, attrs: { color: 'red' } },
        { id: 'c', caption: '红圆', glyph: 'circle', fill: red, scale: 2, attrs: { color: 'red' } },
        { id: 'p', caption: '绿梨', glyph: 'pear', fill: green, scale: 2, attrs: { color: 'green' } },
      ],
      options: ['a', 'h', 'c', 'p'], answerId: 'p',
    },
    {
      kind: 'diff', prompt: '哪个表情不一样？',
      rule: { type: 'unique-odd', dimension: 'mood', explain: '三笑一哭' },
      objects: [
        { id: 's1', caption: '笑', glyph: 'circle', fill: yellow, scale: 2, attrs: { mood: 'smile' } },
        { id: 's2', caption: '笑', glyph: 'circle', fill: yellow, scale: 2, attrs: { mood: 'smile' } },
        { id: 's3', caption: '笑', glyph: 'circle', fill: yellow, scale: 2, attrs: { mood: 'smile' } },
        { id: 'cry', caption: '哭', glyph: 'circle', fill: blue, scale: 2, attrs: { mood: 'cry' } },
      ],
      options: ['s1', 's2', 's3', 'cry'], answerId: 'cry',
    },
    {
      kind: 'diff', prompt: '哪个形状不一样？',
      rule: { type: 'unique-odd', dimension: 'shape', explain: '三方一圆' },
      objects: [
        { id: 's1', caption: '方', glyph: 'square', fill: '#111', scale: 2, attrs: { shape: 'square' } },
        { id: 's2', caption: '方', glyph: 'square', fill: '#334155', scale: 2, attrs: { shape: 'square' } },
        { id: 's3', caption: '方', glyph: 'square', fill: '#64748b', scale: 2, attrs: { shape: 'square' } },
        { id: 'c', caption: '圆', glyph: 'circle', fill: blue, scale: 2, attrs: { shape: 'circle' } },
      ],
      options: ['s1', 's2', 's3', 'c'], answerId: 'c',
    },
    {
      kind: 'diff', prompt: '哪个方向不一样？',
      rule: { type: 'unique-odd', dimension: 'direction', explain: '三右一左' },
      objects: [
        { id: 'r1', caption: '向右', glyph: 'arrow', fill: blue, scale: 2, rotate: 0, attrs: { direction: 'right' } },
        { id: 'r2', caption: '向右', glyph: 'arrow', fill: blue, scale: 2, rotate: 0, attrs: { direction: 'right' } },
        { id: 'r3', caption: '向右', glyph: 'arrow', fill: blue, scale: 2, rotate: 0, attrs: { direction: 'right' } },
        { id: 'l', caption: '向左', glyph: 'arrow', fill: blue, scale: 2, rotate: 180, attrs: { direction: 'left' } },
      ],
      options: ['r1', 'r2', 'r3', 'l'], answerId: 'l',
    },
  ],
  compare: [
    {
      kind: 'compare', prompt: '哪个更高？',
      rule: { type: 'compare', dimension: 'visualSize', direction: 'max', explain: '长颈鹿更高' },
      objects: [
        { id: 'mouse', caption: '老鼠', glyph: 'mouse', fill: '#9ca3af', scale: 1, attrs: { visualSize: '1' } },
        { id: 'gir', caption: '长颈鹿', glyph: 'giraffe', fill: '#d97706', scale: 4, attrs: { visualSize: '4' } },
        { id: 'ant', caption: '蚂蚁', glyph: 'circle', fill: '#111', scale: 1, attrs: { visualSize: '1' } },
        { id: 'chick', caption: '小鸡', glyph: 'bird', fill: yellow, scale: 2, attrs: { visualSize: '2' } },
      ],
      options: ['mouse', 'gir', 'ant', 'chick'], answerId: 'gir',
    },
    {
      kind: 'compare', prompt: '哪个更大？',
      rule: { type: 'compare', dimension: 'visualSize', direction: 'max', explain: '公交车更大' },
      objects: [
        { id: 'cat', caption: '猫', glyph: 'cat', fill: '#f59e0b', scale: 2, attrs: { visualSize: '2' } },
        { id: 'rab', caption: '兔', glyph: 'rabbit', fill: '#a3a3a3', scale: 2, attrs: { visualSize: '2' } },
        { id: 'bus', caption: '公交车', glyph: 'bus', fill: blue, scale: 4, attrs: { visualSize: '4' } },
        { id: 'bird', caption: '鸟', glyph: 'bird', fill: '#0ea5e9', scale: 1, attrs: { visualSize: '1' } },
      ],
      options: ['cat', 'rab', 'bus', 'bird'], answerId: 'bus',
    },
    {
      kind: 'compare', prompt: '哪个更快？',
      rule: { type: 'compare', dimension: 'speed', direction: 'max', explain: '汽车更快' },
      objects: [
        { id: 'turtle', caption: '龟', glyph: 'circle', fill: green, scale: 2, attrs: { speed: '1' } },
        { id: 'snail', caption: '蜗牛', glyph: 'circle', fill: '#a3a3a3', scale: 1, attrs: { speed: '1' } },
        { id: 'car', caption: '汽车', glyph: 'car', fill: red, scale: 2, attrs: { speed: '4' } },
        { id: 'walk', caption: '走', glyph: 'skate', fill: '#64748b', scale: 2, attrs: { speed: '2' } },
      ],
      options: ['turtle', 'snail', 'car', 'walk'], answerId: 'car',
    },
    {
      kind: 'compare', prompt: '哪个更重？',
      rule: { type: 'compare', dimension: 'weight', direction: 'max', explain: '公交车更重' },
      objects: [
        { id: 'feather', caption: '羽毛', glyph: 'leaf', fill: '#e2e8f0', scale: 1, attrs: { weight: '1' } },
        { id: 'balloon', caption: '气球', glyph: 'circle', fill: '#f472b6', scale: 2, attrs: { weight: '1' } },
        { id: 'bus', caption: '公交车', glyph: 'bus', fill: blue, scale: 4, attrs: { weight: '4' } },
        { id: 'leaf', caption: '叶子', glyph: 'leaf', fill: green, scale: 1, attrs: { weight: '1' } },
      ],
      options: ['feather', 'balloon', 'bus', 'leaf'], answerId: 'bus',
    },
  ],
}

export function examplePlay(example: LogicExample) {
  return example.kind === 'order' ? 'order' : 'choice'
}

export function quizToPractice(question: LogicGeneratedQuiz): PracticeQuestion {
  const example = quizToExample(question)
  const play: PlayMode = example.kind === 'order' ? 'order' : example.kind === 'classify' ? 'classify' : 'choice'
  const options: PracticeOption[] = (example.kind === 'order' ? example.displayOrder ?? [] : example.options ?? []).map((id) => {
    const object = example.objects.find((item) => item.id === id)!
    return { id: object.id, glyph: object.glyph, caption: object.caption, weight: object.scale || 2 }
  })
  return {
    id: question.instanceId,
    prompt: example.prompt,
    items: example.kind === 'order' ? (example.correctSequence ?? []) : (example.sequence ?? []).map((id) => example.objects.find((item) => item.id === id)?.glyph ?? id),
    options,
    answerIndex: example.kind === 'order' ? 0 : Math.max(0, (example.options ?? []).indexOf(example.answerId ?? '')),
    emojiOptions: options.every((option) => isEmojiGlyph(option.glyph)),
    play,
  }
}
