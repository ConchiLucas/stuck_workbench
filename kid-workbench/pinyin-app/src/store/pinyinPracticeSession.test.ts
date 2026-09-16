import { beforeEach, expect, it, vi } from 'vitest'
import { createPinyinPracticeStore, sessionKey } from './pinyinPracticeSession'
import type { pinyinApi } from '../api/pinyin'
import type { PinyinAnswerResult, PinyinGeneratedQuiz } from '../api/types'

const question = (id = 'q1'): PinyinGeneratedQuiz => ({ instanceId: id, type: 'listen', targetId: 11, kpId: 101, expiresAt: '2099-01-01T00:00:00Z', stem: '听', visual: { kind: 'sound' }, options: [{ id: 'a', label: 'a' }, { id: 'b', label: 'b' }] })
const result: PinyinAnswerResult = { instanceId: 'q1', attemptId: 1, selectedOptionId: 'a', correct: false, answerOptionId: 'b', skill: { code: 'listen', status: 'learning' }, knowledge: { kpId: 101, status: 'learning', newlyMastered: false } }
const key = sessionKey(1, 'listen')
function fixture() {
  const api = { generateQuiz: vi.fn<typeof pinyinApi.generateQuiz>(async () => question()), quizInstance: vi.fn<typeof pinyinApi.quizInstance>(async () => ({ ...question() })), answerQuiz: vi.fn<typeof pinyinApi.answerQuiz>(async () => result) }
  const store = createPinyinPracticeStore({ api, storage: localStorage, count: 1 })
  return { api, store }
}
beforeEach(() => localStorage.clear())

it('freezes the first payload and retries it unchanged after network failure', async () => {
  const { api, store } = fixture()
  api.answerQuiz.mockRejectedValueOnce(new Error('network'))
  await store.getState().ensure(1, 'listen')
  const id = store.getState().sessions[key]!.id
  await store.getState().submit(1, 'listen', id, 'q1', 'a', 123)
  expect(store.getState().sessions[key]!.entries[0].result).toBeUndefined()
  await store.getState().submit(1, 'listen', id, 'q1', 'b', 999)
  expect(api.answerQuiz.mock.calls[1]).toEqual(api.answerQuiz.mock.calls[0])
  expect(api.answerQuiz.mock.calls[0]?.[2]).toMatchObject({ optionId: 'a', costMs: 123 })
  expect(store.getState().sessions[key]!.entries[0].result?.correct).toBe(false)
})

it('does not duplicate a submission while the first request is unresolved', async () => {
  const { api, store } = fixture()
  let accept!: (value: PinyinAnswerResult) => void
  api.answerQuiz.mockImplementation(() => new Promise(resolve => { accept = resolve }))
  await store.getState().ensure(1, 'listen')
  const id = store.getState().sessions[key]!.id
  const first = store.getState().submit(1, 'listen', id, 'q1', 'a', 10)
  await store.getState().submit(1, 'listen', id, 'q1', 'b', 20)
  expect(api.answerQuiz).toHaveBeenCalledTimes(1)
  accept(result)
  await first
})

it('verifies saved instances after refresh and recovers a lost accepted response without posting again', async () => {
  const { api, store } = fixture()
  api.answerQuiz.mockRejectedValueOnce(new Error('response lost'))
  await store.getState().ensure(1, 'listen')
  await store.getState().submit(1, 'listen', store.getState().sessions[key]!.id, 'q1', 'a', 10)
  const restored = createPinyinPracticeStore({ api: { ...api, quizInstance: vi.fn(async () => ({ ...question(), acceptedResult: result })) }, storage: localStorage, count: 1 })
  await restored.getState().ensure(1, 'listen')
  expect(restored.getState().sessions[key]!.entries[0].result).toEqual(result)
  expect(restored.getState().sessions[key]!.entries[0].pending).toBeUndefined()
  expect(api.answerQuiz).toHaveBeenCalledTimes(1)
})

it('does not trust a locally cached accepted result if the server reports an unanswered question', async () => {
  const { api, store } = fixture()
  await store.getState().ensure(1, 'listen')
  await store.getState().submit(1, 'listen', store.getState().sessions[key]!.id, 'q1', 'a', 10)
  const restored = createPinyinPracticeStore({ api, storage: localStorage, count: 1 })
  await restored.getState().ensure(1, 'listen')
  expect(api.quizInstance).toHaveBeenCalledWith(1, 'q1')
  expect(restored.getState().sessions[key]!.entries[0].result).toBeUndefined()
})

it('isolates children and ignores the generation response of a replaced session', async () => {
  const { api, store } = fixture()
  let finish!: (value: PinyinGeneratedQuiz) => void
  api.generateQuiz.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
  const stale = store.getState().ensure(1, 'listen')
  store.getState().restart(1, 'listen')
  api.generateQuiz.mockResolvedValueOnce(question('fresh'))
  await store.getState().ensure(1, 'listen')
  await store.getState().ensure(2, 'listen')
  finish(question('stale'))
  await stale
  expect(store.getState().sessions[key]!.entries[0].question.instanceId).toBe('fresh')
  expect(store.getState().sessions[sessionKey(2, 'listen')]!.entries[0].result).toBeUndefined()
})

