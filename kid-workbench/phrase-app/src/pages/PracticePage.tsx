import { useEffect, useRef, useState } from 'react'
import { Navigate, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PhrasePlayer, type PhraseExample, type PhraseKind } from '@kid-workbench/phrase-player'
import '@kid-workbench/phrase-player/player.css'
import { phraseApi } from '../api/phrase'
import type { PhraseCode, PlanItem } from '../api/types'
import { PracticeStage } from '../components/PracticeStage'
import { phraseTypeNames } from '../content/phraseTypes'
import { useChildStore } from '../store/childStore'
import { usePendingAnswerStore } from '../store/pendingAnswerStore'

const ADVANCE_MS = 450

function optionId(item: PlanItem, index: number) {
  const option = item.question.options[index]
  return option?.id || String(index)
}

function pickedId(item: PlanItem) {
  const value = item.picks.trim()
  if (!value || value.includes(',')) return ''
  if (item.question.options.some((option) => option.id === value)) return value
  const index = Number(value)
  if (Number.isInteger(index) && item.question.options[index]) return optionId(item, index)
  return ''
}

function toExample(item: PlanItem): PhraseExample {
  return {
    kind: item.question.code as PhraseKind,
    stem: item.question.stem,
    prompt: item.question.visual?.text || item.scene || item.replyTo,
    speech: item.question.speech?.text || item.phrase,
    speechUrl: item.question.speech?.url,
    options: item.question.options.map((option, index) => ({ id: option.id || String(index), label: option.label })),
    scene: item.scene,
    replyTo: item.replyTo,
  }
}

export function PracticePage() {
  const planId = Number(useParams().planId)
  const n = Number(useParams().n || 1)
  const childId = useChildStore((s) => s.childId)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const plan = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => phraseApi.plan(childId, planId) })
  const [picked, setPicked] = useState<string | null>(null)
  const [seenId, setSeenId] = useState<number | null>(null)
  const startedAt = useRef(Date.now())
  const advanceTimer = useRef(0)
  const getClientId = usePendingAnswerStore((s) => s.get)
  const clearClientId = usePendingAnswerStore((s) => s.clear)
  const items = plan.data?.items ?? []
  const current = items[n - 1]
  if (current && current.id !== seenId) {
    setSeenId(current.id)
    setPicked(pickedId(current) || null)
    startedAt.current = Date.now()
  }

  useEffect(() => () => window.clearTimeout(advanceTimer.current), [])

  const answer = useMutation({
    mutationFn: ({ item, option }: { item: PlanItem; option: number }) => {
      const key = `${planId}:${item.id}:1`
      return phraseApi.answer(childId, planId, item.id, { clientId: getClientId(key), optionIndex: option, costMs: Date.now() - startedAt.current })
        .then((result) => { clearClientId(key); return result })
    },
    onSuccess: async (_result, variables) => {
      const total = items.length || 1
      const index = items.findIndex((item) => item.id === variables.item.id)
      const at = index >= 0 ? index + 1 : n
      const nextHref = at < total ? `/practice/${planId}/${at + 1}` : `/practice/${planId}/result`
      await queryClient.invalidateQueries({ queryKey: ['plan', childId, planId] })
      window.clearTimeout(advanceTimer.current)
      advanceTimer.current = window.setTimeout(() => navigate(nextHref), ADVANCE_MS)
    },
    onError: () => setPicked(current ? pickedId(current) || null : null),
  })

  if (plan.isLoading) return <section className="practice-page"><div className="skeleton-block">正在准备今天的短句……</div></section>
  if (plan.isError || !plan.data) return <section className="practice-page"><button className="retry-card" onClick={() => plan.refetch()}>题目没有加载出来，点这里再试一次</button></section>
  if (!Number.isInteger(n) || n < 1) return <Navigate to={`/practice/${planId}/1`} replace />
  if (items.length && n > items.length) return <Navigate to={`/practice/${planId}/result`} replace />
  if (!current) return <section className="practice-page"><div className="skeleton-block">正在准备今天的短句……</div></section>

  const code = current.question.code
  const total = items.length || plan.data.plan.targetCount || 1
  const prevHref = n > 1 ? `/practice/${planId}/${n - 1}` : undefined
  const nextHref = n < total ? `/practice/${planId}/${n + 1}` : `/practice/${planId}/result`
  const nextLabel = n < total ? '下一题' : '查看结果'
  const answered = items.filter((item) => item.status !== 'pending' || Boolean(item.picks)).length
  const done = answered + (current.status === 'pending' && !current.picks && picked !== null ? 1 : 0)
  const locked = current.status !== 'pending' || answer.isPending || picked !== null

  return (
    <PracticeStage label={phraseTypeNames[code as PhraseCode]} done={done} total={total} prevHref={prevHref} nextHref={nextHref} nextLabel={nextLabel}>
      <PhrasePlayer
        example={toExample(current)}
        locked={locked}
        selectedId={picked || undefined}
        onSelect={(id) => {
          const index = current.question.options.findIndex((option, optionIndex) => (option.id || String(optionIndex)) === id)
          if (index < 0) return
          setPicked(id)
          answer.mutate({ item: current, option: index })
        }}
      />
    </PracticeStage>
  )
}
