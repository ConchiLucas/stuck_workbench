import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { literacyApi } from '../api/literacy'
import type { AnswerResult, PlanItem } from '../api/types'
import { PracticeStage } from '../components/PracticeStage'
import {LiteracyPlayer,type PlayerQuestion,type PlayerResponse} from '@kid-workbench/literacy-player'
import { useChildStore } from '../store/childStore'
import { usePendingAnswerStore } from '../store/pendingAnswerStore'

export function PracticePage() {
  const planId = Number(useParams().planId)
  const childId = useChildStore((s) => s.childId)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const plan = useQuery({ queryKey: ['plan', childId, planId], queryFn: () => literacyApi.plan(childId, planId) })
  const [feedback, setFeedback] = useState<AnswerResult | null>(null)
  const startedAt = useRef(Date.now())
  const advanceTimer = useRef<number>(0)
  const getClientId = usePendingAnswerStore((s) => s.get)
  const clearClientId = usePendingAnswerStore((s) => s.clear)
  const items = plan.data?.items ?? []
  const firstPending = items.findIndex((item) => item.status === 'pending')
  const [cursor, setCursor] = useState<number | null>(null)
  const index = cursor ?? Math.max(0, firstPending)
  const current = items[index]

  useEffect(() => {
    startedAt.current = Date.now()
    setFeedback(null)
  }, [current?.id])

  useEffect(() => () => window.clearTimeout(advanceTimer.current), [])

  useEffect(() => {
    if (!plan.data) return
    if (plan.data.plan.status === 'done') {
      navigate(`/practice/${planId}/result`, { replace: true })
      return
    }
    if (plan.data.items.some((item) => item.status === 'pending')) return
    literacyApi.finish(childId, planId).then(() => navigate(`/practice/${planId}/result`, { replace: true }))
  }, [childId, current, navigate, plan.data, planId])

  const answer = useMutation({
    mutationFn: ({ item, response }: { item: PlanItem; response: PlayerResponse }) => {
      const key = `${planId}:${item.id}:${JSON.stringify(response)}`
      return literacyApi.answer(childId, planId, item.id, { clientId: getClientId(key), ...(item.question.snapshot?.responseSchemaVersion === 2 || response.kind === 'handwriting' ? {response} : {optionIndex:item.question.options.findIndex((o,i)=>(o.id??String(i))===response.selectedOptionId)}), costMs: Date.now() - startedAt.current })
        .then((result) => { clearClientId(key); return result })
    },
    onSuccess: (result) => {
      setFeedback(result)
      if (!result.correct && result.canRetry) return
      window.clearTimeout(advanceTimer.current)
      advanceTimer.current = window.setTimeout(() => {
        setCursor(null)
        void Promise.all([
          queryClient.invalidateQueries({ queryKey: ['plan', childId, planId] }),
          queryClient.invalidateQueries({ queryKey: ['home', childId] }),
          queryClient.invalidateQueries({ queryKey: ['progress', childId] }),
        ])
      }, 1200)
    },
  })

  if (plan.isLoading) return <section className="practice-page"><div className="skeleton-block">正在准备今天的汉字……</div></section>
  if (plan.isError || !plan.data) return <section className="practice-page"><button className="retry-card" onClick={() => plan.refetch()}>题目没有加载出来，点这里再试一次</button></section>
  if (!current) return <section className="practice-page"><div className="skeleton-block">正在收好字帖……</div></section>
  const legacyWriting=current.status==='pending'&&!current.question.versionId&&(current.question.code==='write_char'||current.question.type==='write')
  const question = toPlayerQuestion(current)
  const locked=current.status!=='pending'||answer.isPending||!!(feedback&&!feedback.canRetry)
  return <PracticeStage current={index+1} total={items.length} onPrev={()=>setCursor(Math.max(0,index-1))} onNext={()=>setCursor(Math.min(items.length-1,index+1))}>
    {legacyWriting?<div className="skeleton-block"><p>这道旧听写题缺少冻结的书写模板，无法使用新版判题。</p><p>原有学习记录会保留，请从新版听写任务继续练习。</p><Link className="primary-button" to="/tasks?questionType=write_char">前往新版听写任务</Link></div>:<LiteracyPlayer key={current.id} question={question} disabled={locked} mediaResolver={ref=>typeof ref==='string'?literacyApi.frozenUrl(ref):literacyApi.revisionUrl(ref as import('../api/types').MediaRef)} onSubmit={async response=>{const result=await answer.mutateAsync({item:current,response});return {...result,answerOptionId:result.answerOptionId??(result.answerIndex===undefined?undefined:question.options?.[result.answerIndex]?.id)}}}/>}
  </PracticeStage>
}
export function toPlayerQuestion(item:PlanItem):PlayerQuestion{
 const q=item.question,versioned=!!q.versionId,write=q.code==='write_char'||q.type==='write',glyph=q.code==='glyph_sense'
 return {id:item.id,questionType:glyph?'glyph_sense':undefined,interaction:write?'handwriting':'choice',prompt:versioned?q.stem:(write?'听音写字':undefined),stem:versioned?(q.snapshot?.stem??{}):{image:write?undefined:glyph?literacyApi.glyphUrl(item.kpId):literacyApi.senseUrl(item.kpId),audio:literacyApi.speechUrl(item.kpId)},options:write?undefined:(q.options??[]).map((o,i)=>({id:o.id??String(i),text:o.label,image:versioned?o.image:o.kpId?(glyph?literacyApi.senseUrl(o.kpId):literacyApi.glyphUrl(o.kpId)):o.image,audio:versioned?o.audio:o.kpId?literacyApi.speechUrl(o.kpId):undefined}))}
}
