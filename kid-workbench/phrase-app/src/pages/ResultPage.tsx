import { useEffect, useRef } from 'react'
import { X } from '@phosphor-icons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { phraseApi } from '../api/phrase'
import type { PlanItem } from '../api/types'
import { useChildStore } from '../store/childStore'

function pickedLabel(item: PlanItem) {
  const value = item.picks.trim()
  if (!value) return '未选'
  const byId = item.question.options.find((option) => option.id === value)
  if (byId) return byId.label
  const index = Number(value.split(',').at(-1))
  if (Number.isInteger(index)) return item.question.options[index]?.label ?? `选项 ${index + 1}`
  return value
}

export function ResultPage() {
  const planId = Number(useParams().planId)
  const childId = useChildStore((s) => s.childId)
  const queryClient = useQueryClient()
  const detail = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => phraseApi.plan(childId, planId) })
  const finish = useMutation({
    mutationFn: () => phraseApi.finish(childId, planId),
    onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ['plan', childId, planId] }) },
  })
  const plan = detail.data?.plan
  const items = detail.data?.items ?? []
  const retryCode = items[0]?.question.code
  const finishStarted = useRef(false)

  useEffect(() => {
    if (!detail.data || detail.data.plan.status === 'done' || finishStarted.current) return
    finishStarted.current = true
    finish.mutate()
  }, [detail.data, finish])

  return (
    <section className="result-page result-list-page page-enter">
      <Link className="top-icon top-icon-close result-exit" to="/" aria-label="退出练习"><span aria-hidden="true"><X weight="bold" /></span></Link>
      <p className="eyebrow">今天的短句完成了</p>
      <h1>答题结果</h1>
      <p className="result-count">答完 {plan?.doneCount ?? '—'} 题，答对 {plan?.correctCount ?? '—'} 题</p>
      <ol className="result-list">
        {items.map((item) => (
          <li key={item.id} className={item.status === 'correct' ? 'is-correct' : item.status === 'wrong' ? 'is-wrong' : 'is-skipped'}>
            <span className="result-seq">第 {item.seq} 题</span>
            <strong>{item.phrase}</strong>
            <span>你选了 {pickedLabel(item)}</span>
            <b>{item.status === 'correct' ? '答对' : item.status === 'wrong' ? '答错' : '未作答'}</b>
          </li>
        ))}
      </ol>
      <div className="result-actions">
        <Link className="primary-button" to={retryCode ? `/practice/type/${retryCode}` : '/'}>再练一次</Link>
        <Link className="secondary-button" to="/">回到首页</Link>
      </div>
    </section>
  )
}
