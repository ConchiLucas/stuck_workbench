import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {expect,test} from 'vitest'

test('browser tab uses a kid-facing product name', () => {
  const html = readFileSync(resolve(__dirname, '../index.html'), 'utf8')
  expect(html).toContain('<title>单词乐园</title>')
})
