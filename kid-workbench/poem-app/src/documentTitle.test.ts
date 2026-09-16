import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { expect, test } from 'vitest'

test('browser tab uses the poetry garden name', () => {
  const html = readFileSync(resolve(__dirname, '../index.html'), 'utf8')
  expect(html).toContain('<title>小儿诗园</title>')
})
