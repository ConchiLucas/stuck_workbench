import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { scienceApi } from '../api/science'
import type { AnswerResult, PlanItem } from '../api/types'
import { useChildStore } from '../store/childStore'
import { usePendingAnswerStore } from '../store/pendingAnswerStore'

export function PracticePage() {
  const planId = Number(useParams().planId)
  const childId = useChildStore((s) => s.childId)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const plan = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => scienceApi.plan(childId, planId) })
  const [feedback, setFeedback] = useState<AnswerResult | null>(null)
  const startedAt = useRef(Date.now())
  const getClientId = usePendingAnswerStore((s) => s.get)
  const clearClientId = usePendingAnswerStore((s) => s.clear)
  const current = useMemo(() => plan.data?.items.find((item) => item.status !== 'completed'), [plan.data])

  useEffect(() => { startedAt.current = Date.now(); setFeedback(null) }, [current?.id])
  useEffect(() => {
    if (!plan.data || current || plan.data.plan.status === 'completed') return
    scienceApi.finish(childId, planId).then(() => navigate(`/practice/${planId}/done`, { replace: true }))
  }, [childId, current, navigate, plan.data, planId])

  const answer = useMutation({
    mutationFn: ({ item, option }: { item: PlanItem; option: number }) => {
      const key = `${planId}:${item.id}:${item.tries + 1}`
      return scienceApi.answer(childId, planId, item.id, { clientId: getClientId(key), optionIndex: option, costMs: Date.now() - startedAt.current })
        .then((result) => { clearClientId(key); return result })
    },
    onSuccess: async (result) => {
      setFeedback(result)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['plan', childId, planId] }),
        queryClient.invalidateQueries({ queryKey: ['home', childId] }),
        queryClient.invalidateQueries({ queryKey: ['progress', childId] }),
      ])
    },
  })

  if (plan.isLoading) return <div className="skeleton-block">正在准备今天的科普……</div>
  if (plan.isError || !plan.data) return <button className="retry-card" onClick={() => plan.refetch()}>题目没有加载出来，点这里再试一次</button>
  if (!current) return <div className="skeleton-block">正在收好答题卡……</div>
  return <section className="practice-page page-enter">
    <div className="practice-top"><span>第 {current.seq} 题 / {plan.data.plan.targetCount} 题</span><div className="progress-track"><i style={{ width: `${((current.seq - 1) / plan.data.plan.targetCount) * 100}%` }} /></div></div>
    <div className="question-stage">
      <div className="question-prompt"><p className="eyebrow">观察 · 思考 · 选择</p><h1>{current.question.stem}</h1><div className="question-specimen"><span>{String(current.question.visual.emoji ?? '🔎')}</span><small>{current.title}</small></div>
      </div>
      <div className="option-grid">{current.question.options.map((option, index) => {
        const selected = feedback && index === feedback.answerIndex
        return <button key={index} disabled={answer.isPending || (!!feedback && !feedback.canRetry)}
          className={`option-button${selected ? ' correct-answer' : ''}`}
          onClick={() => { setFeedback(null); answer.mutate({ item: current, option: index }) }}>
          {option.emoji && <span>{option.emoji}</span>}<strong>{option.label ?? `选项 ${index + 1}`}</strong>
        </button>
      })}</div>
    </div>
    <div className={`feedback-bar${feedback ? ' show' : ''}`}>{feedback && <>{feedback.correct ? '发现正确！✨' : feedback.canRetry ? '再观察一次，你快找到了' : '答案已经揭晓，下次会认出来的'}{feedback.explanation && <span>{feedback.explanation}</span>}</>}</div>
  </section>
}
