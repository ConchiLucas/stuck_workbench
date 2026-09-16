import { describe, expect, it } from 'vitest'
import type { QTaskItem } from '../api/qtaskTypes'
import { qtaskItemToQuizQuestion } from './qtaskToQuiz'

const item: QTaskItem = {
  seq: 1,
  kpId: 1,
  questionId: 101,
  charText: '一',
  code: 'glyph_sense',
  stem: '看字图，选出义图',
  options: [{ label: '一' }, { label: '二' }],
  answerIndex: 0,
  glyphImageUrl: 'http://localhost:19091/one-glyph.png',
  senseImageUrl: 'http://localhost:19091/one-sense.png',
  speechAudioUrl: 'http://localhost:19091/one.mp3',
  optionAssets: [
    {
      label: '一',
      kpId: 1,
      glyphImageUrl: 'http://localhost:19091/one-glyph.png',
      senseImageUrl: 'http://localhost:19091/one-sense.png',
      speechAudioUrl: 'http://localhost:19091/one.mp3',
    },
    {
      label: '二',
      kpId: 2,
      glyphImageUrl: 'http://localhost:19091/two-glyph.png',
      senseImageUrl: 'http://localhost:19091/two-sense.png',
    },
  ],
}

describe('qtaskItemToQuizQuestion', () => {
  it('maps stored assets onto stem and options without literacy API', () => {
    const q = qtaskItemToQuizQuestion(item)
    expect(q).not.toBeNull()
    expect(q?.target.glyphImageUrl).toBe('http://localhost:19091/one-glyph.png')
    expect(q?.options[0].imageUrl).toBe('http://localhost:19091/one-sense.png')
    expect(q?.options[1].kpId).toBe(2)
    expect(q?.options[1].imageUrl).toBe('http://localhost:19091/two-sense.png')
    expect(q?.speechUrl).toBe('http://localhost:19091/one.mp3')
  })

  it('returns null for unsupported question codes', () => {
    expect(qtaskItemToQuizQuestion({ ...item, code: 'listen_glyph' })).toBeNull()
  })
})