it('ignores the answer response of a replaced session and keeps accepted questions read only', async () => {
  const { api, store } = fixture()
  await store.getState().ensure(1, 'listen')
  const id = store.getState().sessions[key]!.id
  await store.getState().submit(1, 'listen', id, 'q1', 'a', 10)
  await store.getState().submit(1, 'listen', id, 'q1', 'b', 10)
  expect(api.answerQuiz).toHaveBeenCalledTimes(1)
  store.getState().restart(1, 'listen')
  await store.getState().ensure(1, 'listen')
  let finish!: (value: PinyinAnswerResult) => void
  api.answerQuiz.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
  const pending = store.getState().submit(1, 'listen', store.getState().sessions[key]!.id, 'q1', 'a', 10)
  store.getState().restart(1, 'listen')
  await store.getState().ensure(1, 'listen')
  finish(result)
  expect(await pending).toBeUndefined()
  expect(store.getState().sessions[key]!.entries[0].result).toBeUndefined()
})

it('restores an unconfirmed choice as retryable with its original client ID', async () => {
  const { api, store } = fixture()
  api.answerQuiz.mockRejectedValueOnce(new Error('offline'))
  await store.getState().ensure(1, 'listen')
  await store.getState().submit(1, 'listen', store.getState().sessions[key]!.id, 'q1', 'a', 99)
  const original = store.getState().sessions[key]!.entries[0].pending
  const restored = createPinyinPracticeStore({ api, storage: localStorage, count: 1 })
  await restored.getState().ensure(1, 'listen')
  const entry = restored.getState().sessions[key]!.entries[0]
  expect(entry.pending).toEqual(original)
  expect(entry.error).toContain('重试')
})

it('preserves position independently by child and type', async () => {
  const { api, store } = fixture()
  const many = createPinyinPracticeStore({ api, storage: localStorage, count: 4 })
  await many.getState().ensure(1, 'listen')
  many.getState().setPosition(1, 'listen', many.getState().sessions[key]!.id, 3)
  await store.getState().ensure(2, 'listen')
  api.generateQuiz.mockResolvedValueOnce({ ...question('shape-1'), type: 'shape' })
  await store.getState().ensure(1, 'shape')
  const restored = createPinyinPracticeStore({ api, storage: localStorage, count: 4 })
  await restored.getState().ensure(1, 'listen')
  expect(restored.getState().sessions[key]!.position).toBe(3)
  expect(store.getState().sessions[sessionKey(2, 'listen')]!.position).toBe(1)
  expect(store.getState().sessions[sessionKey(1, 'shape')]!.entries[0].question.type).toBe('shape')
})

it('does not show local results when refresh verification fails transiently', async () => {
  const { ApiError } = await import('../api/client')
  const { api, store } = fixture()
  await store.getState().ensure(1, 'listen')
  await store.getState().submit(1, 'listen', store.getState().sessions[key]!.id, 'q1', 'a', 10)
  api.quizInstance.mockRejectedValueOnce(new ApiError(503, 'unavailable', '稍后重试'))
  const restored = createPinyinPracticeStore({ api, storage: localStorage, count: 1 })
  await restored.getState().ensure(1, 'listen')
  expect(restored.getState().sessions[key]!.status).toBe('error')
  expect(restored.getState().sessions[key]!.verified).toBe(false)
  expect(restored.getState().sessions[key]!.entries[0].result).toBeUndefined()
  await restored.getState().ensure(1, 'listen')
  expect(restored.getState().sessions[key]!.status).toBe('ready')
})

it('does not submit an expired restored instance and can replace it', async () => {
  const { ApiError } = await import('../api/client')
  const { api, store } = fixture()
  await store.getState().ensure(1, 'listen')
  api.quizInstance.mockRejectedValueOnce(new ApiError(410, 'expired', '题目已过期'))
  const restored = createPinyinPracticeStore({ api, storage: localStorage, count: 1 })
  await restored.getState().ensure(1, 'listen')
  const session = restored.getState().sessions[key]!
  expect(session.entries[0].unavailable).toBe(true)
  await restored.getState().submit(1, 'listen', session.id, 'q1', 'a', 10)
  expect(api.answerQuiz).not.toHaveBeenCalled()
  restored.getState().restart(1, 'listen')
  api.generateQuiz.mockResolvedValueOnce(question('replacement'))
  await restored.getState().ensure(1, 'listen')
  expect(restored.getState().sessions[key]!.entries[0].question.instanceId).toBe('replacement')
})

it('reconciles another tab’s accepted result after an instance conflict', async () => {
  const { ApiError } = await import('../api/client')
  const { api, store } = fixture()
  await store.getState().ensure(1, 'listen')
  api.answerQuiz.mockRejectedValueOnce(new ApiError(409, 'conflict', '题目已提交'))
  api.quizInstance.mockResolvedValueOnce({ ...question(), acceptedResult: result })
  const accepted = await store.getState().submit(1, 'listen', store.getState().sessions[key]!.id, 'q1', 'b', 10)
  expect(accepted).toEqual(result)
  expect(store.getState().sessions[key]!.entries[0].pending).toBeUndefined()
})

it('retains confirmed snapshots when restarting a legacy session and refuses to discard pending', async () => {
 const {api,store}=fixture()
 const accepted={question:question(),result}
 store.setState({sessions:{[key]:{id:'old',childId:1,type:'listen',position:1,status:'ready',verified:true,error:'',entries:[accepted]}}})
 store.getState().restart(1,'listen',true)
 await store.getState().ensure(1,'listen')
 expect(store.getState().sessions[key]!.entries[0]).toEqual(accepted)
 expect(api.generateQuiz).not.toHaveBeenCalled()
 const pending={clientId:'pending',optionId:'a',costMs:10}
 const before={...store.getState().sessions[key]!,entries:[{question:question(),pending}]}
 store.setState({sessions:{[key]:before}})
 store.getState().restart(1,'listen',true)
 expect(store.getState().sessions[key]).toBe(before)
})
