import { test as base, expect, type Route } from '@playwright/test'
import { readFile } from 'node:fs/promises'
import { resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { PinyinAnswerRequest, PinyinAnswerResult, PinyinGeneratedQuiz, PinyinGeneratedQuizType } from '../src/api/types'

const origin = 'https://pinyin.test'
const dist = fileURLToPath(new URL('../dist/', import.meta.url))
const mime: Record<string, string> = { js: 'text/javascript', css: 'text/css', html: 'text/html', json: 'application/json', webmanifest: 'application/manifest+json', png: 'image/png', svg: 'image/svg+xml', woff2: 'font/woff2', ico: 'image/x-icon' }

type Instance = { childId: number; question: PinyinGeneratedQuiz; correctOptionId: string; accepted?: PinyinAnswerResult; payload?: PinyinAnswerRequest }
export class MockPinyinAPI {
  instances = new Map<string, Instance>()
  submissions: Array<{ childId: number; instanceId: string; input: PinyinAnswerRequest }> = []
  generated: Array<{ childId: number; type: PinyinGeneratedQuizType; excludeTargetIds: number[] }> = []
  reads: string[] = []
  ledger: Array<{ childId: number; kpId: number; type: PinyinGeneratedQuizType; optionId: string }> = []
  unexpected: string[] = []
  failNextAnswer?: 'before-write' | 'after-write'

  async handle(route: Route) {
    const request = route.request()
    const path = new URL(request.url()).pathname
    const envelope = (data: unknown, status = 200, error: unknown = null) => route.fulfill({ status, json: { data, error } })
    const generated = path.match(/^\/api\/v1\/children\/(\d+)\/pinyin\/quiz\/generate$/)
    if (generated && request.method() === 'POST') {
      const { type, excludeTargetIds } = request.postDataJSON() as { type: PinyinGeneratedQuizType; excludeTargetIds: number[] }
      const childId = Number(generated[1])
      this.generated.push({ childId, type, excludeTargetIds })
      const offset = this.generated.length
      const targetId = (type === 'blend' ? 700 : 10) + offset
      const options = type === 'shape' || type === 'blend'
        ? [{ id: String(targetId + 100), label: 'pā', speechText: '趴' }, { id: String(targetId), label: 'bā', speechText: '八' }, { id: String(targetId + 200), label: 'bǎ', speechText: '把' }, { id: String(targetId + 300), label: 'mā', speechText: '妈' }]
        : [{ id: String(targetId + 100), label: 'b' }, { id: String(targetId), label: 'p' }, { id: String(targetId + 200), label: 'm' }, { id: String(targetId + 300), label: 'f' }]
      const question: PinyinGeneratedQuiz = {
        instanceId: `fixture-${childId}-${type}-${offset}`, type, targetId, kpId: 9000 + targetId,
        expiresAt: '2099-01-01T00:00:00Z', stem: 'stem', speechText: '坡',
        visual: type === 'blend' ? { kind: 'blend', initial: 'b', final: 'ā' } : type === 'shape' ? { kind: 'glyph', text: 'ɑ' } : type === 'inword' ? { kind: 'char', text: '坡' } : { kind: 'sound' },
        options,
      }
      this.instances.set(question.instanceId, { childId, question, correctOptionId: String(targetId) })
      return envelope(question)
    }
    const instancePath = path.match(/^\/api\/v1\/children\/(\d+)\/pinyin\/quiz\/([^/]+)(\/answer)?$/)
    if (instancePath) {
      const [, child, instanceId, answer] = instancePath
      const childId = Number(child)
      const instance = this.instances.get(instanceId)
      if (!instance || instance.childId !== childId) return envelope(null, 404, { code: 'not_found', message: '题目不存在' })
      if (!answer && request.method() === 'GET') {
        this.reads.push(instanceId)
        return envelope({ ...instance.question, ...(instance.accepted ? { acceptedResult: instance.accepted } : {}) })
      }
      if (answer && request.method() === 'POST') {
        const input = request.postDataJSON() as PinyinAnswerRequest
        this.submissions.push({ childId, instanceId, input })
        expect(Object.keys(input).sort()).toEqual(['clientId', 'costMs', 'optionId'])
        expect(typeof input.optionId).toBe('string')
        expect(instance.question.options.some(option => option.id === input.optionId)).toBe(true)
        expect(input.clientId).toBeTruthy()
        if (this.failNextAnswer === 'before-write') {
          this.failNextAnswer = undefined
          return envelope(null, 503, { code: 'unavailable', message: '答案还未确认，请重试' })
        }
        if (instance.accepted) {
          if (JSON.stringify(instance.payload) !== JSON.stringify(input)) return envelope(null, 409, { code: 'conflict', message: '题目已提交' })
          return envelope(instance.accepted)
        }
        instance.payload = input
        instance.accepted = {
          instanceId, attemptId: this.ledger.length + 1, selectedOptionId: input.optionId,
          correct: input.optionId === instance.correctOptionId, answerOptionId: instance.correctOptionId,
          skill: { code: instance.question.type, status: 'learning' },
          knowledge: { kpId: instance.question.kpId, status: 'learning', newlyMastered: false },
        }
        this.ledger.push({ childId, kpId: instance.question.kpId, type: instance.question.type, optionId: input.optionId })
        if (this.failNextAnswer === 'after-write') {
          this.failNextAnswer = undefined
          return route.abort('failed')
        }
        return envelope(instance.accepted)
      }
    }
    this.unexpected.push(`${request.method()} ${path}`)
    return envelope(null, 501, { code: 'unexpected_fixture_request', message: path })
  }
}

// Every request is fulfilled from dist or an in-memory API. Nothing reaches a server.
export const test = base.extend<{ mockAPI: MockPinyinAPI }>({
  mockAPI: [async ({ page }, use) => {
    const mock = new MockPinyinAPI()
    await readFile(resolve(dist, 'index.html')) // Fail early with a useful missing-build path.
    await page.route('**/*', async route => {
      const request = route.request()
      const url = new URL(request.url())
      if (url.origin !== origin) {
        mock.unexpected.push(`external request ${request.url()}`)
        return route.abort('blockedbyclient')
      }
      if (url.pathname.startsWith('/api/')) return mock.handle(route)
      const relativePath = request.isNavigationRequest() ? 'index.html' : decodeURIComponent(url.pathname).replace(/^\/+/, '')
      const path = resolve(dist, relativePath)
      if (!path.startsWith(resolve(dist) + sep)) return route.abort('blockedbyclient')
      try {
        await route.fulfill({ body: await readFile(path), contentType: mime[path.split('.').at(-1)!] ?? 'application/octet-stream' })
      } catch { await route.fulfill({ status: 404, body: 'Missing local test asset' }) }
    })
    await use(mock)
    expect(mock.unexpected, 'No API or network request may bypass the isolated fixture').toEqual([])
  }, { auto: true }],
})
export { expect }
