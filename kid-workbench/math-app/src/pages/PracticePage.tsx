import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { mathApi } from '../api/math'
import type { PlanDetail } from '../api/types'
import { mathAudio } from '../audio/controller'
import { MathVisualCard } from '../components/MathVisual'
import { ShapeGlyph } from '../components/ShapeGlyph'
import { useChildStore } from '../store/childStore'
import { pendingAnswerStore } from '../store/pendingAnswerStore'

export function PracticePage() {
  const childId = useChildStore((state) => state.childId)
  const planId = Number(useParams().planId)
  const navigate = useNavigate()
  const [detail, setDetail] = useState<PlanDetail>()
  const [index, setIndex] = useState(0)
  const [feedback, setFeedback] = useState('')
  const [busy, setBusy] = useState(false)
  const [settlementRequired, setSettlementRequired] = useState(false)
  const [settlementError, setSettlementError] = useState(false)
  const [audioReady, setAudioReady] = useState(() => mathAudio.isUnlocked())
  const [loadError, setLoadError] = useState(false)
  const [loadNonce, setLoadNonce] = useState(0)
  const startedAt = useRef(Date.now())
  const item = detail?.items[index]

  const finishPlan = async () => {
    setBusy(true)
    setSettlementError(false)
    try {
      await mathApi.finish(childId, planId)
      navigate(`/result/${planId}`)
    } catch {
      setSettlementRequired(true)
      setSettlementError(true)
    } finally { setBusy(false) }
  }

  useEffect(() => {
    let live = true
    setLoadError(false)
    mathApi.plan(childId, planId).then(async (loaded) => {
      if (loaded.plan.status === 'pending') await mathApi.startPlan(childId, planId)
      if (live) {
        const firstPending = loaded.items.findIndex((value) => value.status === 'pending')
        if (firstPending === -1) {
          if (loaded.plan.status === 'done') {
            navigate(`/result/${planId}`, { replace: true })
            return
          }
          setDetail(loaded)
          setSettlementRequired(true)
          return
        }
        setIndex(Math.max(0, firstPending))
        setDetail(loaded)
      }
    }).catch(() => { if (live) setLoadError(true) })
    return () => { live = false }
  }, [childId, loadNonce, navigate, planId])

  useEffect(() => {
    if (!item?.question.audioUrl) return
    const next = detail?.items[index + 1]?.question.audioUrl
    mathAudio.preload([item.question.audioUrl, ...(next ? [next] : [])])
    if (mathAudio.isUnlocked()) void mathAudio.play(item.question.audioUrl).catch(() => setFeedback('朗读暂时没播放，请点一下再听'))
    return () => mathAudio.stop()
  }, [detail?.items, index, item?.itemId, item?.question.audioUrl])

  const choose = async (optionIndex: number) => {
    if (!item || busy) return
    setBusy(true)
    const tryNumber = item.tries + 1
    const clientId = pendingAnswerStore.getState().clientId(planId, item.itemId, tryNumber)
    try {
      const result = await mathApi.answer(childId, planId, item.itemId, { clientId, optionIndex, costMs: Date.now() - startedAt.current })
      pendingAnswerStore.getState().acknowledge(planId, item.itemId, tryNumber)
      if (result.canRetry) {
        item.tries = result.tries
        setFeedback('再想一想，你还有一次机会')
        startedAt.current = Date.now()
      } else if (detail && index + 1 < detail.items.length) {
        setFeedback(result.correct ? '答对啦！' : `正确答案是第 ${result.answerIndex + 1} 个`)
        window.setTimeout(() => { setIndex((value) => value + 1); setFeedback(''); startedAt.current = Date.now() }, 350)
      } else {
        setDetail((current) => current ? { ...current, items: current.items.map((value) => value.itemId === item.itemId ? { ...value, status: result.status, tries: result.tries } : value) } : current)
        await finishPlan()
      }
    } catch {
      setFeedback('答案已经保留，网络恢复后请再点一次')
    } finally { setBusy(false) }
  }

  if (settlementRequired) return <section className="page practice-page settlement-page"><article className="question-sheet settlement-card">
    <p className="eyebrow">ALL QUESTIONS DONE</p><h1>题目都完成啦</h1>
    <p>{settlementError ? '成绩保存遇到了一点网络问题，答案已经保留。' : '只差最后一步，收好今天的星星。'}</p>
    <button className="primary-button" disabled={busy} onClick={() => void finishPlan()}>{settlementError ? '重试结算' : '完成结算'}</button>
  </article></section>
  if (loadError) return <section className="page practice-page settlement-page"><article className="question-sheet settlement-card">
    <p className="eyebrow">KEEP YOUR PLACE</p><h1>练习本暂时没打开</h1><p>你的进度没有丢失，可以再试一次。</p>
    <button className="primary-button" onClick={() => setLoadNonce((value) => value + 1)}>重试加载</button>
  </article></section>
  if (!item) return <section className="page practice-page"><p className="loading">正在打开练习本…</p></section>
  return <section className="page practice-page">
    <header className="practice-header"><button className="icon-button" onClick={() => navigate('/')}>×<span className="sr-only">退出练习</span></button>
      <div className="progress-dots" aria-label={`第 ${index + 1} 题，共 ${detail.items.length} 题`}>{detail.items.map((_, dot) => <i key={dot} className={dot < index ? 'done' : dot === index ? 'current' : ''} />)}</div><span>{index + 1}/{detail.items.length}</span>
    </header>
    <article className="question-sheet">
      {!audioReady && <button className="audio-unlock" onClick={() => void mathAudio.unlock(item.question.audioUrl).then(() => setAudioReady(true)).catch(() => setFeedback('朗读暂时没播放，请再点一次'))}>🔊 点击播放题目朗读</button>}
      <div className="question-label"><span>题目 {String(index + 1).padStart(2, '0')}</span><button className="sound-button" onClick={() => void mathAudio.unlock(item.question.audioUrl).then(() => setAudioReady(true))}>🔊 再听一遍</button></div>
      <h1>{item.question.stem}</h1><MathVisualCard visual={item.question.visual} />
      <div className="option-grid">{item.question.options.map((option, optionIndex) => <button key={optionIndex} className="option-button" disabled={busy} onClick={() => choose(optionIndex)}>
        <span className="option-letter">{String.fromCharCode(65 + optionIndex)}</span>{option.shape ? <ShapeGlyph shape={option.shape} /> : option.label}
      </button>)}</div>
      <p className="feedback" role="status">{feedback}</p>
    </article>
  </section>
}
