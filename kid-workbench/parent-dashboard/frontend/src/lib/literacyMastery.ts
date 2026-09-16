import type { MatrixSkill } from '../api/types'

export const LITERACY_TYPES = [
  { code: 'glyph_sense', short: '义', name: '看字选义' },
  { code: 'sense_char', short: '字', name: '看义选字' },
  { code: 'write_char', short: '写', name: '手写' },
] as const

export type LiteracySkillCode = typeof LITERACY_TYPES[number]['code']
export type LiteracyView = 'all' | LiteracySkillCode
type SkillPoint = { skills?: MatrixSkill[] }

export function skillState(skill: MatrixSkill) {
  const lit = skill.status === 'mastered' || skill.status === 'review_due'
  return {
    ...skill,
    lit,
    detail: lit ? '已掌握' : skill.attempts > 0 ? '已作答未掌握' : '未作答',
  }
}

export function literacyState(point: SkillPoint) {
  const skills = LITERACY_TYPES.map(type => ({
    ...type,
    ...skillState(point.skills?.find(skill => skill.code === type.code) ?? {
      code: type.code, status: 'not_started', attempts: 0, accuracy: 0,
    }),
  }))
  const mastered = skills.filter(skill => skill.lit).length
  const complete = mastered === LITERACY_TYPES.length
  return { skills, mastered, complete, label: complete ? '完全掌握' : mastered ? '部分题型掌握' : '尚无题型掌握' }
}

export function literacySummary(points: SkillPoint[]) {
  const states = points.map(literacyState)
  const types = Object.fromEntries(LITERACY_TYPES.map(({ code }) => {
    const skills = states.flatMap(state => state.skills.filter(skill => skill.code === code))
    return [code, {
      mastered: skills.filter(skill => skill.lit).length,
      answered: skills.filter(skill => !skill.lit && skill.attempts > 0).length,
      unattempted: skills.filter(skill => !skill.lit && skill.attempts === 0).length,
      total: points.length,
    }]
  })) as Record<LiteracySkillCode, { mastered: number; answered: number; unattempted: number; total: number }>
  return {
    overall: {
      mastered: states.filter(state => state.complete).length,
      partial: states.filter(state => !state.complete && state.mastered > 0).length,
      none: states.filter(state => state.mastered === 0).length,
      total: points.length,
    },
    types,
  }
}

export function accuracyLabel(accuracy: number, attempts: number) {
  return attempts > 0 ? `${Math.round(accuracy * 100)}%` : '—'
}
