import { useState } from 'react'
import type { KidAnswer } from '@kid-workbench/math-player'
import './mathDetail.css'
import { appPath } from '../appPath'
import { Link, useParams } from 'react-router-dom'
import { MathPlayer } from '@kid-workbench/math-player'
import '@kid-workbench/math-player/player.css'
import { usePublishedMathDetails } from '../api/details'
import { getQuestionTypePack } from '../content/questionTypePrototype'

export function QuestionTypeDetailPage() {
  const [answers,setAnswers]=useState<Record<string,KidAnswer>>({})
  const params=useParams<{typeId:string;type:string;n:string}>()
  const pack=getQuestionTypePack(params.typeId??params.type??'addition-equation')
  const typeId=params.typeId??pack?.questionIds[Math.max(0,Number(params.n??1)-1)]??pack?.questionIds[0]??''
  const query=usePublishedMathDetails()
  const items=query.data?.items??[]
  const detail=items.find(item=>item.id===typeId)
  const questions=pack?.questionIds.flatMap(id=>items.filter(item=>item.id===id))??[]
  const index=questions.findIndex(item=>item.id===typeId)
  const previous=questions[index-1]
  const next=questions[index+1]
  if(query.isPending) return <section className="type-detail-page math-detail-page"><p>正在加载题目…</p></section>
  if(query.isError) return <section className="type-detail-page math-detail-page"><p role="alert">{query.error.message}</p><button onClick={()=>void query.refetch()}>重新加载</button><Link to="/">返回首页</Link></section>
  if(!detail) return <section className="type-detail-page math-detail-page type-not-found"><h1>这个题型还没有发布素材</h1><p>回首页选择其他题型吧。</p><Link to="/">返回首页</Link></section>
  return <section className={`type-detail-page math-detail-page ${detail.groupId}`}>
    <header className="math-detail-nav">
      <Link className="math-detail-icon" to="/" aria-label="退出练习">×</Link>
      <div className="math-detail-progress" role="progressbar" aria-label={`第 ${index+1} 题，共 ${questions.length} 题`} aria-valuemin={0} aria-valuemax={questions.length} aria-valuenow={index+1}><i style={{width:`${(index+1)/Math.max(1,questions.length)*100}%`}}/><span>{index+1} / {questions.length}</span></div>
      <nav aria-label="浏览题目">{previous ? <Link className="math-detail-icon" to={`/types/${previous.id}`} aria-label="上一题">‹</Link> : <button className="math-detail-icon" disabled aria-label="上一题">‹</button>}{next ? <Link className="math-detail-icon" to={`/types/${next.id}`} aria-label="下一题">›</Link> : <button className="math-detail-icon" disabled aria-label="下一题">›</button>}</nav>
    </header>
    <h1 className="sr-only">{detail.title}</h1>
    {query.data?.stale && <p role="status">素材服务暂时不可用，正在显示上次发布的内容。</p>}
    <MathPlayer key={`${detail.id}:${detail.revision}`} mode="kid" detail={detail} resolveAssetUrl={appPath} initialAnswer={answers[`${detail.id}:${detail.revision}`]} onAnswer={answer=>setAnswers(current=>({...current,[`${detail.id}:${detail.revision}`]:answer}))}/>
    <div className="math-detail-footer" hidden={!answers[`${detail.id}:${detail.revision}`]?.correct}>{answers[`${detail.id}:${detail.revision}`]?.correct && (next ? <Link className="math-detail-continue" to={`/types/${next.id}`}>继续下一题 →</Link> : <Link className="math-detail-continue" to="/">完成，回到首页 ✓</Link>)}</div>
  </section>
}
