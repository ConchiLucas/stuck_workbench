import { X } from '@phosphor-icons/react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { pinyinApi } from '../api/pinyin'
import type { PlanItem } from '../api/types'
import { useChildStore } from '../store/childStore'

function pickedLabel(item: PlanItem) {
  const parts = item.picks.split(',').map((part) => part.trim()).filter(Boolean)
  const last = parts.at(-1)
  if (last === undefined) return '未选'
  const index = Number(last)
  if (!Number.isInteger(index)) return last
  return item.question.options[index]?.label ?? `选项 ${index + 1}`
}

export function ResultPage() {
  const planId = Number(useParams().planId)
  const childId = useChildStore((s) => s.childId)
  const detail = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => pinyinApi.plan(childId, planId) })
  const plan = detail.data?.plan
  const items = detail.data?.items ?? []
  return <section className="result-page result-list-page page-enter">
    <Link className="top-icon top-icon-close result-exit" to="/" aria-label="退出练习"><span aria-hidden="true"><X weight="bold" /></span></Link>
    <p className="eyebrow">今天的拼音完成了</p>
    <h1>答题结果</h1>
    <p className="result-count">答完 {plan?.doneCount ?? '—'} 题，答对 {plan?.correctCount ?? '—'} 题</p>
    <ol className="result-list">
      {items.map((item) => <li key={item.id} className={item.status === 'correct' ? 'is-correct' : item.status === 'wrong' ? 'is-wrong' : 'is-skipped'}>
        <span className="result-seq">第 {item.seq} 题</span>
        <strong>{item.letter}</strong>
        <span>你选了 {pickedLabel(item)}</span>
        <b>{item.status === 'correct' ? '答对' : item.status === 'wrong' ? '答错' : '未作答'}</b>
      </li>)}
    </ol>
    <div className="result-actions"><Link className="primary-button" to="/">再练一次</Link></div>
  </section>
}
