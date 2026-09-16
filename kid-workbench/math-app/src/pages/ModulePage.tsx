import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { learningAudioURL, useCreatePlan, useMathModule, useMathStage } from '../api/math'
import type { ModuleCode, QuestionCode, StageCode } from '../api/types'
import { mathAudio } from '../audio/controller'
import { ShapeGlyph } from '../components/ShapeGlyph'
import { useChildStore } from '../store/childStore'

const defaultStage: Record<ModuleCode, StageCode> = { add10: 'within5', sub10: 'within5', shape: 'basic-shapes' }
const skillNames: Array<[QuestionCode, string]> = [['calc', '算一算'], ['story', '听故事'], ['find', '找答案'], ['name', '说名称']]

export function ModulePage() {
  const moduleCode = (useParams().moduleCode ?? 'add10') as ModuleCode
  const childId = useChildStore((state) => state.childId)
  const module = useMathModule(moduleCode).data
  const [selected, setSelected] = useState<StageCode>(defaultStage[moduleCode])
  const items = useMathStage(moduleCode, selected).data
  const createPlan = useCreatePlan(childId)
  const navigate = useNavigate()
  const start = () => {
    void mathAudio.unlock().catch(() => undefined)
    createPlan.mutate({ kind: 'module', moduleCode, stageCode: selected }, { onSuccess: (detail) => navigate(`/practice/${detail.plan.id}`) })
  }
  const play = (kpId: number, code: QuestionCode) => mathAudio.play(learningAudioURL(childId, kpId, code))

  return <section className={`page module-page ${moduleCode}`}>
    <header className="page-header"><Link className="back-link" to="/map">← 地图</Link><div><p className="eyebrow">EXPLORE & PRACTICE</p><h1>{module?.name ?? '模块学习'}</h1><p>先看一看、听一听，再开始练习。</p></div><button className="primary-button" onClick={start} disabled={createPlan.isPending}>练一练 →</button></header>
    <nav className="stage-tabs" aria-label="选择阶段">{module?.stages.map((stage) => <button key={stage.code} className={selected === stage.code ? 'active' : ''} onClick={() => setSelected(stage.code)}>{stage.name}<small>{stage.itemCount} 个</small></button>)}</nav>
    <div className={moduleCode === 'shape' ? 'shape-gallery' : 'fact-grid'}>{items?.map((item) => <article className="explore-card" key={item.kpId}>
      {item.kind === 'shape' && item.shape ? <ShapeGlyph shape={item.shape} /> : <div className="fact">{item.a} <span>{item.kind === 'add' ? '+' : '−'}</span> {item.b}</div>}
      <h2>{item.title}</h2>
      <div className="audio-actions">{skillNames.filter(([code]) => item.kind === 'shape' ? ['find', 'name'].includes(code) : ['calc', 'story'].includes(code)).map(([code, label]) => <button key={code} onClick={() => play(item.kpId, code)}>🔊 {label}</button>)}</div>
    </article>)}</div>
  </section>
}
