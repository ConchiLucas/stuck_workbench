import { appPath } from '../appPath'
import { useEffect, useRef, type ReactNode } from 'react'
import { Navigate, useNavigate, useParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { PinyinQuestion, type PinyinQuestionView } from '@kid-workbench/pinyin-player'
import { useChildStore } from '../store/childStore'
import type { PinyinGeneratedQuiz } from '../api/types'
import { PracticeStage } from '../components/PracticeStage'
import { sessionKey, useDemoQuizStore } from '../store/demoQuizStore'
import { demoHref, demoResultHref, isDemoType, typeTitle } from './demoBank'

function playableURL(url?: string) {
  if (!url) return undefined
  try {
    const parsed = new URL(url, window.location.origin)
    if (parsed.pathname.startsWith('/api/')) return appPath(`${parsed.pathname}${parsed.search}`)
    return url
  } catch {
    return url
  }
}

export function toPinyinView(question: PinyinGeneratedQuiz): PinyinQuestionView {
  return {
    id: question.instanceId,
    type: question.type,
    stem: question.stem,
    speechText: question.speechText,
    speechUrl: question.speechUrl,
    visual: question.visual,
    options: question.options,
  }
}

export function DemoPracticePage() {
  const { type, n } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const childId = useChildStore(s => s.childId)
  const ensure = useDemoQuizStore(s => s.ensure)
  const restart = useDemoQuizStore(s => s.restart)
  const submit = useDemoQuizStore(s => s.submit)
  const setPosition = useDemoQuizStore(s => s.setPosition)
  const session = useDemoQuizStore(s => isDemoType(type) ? s.sessions[sessionKey(childId, type)] : undefined)
  const routeChildId = useRef(childId)
  const resumingChild = routeChildId.current !== childId
  const current = resumingChild ? session?.position ?? 1 : n ? Number(n) : session?.position ?? 1
  const total = session?.entries.length || 4
  const entry = session?.entries[current - 1]
  const question = entry?.question
  const pickedId = entry?.result?.selectedOptionId ?? entry?.pending?.optionId
  const locked = Boolean(entry?.pending || entry?.result || entry?.submitting || entry?.unavailable)
  const blocked = Boolean(entry?.pending && !entry?.unavailable)
  const generation = useRef(0)
  const startedAt = useRef(Date.now())

  useEffect(() => {
    if (isDemoType(type)) void ensure(childId, type)
  }, [childId, type, ensure])

  useEffect(() => {
    if (!isDemoType(type) || session?.status !== 'ready') return
    if (routeChildId.current !== childId) {
      if (n !== String(session.position)) {
        navigate(demoHref(type, session.position), { replace: true })
        return
      }
      routeChildId.current = childId
      return
    }
    if (Number.isInteger(current) && current > 0 && current <= total) {
      setPosition(childId, type, session.id, current)
    }
  }, [childId, type, n, session?.id, session?.status, session?.position, current, total, navigate, setPosition])

  useEffect(() => {
    ++generation.current
    startedAt.current = Date.now()
    return () => { ++generation.current }
  }, [childId, type, current, session?.id])

  if (!isDemoType(type)) return <Navigate to="/" replace />
  const label = typeTitle(type)
  if (!session || session.status !== 'ready' || !session.verified) {
    return (
      <PracticeStage className="demo-practice" label={label} done={0} total={4}>
        <section className="question-workspace" aria-label="当前题目">
          <div className="quiz-status" role="status">
            <p>{session?.status === 'error' ? session.error : '出题中…'}</p>
            {session?.status === 'error' ? <button type="button" className="primary-button" onClick={() => { void ensure(childId, type) }}>再试一次</button> : null}
          </div>
        </section>
      </PracticeStage>
    )
  }
  if (!Number.isInteger(current) || current < 1 || current > total || !question || !entry) {
    return <Navigate to={demoHref(type, 1)} replace />
  }

  const missingMedia = !entry.pending && !entry.result && !entry.submitting && (
    question.type === 'shape' || question.type === 'blend'
      ? question.options.some(option => !option.speechUrl?.trim())
      : !question.speechUrl?.trim()
  )
  if (missingMedia) {
    const pendingIndex = session.entries.findIndex(item => item.pending || item.submitting)
    return <PracticeStage className="demo-practice" label={label} done={session.entries.filter(item => item.result).length} total={total}>
      <section className="question-workspace" aria-label="当前题目">
        <div className="quiz-status" role="alert">
          <p>这份旧题未保存读音，请重新练习</p>
          {pendingIndex >= 0 ? <><p>还有上次提交需要确认，请先处理。</p><button type="button" className="primary-button" onClick={() => navigate(demoHref(type, pendingIndex + 1))}>返回待确认题目</button></>
            : <button type="button" className="primary-button" onClick={() => {
              const latest = useDemoQuizStore.getState().sessions[sessionKey(childId, type)]
              if (latest?.entries.some(item => item.pending || item.submitting)) return
              restart(childId, type, true)
              const position = useDemoQuizStore.getState().sessions[sessionKey(childId, type)]?.position ?? 1
              navigate(demoHref(type, position))
              void ensure(childId, type)
            }}>重新练习</button>}
        </div>
      </section>
    </PracticeStage>
  }

  const quizType = type
  const currentQuestion = question
  const sessionId = session.id
  const prevHref = current > 1 && !blocked ? demoHref(quizType, current - 1) : undefined
  const nextHref = current < total ? demoHref(quizType, current + 1) : demoResultHref(quizType)
  const done = session.entries.filter(item => item.result).length

  async function send(optionId: string) {
    const gen = generation.current
    const result = await submit(childId, quizType, sessionId, currentQuestion.instanceId, optionId, Date.now() - startedAt.current)
    if (!result) return
    void queryClient.invalidateQueries({ queryKey: ['home', childId] })
    void queryClient.invalidateQueries({ queryKey: ['progress', childId] })
    if (generation.current === gen && useChildStore.getState().childId === childId) {
      setPosition(childId, quizType, sessionId, Math.min(current + 1, total))
      navigate(nextHref)
    }
  }

  const footer: ReactNode = <>
    {entry.submitting ? <p role="status">正在确认答案…</p> : null}
    {entry.result ? <p role="status">本题已作答，可继续下一题</p> : null}
    {entry.error ? <div role="alert"><p>{entry.error}</p>
      {entry.unavailable
        ? <button type="button" className="primary-button" onClick={() => { restart(childId, quizType); navigate(demoHref(quizType, 1)); void ensure(childId, quizType) }}>重新练习</button>
        : <button type="button" className="primary-button" disabled={entry.submitting} onClick={() => { if (entry.pending) void send(entry.pending.optionId) }}>重试提交</button>}
    </div> : null}
  </>

  return (
    <PracticeStage className="demo-practice" label={label} done={done} total={total} prevHref={prevHref} nextHref={blocked ? undefined : nextHref}>
      <section className="question-workspace" aria-label="当前题目">
        <PinyinQuestion
          question={toPinyinView(currentQuestion)}
          selectedOptionId={pickedId}
          disabled={locked}
          allowSyntheticSpeech={false}
          resolveUrl={playableURL}
          onPick={id => { if (!locked) void send(id) }}
          footer={footer}
        />
      </section>
    </PracticeStage>
  )
}
