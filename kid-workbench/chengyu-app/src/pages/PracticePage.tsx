import { useEffect, useRef, useState } from 'react'
import { Navigate, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChengyuPlayer, type ChengyuExample, type ChengyuKind } from '@kid-workbench/chengyu-player'
import '@kid-workbench/chengyu-player/player.css'
import { appPath } from '../appPath'
import { chengyuApi } from '../api/chengyu'
import type { ChengyuCode, PlanItem } from '../api/types'
import { PracticeStage } from '../components/PracticeStage'
import { chengyuTypeNames } from '../content/chengyuTypes'
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

function toExample(item: PlanItem): ChengyuExample {
  const kind = item.question.code as ChengyuKind
  const blanked = item.question.visual?.blanked || (kind === 'example' ? item.question.visual?.text : '')
  return {
    kind,
    stem: item.question.stem,
    prompt: item.question.visual?.text || (kind === 'pick' ? item.meaning : '') || (kind === 'pinyin' ? item.pinyin : '') || blanked,
    speech: item.question.speech?.text || item.chengyu,
    speechUrl: item.question.speech?.url,
    options: item.question.options.map((option, index) => ({ id: option.id || String(index), label: option.label })),
    chengyu: item.chengyu,
    pinyin: item.pinyin,
    meaning: item.meaning,
    example: item.example,
    blank: item.question.visual?.full && item.question.visual?.blanked
      ? {
          full: item.question.visual.full,
          blanked: item.question.visual.blanked,
          target: item.question.visual.target || item.chengyu,
          start: item.question.visual.start ?? 0,
          length: item.question.visual.length ?? 0,
        }
      : undefined,
  }
}

export function PracticePage() {
  const planId = Number(useParams().planId)
  const n = Number(useParams().n || 1)
  const childId = useChildStore((s) => s.childId)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const plan = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => chengyuApi.plan(childId, planId) })
  const [picked, setPicked] = useState<string | null>(null)
  const startedAt = useRef(Date.now())
  const advanceTimer = useRef(0)
  const getClientId = usePendingAnswerStore((s) => s.get)
  const clearClientId = usePendingAnswerStore((s) => s.clear)
  const items = plan.data?.items ?? []
  const current = items[n - 1]

  useEffect(() => () => window.clearTimeout(advanceTimer.current), [])
  useEffect(() => {
    startedAt.current = Date.now()
    setPicked(current ? pickedId(current) || null : null)
  }, [current?.id, current?.status, current?.picks])

  const answer = useMutation({
    mutationFn: ({ item, option }: { item: PlanItem; option: number }) => {
      const key = `${planId}:${item.id}:${item.tries + 1}`
      return chengyuApi.answer(childId, planId, item.id, { clientId: getClientId(key), optionIndex: option, costMs: Date.now() - startedAt.current })
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

  if (plan.isLoading) return <section className="practice-page"><div className="skeleton-block">正在准备成语……</div></section>
  if (plan.isError || !plan.data) return <section className="practice-page"><button className="retry-card" onClick={() => plan.refetch()}>题目没有加载出来，点这里再试一次</button></section>
  if (!Number.isInteger(n) || n < 1) return <Navigate to={`/practice/${planId}/1`} replace />
  if (items.length && n > items.length) return <Navigate to={`/practice/${planId}/result`} replace />
  if (!current) return <section className="practice-page"><div className="skeleton-block">正在准备成语……</div></section>

  const code: ChengyuCode = current.question.code
  const total = items.length || plan.data.plan.targetCount || 1
  const prevHref = n > 1 ? `/practice/${planId}/${n - 1}` : undefined
  const nextHref = n < total ? `/practice/${planId}/${n + 1}` : `/practice/${planId}/result`
  const nextLabel = n < total ? '下一题' : '查看结果'
  const locked = current.status !== 'pending' || answer.isPending || picked !== null

  return (
    <PracticeStage label={chengyuTypeNames[code]} current={n} total={total} prevHref={prevHref} nextHref={nextHref} nextLabel={nextLabel}>
      <ChengyuPlayer
        example={toExample(current)}
        locked={locked}
        selectedId={picked || undefined}
        resolveAssetUrl={(url) => appPath(url)}
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
