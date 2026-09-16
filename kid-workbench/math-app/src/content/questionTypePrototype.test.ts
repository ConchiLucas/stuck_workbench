import { describe, expect, it } from 'vitest'
import {
  getPackQuestionDetails,
  getQuestionTypeDetail,
  getQuestionTypePack,
  questionTypeDetails,
  questionTypePacks,
} from './questionTypePrototype'

describe('math question type gallery content', () => {
  it('shows four mixed arithmetic packs and one shape pack', () => {
    expect(questionTypePacks.map((pack) => pack.title)).toEqual([
      '看算式选答案',
      '看数量图选答案',
      '补全算式',
      '判断算式对错',
      '认识图形',
    ])
    expect(questionTypePacks.map((pack) => pack.questionIds.length)).toEqual([2, 2, 2, 2, 4])
  })

  it('uses unique pack ids and a balanced capability status', () => {
    expect(new Set(questionTypePacks.map((pack) => pack.id)).size).toBe(5)
    expect(questionTypePacks.filter((pack) => pack.status === 'supported')).toHaveLength(3)
    expect(questionTypePacks.filter((pack) => pack.status === 'expandable')).toHaveLength(2)
  })

  it('links every pack to a four-question practice', () => {
    expect(questionTypePacks.map((pack) => pack.href)).toEqual([
      '/practice/type/equation',
      '/practice/type/story',
      '/practice/type/missing',
      '/practice/type/judge',
      '/practice/type/shape',
    ])
  })

  it('keeps a complete detail for every question inside the packs', () => {
    const questionIds = questionTypePacks.flatMap((pack) => pack.questionIds)
    expect(questionTypeDetails).toHaveLength(12)
    expect(questionIds).toHaveLength(12)
    expect(questionIds.map((id) => getQuestionTypeDetail(id)?.id)).toEqual(questionIds)
    expect(questionTypeDetails.every((detail) => detail.rules.length === 3)).toBe(true)
  })

  it('groups addition and subtraction variants inside the same pack', () => {
    expect(getQuestionTypePack('addition-equation')?.id).toBe('equation')
    expect(getQuestionTypePack('subtraction-equation')?.id).toBe('equation')
    expect(getPackQuestionDetails(getQuestionTypePack('addition-equation')!).map((item) => item.id)).toEqual([
      'addition-equation',
      'subtraction-equation',
    ])
    expect(getQuestionTypePack('shape-sort')?.id).toBe('shape')
    expect(getPackQuestionDetails(getQuestionTypePack('shape-find')!).map((item) => item.id)).toEqual([
      'shape-find',
      'shape-name',
      'shape-feature',
      'shape-sort',
    ])
  })

  it('keeps practice available only for the implemented addition equation type', () => {
    expect(questionTypeDetails.filter((detail) => detail.practiceHref)).toEqual([
      expect.objectContaining({ id: 'addition-equation', practiceHref: '/types/addition-equation/practice' }),
    ])
  })
})
