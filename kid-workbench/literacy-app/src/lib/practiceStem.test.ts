import { expect, it } from 'vitest'
import { practiceStem } from './practiceStem'

it('uses a short kid-facing stem for known question codes', () => {
  expect(practiceStem('glyph_sense', '看字图，选出义图（暂用字卡）')).toBe('看这个字，选出它的意思')
  expect(practiceStem('write_char', '写一写这个字')).toBe('先听读音，再写到格子里')
})

it('strips placeholder notes from unknown stems', () => {
  expect(practiceStem('listen_char', '听一听，选出字（暂用字卡）')).toBe('听一听，选出字')
})
