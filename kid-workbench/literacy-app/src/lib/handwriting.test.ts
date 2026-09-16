import { expect, it } from 'vitest'
import { inkMatch, type InkStroke } from './handwriting'

const mountain: InkStroke[] = [
  [{ x: 20, y: 80 }, { x: 80, y: 80 }],
  [{ x: 50, y: 80 }, { x: 50, y: 20 }],
  [{ x: 80, y: 80 }, { x: 80, y: 35 }],
]

it('accepts the same shape even when it is moved and scaled', () => {
  const shifted = mountain.map((stroke) => stroke.map((point) => ({ x: point.x * 0.4 + 12, y: point.y * 0.4 + 30 })))
  expect(inkMatch(shifted, mountain).ok).toBe(true)
  expect(inkMatch(shifted, mountain).score).toBeGreaterThan(0.7)
})

it('rejects a different character shape', () => {
  const line: InkStroke[] = [[{ x: 10, y: 50 }, { x: 90, y: 50 }]]
  expect(inkMatch(line, mountain).ok).toBe(false)
})

it('rejects 山 when the right stroke is missing', () => {
  const incomplete = [mountain[0], mountain[1]]
  expect(inkMatch(incomplete, mountain).ok).toBe(false)
})

it('accepts 山 written as one connected stroke', () => {
  const connected: InkStroke[] = [[
    { x: 50, y: 20 },
    { x: 50, y: 80 },
    { x: 20, y: 80 },
    { x: 80, y: 80 },
    { x: 80, y: 35 },
  ]]
  expect(inkMatch(connected, mountain).ok).toBe(true)
})
