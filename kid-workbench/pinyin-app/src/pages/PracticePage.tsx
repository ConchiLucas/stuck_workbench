import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { PinyinQuestion, type PinyinQuestionType } from '@kid-workbench/pinyin-player'
import { pinyinApi } from '../api/pinyin'
import type { PlanItem } from '../api/types'
import { PracticeStage } from '../components/PracticeStage'
import { useChildStore } from '../store/childStore'
import { usePendingAnswerStore } from '../store/pendingAnswerStore'

function visualText(visual: Record<string, unknown>) {
  return typeof visual.text === 'string' ? visual.text : ''
}

function planQuestionType(code: string): PinyinQuestionType {
  return code === 'inword' ? 'inword' : 'listen'
}

export function PracticePage() {
  const planId = Number(useParams().planId)
  const childId = useChildStore((s) => s.childId)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const plan = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => pinyinApi.plan(childId, planId) })
  const [picked, setPicked] = useState<string | undefined>()
  const startedAt = useRef(Date.now())
  const getClientId = usePendingAnswerStore((s) => s.get)
  const clearClientId = usePendingAnswerStore((s) => s.clear)
  const current = useMemo(() => plan.data?.items.find((item) => item.status === 'pending'), [plan.data])

  useEffect(() => { startedAt.current = Date.now(); setPicked(undefined) }, [current?.id])
  useEffect(() => {
    if (!plan.data) return
    if (plan.data.plan.status === 'done') {
      navigate(`/practice/${planId}/result`, { replace: true })
      return
    }
    if (current) return
    pinyinApi.finish(childId, planId).then(() => navigate(`/practice/${planId}/result`, { replace: true }))
  }, [childId, current, navigate, plan.data, planId])

  const answer = useMutation({
    mutationFn: ({ item, option }: { item: PlanItem; option: number }) => {
      const key = `${planId}:${item.id}:${item.tries + 1}`
      return pinyinApi.answer(childId, planId, item.id, { clientId: getClientId(key), optionIndex: option, costMs: Date.now() - startedAt.current })
        .then((result) => { clearClientId(key); return result })
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['plan', childId, planId] }),
        queryClient.invalidateQueries({ queryKey: ['home', childId] }),
        queryClient.invalidateQueries({ queryKey: ['progress', childId] }),
      ])
    },
  })

  if (plan.isLoading) return <section className="practice-page"><div className="skeleton-block">正在准备今天的拼音……</div></section>
  if (plan.isError || !plan.data) return <section className="practice-page"><button className="retry-card" onClick={() => plan.refetch()}>题目没有加载出来，点这里再试一次</button></section>
  if (!current) return <section className="practice-page"><div className="skeleton-block">正在收好答题卡……</div></section>
  const type = planQuestionType(current.question.code)
  const kind = type === 'inword' ? 'word' : 'solo'
  const typeName = type === 'inword' ? '字中找拼音' : '听音选字母'
  const stimulus = visualText(current.question.visual) || (type === 'inword' ? current.letter : '')
  return <PracticeStage label={typeName} done={plan.data.items.filter((item) => item.status !== 'pending').length} total={plan.data.plan.targetCount}>
    <section className="question-workspace" aria-label="当前题目">
      <PinyinQuestion
        question={{
          id: String(current.id),
          type,
          stem: current.question.stem,
          speechUrl: pinyinApi.speechUrl(current.kpId, kind),
          visual: { kind: type === 'inword' ? 'char' : 'sound', text: stimulus },
          options: current.question.options.map((option, index) => ({ id: String(index), label: option.label })),
        }}
        selectedOptionId={picked}
        disabled={answer.isPending}
        allowSyntheticSpeech={false}
        onPick={(id) => {
          if (answer.isPending) return
          setPicked(id)
          answer.mutate({ item: current, option: Number(id) })
        }}
      />
    </section>
  </PracticeStage>
}
