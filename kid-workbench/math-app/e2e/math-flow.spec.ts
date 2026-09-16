import { expect, test } from './fixture'

const groups = [
  { title: '看算式选答案', ids: ['addition-equation', 'subtraction-equation'] },
  { title: '看数量图选答案', ids: ['addition-story', 'subtraction-story'] },
  { title: '补全算式', ids: ['addition-missing', 'subtraction-missing'] },
  { title: '判断算式对错', ids: ['addition-judge', 'subtraction-error'] },
  { title: '认识图形', ids: ['shape-find', 'shape-name', 'shape-feature', 'shape-sort'] },
]

test('five entrances reach all twelve published material details without submitting attempts', async ({ page, mockAPI }) => {
  expect(mockAPI.catalog.items).toHaveLength(12)
  const visited: string[] = []
  for (const group of groups) {
    await page.goto('/')
    await expect(page.getByTestId('question-type-card')).toHaveCount(5)
    await page.getByRole('link', { name: group.title, exact: true }).click()
    for (const [index, id] of group.ids.entries()) {
      const detail = mockAPI.catalog.items.find(item => item.id === id)!
      await expect(page).toHaveURL(new RegExp(`/types/${id}$`))
      await expect(page.getByRole('heading', { name: detail.title, exact: true })).toBeVisible()
      await expect(page.locator('.math-player-prompt')).toHaveAttribute('aria-label', detail.example.prompt)
      await expect(page.getByText('试做示例，不计入学习进度')).toBeVisible()
      await expect(page.getByRole('progressbar')).toHaveCount(0)
      visited.push(id)
      if (index < group.ids.length - 1) await page.getByRole('link', { name: '下一题' }).click()
    }
  }
  expect(visited.sort()).toEqual(mockAPI.catalog.items.map(item => item.id).sort())
  expect(mockAPI.reads.length).toBeGreaterThan(0)
  expect(mockAPI.writes).toEqual([])
})

test('subtraction shows removed objects and checks answers locally with reset', async ({ page, mockAPI }) => {
  await page.goto('/types/subtraction-story')
  await expect(page.getByLabel('原来 8 个，拿走 3 个')).toBeVisible()
  await expect(page.getByLabel('拿走的苹果', { exact: true })).toHaveCount(3)
  await expect(page.getByLabel('剩下的苹果', { exact: true })).toHaveCount(5)
  await page.getByRole('button', { name: '4', exact: true }).click()
  await expect(page.getByRole('status')).toHaveText('再想一想，再试一次。')
  await page.getByRole('button', { name: '5', exact: true }).click()
  await expect(page.getByRole('status')).toHaveText('答对了！')
  await expect(page).toHaveURL(/\/types\/subtraction-story$/)
  await page.getByRole('button', { name: '重新试做' }).click()
  await expect(page.getByRole('status')).toHaveCount(0)
  expect(mockAPI.writes).toEqual([])
})

test('classification requires placement, checks mistakes and resets on small screens', async ({ page }) => {
  await page.goto('/types/shape-sort')
  const check = page.getByRole('button', { name: '检查分类' })
  await expect(check).toBeDisabled()
  for (const name of ['圆形', '椭圆形', '三角形', '正方形', '菱形']) {
    await page.getByRole('combobox', { name: `${name}放在哪一组`, exact: true }).selectOption('1')
  }
  await check.click()
  await expect(page.getByRole('status')).toHaveText('再想一想，再试一次。')
  for (const name of ['圆形', '椭圆形']) {
    await page.getByRole('combobox', { name: `${name}放在哪一组`, exact: true }).selectOption('0')
  }
  await check.click()
  await expect(page.getByRole('status')).toHaveText('答对了！')
  await page.getByRole('button', { name: '重新试做' }).click()
  await expect(check).toBeDisabled()
  const dimensions = await page.locator('.type-detail-page').evaluate(element => ({ width: element.clientWidth, scroll: element.scrollWidth }))
  expect(dimensions.scroll).toBeLessThanOrEqual(dimensions.width)
})

test('missing material explains its absence without falling back to local questions', async ({ page, mockAPI }) => {
  mockAPI.catalog.items = []
  await page.goto('/types/addition-equation')
  await expect(page.getByRole('heading', { name: '这个题型还没有发布素材' })).toBeVisible()
  await expect(page.locator('.math-player')).toHaveCount(0)
  await expect(page.getByText('3 + 5 = ?')).toHaveCount(0)
  await expect(page.getByRole('link', { name: '返回首页' })).toBeVisible()
})

test('catalog failure has an explicit retry and never opens the old example bank', async ({ page, mockAPI }) => {
  mockAPI.unavailable = true
  await page.goto('/types/addition-equation')
  await expect(page.getByRole('alert')).toHaveText('素材服务暂不可用')
  await expect(page.locator('.math-player')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '用示例题' })).toHaveCount(0)
  mockAPI.unavailable = false
  await page.getByRole('button', { name: '重新加载' }).click()
  await expect(page.locator('.math-player-prompt')).toHaveAttribute('aria-label', '3 + 5 = ?')
})
