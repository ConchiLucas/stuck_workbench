import { expect, test } from './fixture'

test('iPad home and pinyin map keep child-sized controls', async ({ page }) => {
  await page.route('**/api/v1/children/1/pinyin/home', (route) => route.fulfill({ json: { data: {
    child: { id: 1, name: '安安', grade: '一年级', avatarUrl: '', flowers: 8 }, currentPlan: null, dueCount: 1, modules: [],
  }, error: null } }))
  await page.route('**/api/v1/pinyin/modules', (route) => route.fulfill({ json: { data: [{ code: 'initials', name: '声母', orderNo: 1, itemCount: 1 }], error: null } }))
  await page.route('**/api/v1/children/1/pinyin/progress', (route) => route.fulfill({ json: { data: [{ kpId: 100, letter: 'b', status: 'learning', skills: [] }], error: null } }))
  await page.route('**/api/v1/pinyin/modules/initials/items', (route) => route.fulfill({ json: { data: [{ kpId: 100, letter: 'b', moduleCode: 'initials', moduleName: '声母', soloText: 'b', wordText: '广播', hasSoloSpeech: true, hasWordSpeech: true, hasGlyph: true, orderNo: 1 }], error: null } }))

  await page.goto('/')
  await expect(page.getByRole('link', { name: '听音选字母' })).toBeVisible()
  for (const button of await page.getByRole('button').all()) {
    expect((await button.boundingBox())?.height).toBeGreaterThanOrEqual(56)
  }
  await page.goto('/map')
  await expect(page.getByRole('heading', { name: '我的拼音图' })).toBeVisible()
  await expect(page.getByRole('link', { name: '关闭' })).toHaveAttribute('href', '/')
  await expect(page.getByRole('link', { name: /b.*学习中/ })).toBeVisible()
})

test('practice is a single-question workspace without a question list', async ({ page }) => {
  await page.route('**/api/v1/children/1/pinyin/plans/7', (route) => route.fulfill({ json: { data: {
    plan: { id: 7, planDate: '2026-09-02', seqNo: 1, subjectCode: 'pinyin', status: 'doing', targetCount: 8, doneCount: 2, correctCount: 2, stars: 0, durationSec: 0 },
    items: [{ id: 71, seq: 3, kpId: 100, letter: 'b', bucket: 'new', status: 'pending', tries: 0, picks: '', optionOrder: '0,1,2,3', question: {
      id: 1000, code: 'listen', type: 'choice', stem: '听一听，选出你听到的拼音', options: [{ label: 'b' }, { label: 'p' }, { label: 'm' }, { label: 'f' }], visual: {}, speech: {},
    } }],
  }, error: null } }))

  await page.goto('/practice/7')
  await expect(page.getByRole('link', { name: '退出练习' })).toBeVisible()
  await expect(page.locator('.topbar')).toHaveCount(0)
  await expect(page.getByRole('region', { name: '当前题目' })).toHaveCount(1)
  await expect(page.getByRole('heading', { name: '听一听，选出你听到的拼音' })).toBeVisible()
  await expect(page.getByTestId('question-list')).toHaveCount(0)
  const workspace = await page.getByRole('region', { name: '当前题目' }).boundingBox()
  expect(workspace?.height ?? 0).toBeGreaterThan(560)
  expect(await page.evaluate(() => document.documentElement.scrollHeight)).toBeLessThanOrEqual(768)
})

test('question type cards open their matching single-question practice', async ({ page }) => {
  await page.route('**/api/v1/children/1/pinyin/home', (route) => route.fulfill({ json: { data: {
    child: { id: 1, name: '安安', grade: '一年级', avatarUrl: '', flowers: 8 }, currentPlan: null, dueCount: 1, modules: [],
  }, error: null } }))


  await page.goto('/')
  await page.getByRole('link', { name: /看形认读/ }).click()
  await expect(page).toHaveURL('/practice/type/shape')
  await expect(page.getByRole('link', { name: '退出练习' })).toBeVisible()
  await expect(page.locator('.topbar')).toHaveCount(0)
  await expect(page.getByRole('region', { name: '当前题目' })).toHaveCount(1)
  await expect(page.getByRole('heading', { name: '看一看，选出四线格里的拼音' })).toHaveCount(0)
  await expect(page.getByTestId('question-list')).toHaveCount(0)
  const workspace = await page.getByRole('region', { name: '当前题目' }).boundingBox()
  expect(workspace?.height ?? 0).toBeGreaterThan(560)
  expect(await page.evaluate(() => document.documentElement.scrollHeight)).toBeLessThanOrEqual(768)
})

test('shape and blend choices expose sounds instead of written answers', async ({ page }) => {
  for (const type of ['shape', 'blend']) {
    await page.goto(`/practice/type/${type}`)
    await expect(page.getByRole('button', { name: /播放读音/ })).toHaveCount(4)
    await expect(page.locator('.answer-area .option-button strong')).toHaveCount(0)
  }
})
