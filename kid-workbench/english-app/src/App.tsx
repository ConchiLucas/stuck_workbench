import { appPath } from './appPath'
import {useEffect,useState} from 'react'
import {Link,Navigate,Route,Routes,useParams} from 'react-router-dom'
import {api} from './api/client'
import {IconClose,IconNext,IconPrev} from './icons'
import type {Plan} from './api/types'
import {questionTypes,type QuestionType} from './questionTypes'
import {demoHref,demoResultHref,questionDemos,quizToChoiceDemo} from './questionDemos'
import {practiceKind,ListenButton,QuestionTypePreview} from './QuestionTypePreview'
import {ResultPage} from './ResultPage'
import {usePendingAnswers} from './store/pendingAnswerStore'
import {useDemoAnswerStore} from './store/demoAnswerStore'
import {useLiveQuizStore,type LiveQuizCard} from './store/liveQuizStore'

function isLiveCard(id?:string):id is LiveQuizCard{
  return id==='audio-choice'||id==='image-text'
}

function PlayShell({children}:{children:React.ReactNode}){return <main className="play-shell">{children}</main>}
function KidBar({backTo,kind,current,total,prevTo,nextTo}:{backTo:string;kind:string;current:number;total:number;prevTo?:string;nextTo?:string}){
  const progress=total?Math.min(100,Math.max(0,(current/total)*100)):0
  return <nav className="kid-top" aria-label="练习导航">
    <Link className="kid-circle" to={backTo} aria-label="返回"><IconClose/></Link>
    <div className="kid-progress" role="progressbar" aria-label={`${kind}，第 ${current} / ${total} 题`} aria-valuemin={1} aria-valuemax={total} aria-valuenow={current}>
      <i style={{width:`${progress}%`}}/>
      <span className="kid-progress-label" aria-hidden="true">{current} / {total}</span>
    </div>
    <div className="kid-nav">
      {prevTo?<Link className="kid-circle" to={prevTo} aria-label="上一题"><IconPrev/></Link>:<span className="kid-circle is-muted" aria-hidden="true"><IconPrev/></span>}
      {nextTo?<Link className="kid-circle is-next" to={nextTo} aria-label="下一题"><IconNext/></Link>:<span className="kid-circle is-muted" aria-hidden="true"><IconNext/></span>}
    </div>
  </nav>
}

function TypeCard({item}:{item:QuestionType}){
  return <Link to={`/types/${item.id}`} className="type-card" aria-label={`查看题型：${item.kidTitle}`} onClick={()=>{
    useDemoAnswerStore.getState().clearType(item.id)
    if(isLiveCard(item.id)) useLiveQuizStore.getState().invalidate(item.id)
  }}>
    <img src={appPath(`/types/${item.id}.png`)} alt=""/>
    <strong>{item.kidTitle}</strong>
  </Link>
}

function HomePage(){
  return <section className="type-gallery-page" aria-label="题型">
    <div className="type-gallery">{questionTypes.map(item=><TypeCard item={item} key={item.id}/>)}</div>
  </section>
}

function QuestionTypePreviewPage(){
  const{id,n}=useParams()
  const item=questionTypes.find(x=>x.id===id)
  const live=isLiveCard(id)
  const ensure=useLiveQuizStore(s=>s.ensure)
  const useFallback=useLiveQuizStore(s=>s.useFallback)
  const liveQuestions=useLiveQuizStore(s=>live?s.questionsByType[id]:undefined)
  const loading=useLiveQuizStore(s=>live&&s.loadingType===id)
  const error=useLiveQuizStore(s=>s.error)
  const fallback=useLiveQuizStore(s=>live&&Boolean(s.fallback[id]))
  useEffect(()=>{if(live) void ensure(id)},[live,id,ensure])
  const localBank=id?questionDemos[id]:undefined
  const waiting=live&&!fallback&&!liveQuestions?.length
  const bank=waiting?undefined:fallback||!live?localBank:liveQuestions?.map(quizToChoiceDemo)
  const value=useDemoAnswerStore(s=>id?s.pick(id,n?Number(n):1):undefined)
  const setPick=useDemoAnswerStore(s=>s.setPick)
  if(!item||(!waiting&&!bank))return <Navigate to="/" replace/>
  if(waiting){
    return <PlayShell>
      <KidBar backTo="/" kind={practiceKind(item)} current={0} total={4}/>
      <section className="kid-stage" aria-label="当前题目">
        <div className="quiz-status">
          <p>{loading||!error?'出题中…':error}</p>
          {error&&!loading?<button type="button" className="kid-write-go" onClick={()=>useFallback(id)}>用示例题</button>:null}
        </div>
      </section>
    </PlayShell>
  }
  const current=n?Number(n):1
  if(!bank||!Number.isInteger(current)||current<1||current>bank.length)return <Navigate to={demoHref(item.id,1)} replace/>
  const demo=bank[current-1]
  const nextTo=current<bank.length?demoHref(item.id,current+1):demoResultHref(item.id)
  return <PlayShell>
    <KidBar backTo="/" kind={practiceKind(item)} current={current} total={bank.length} prevTo={current>1?demoHref(item.id,current-1):undefined} nextTo={nextTo}/>
    <QuestionTypePreview item={item} demo={demo} value={value} onChange={next=>setPick(item.id,current,next)} nextTo={nextTo}/>
  </PlayShell>
}

function PracticePage(){
  const{planId}=useParams()
  const[d,setD]=useState<Plan>()
  useEffect(()=>{api<Plan>(appPath(`/api/v1/children/1/english/plans/${planId}`)).then(setD)},[planId])
  const item=d?.items.find(x=>x.status==='pending')
  const pending=usePendingAnswers()
  async function answer(index:number){
    if(!item||!d)return
    const key=`${planId}:${item.id}:${item.tries+1}`
    await api(appPath(`/api/v1/children/1/english/plans/${planId}/items/${item.id}/answer`),{method:'POST',body:JSON.stringify({clientId:pending.get(key),optionIndex:index,costMs:1000})})
    pending.ack(key)
    const next=await api<Plan>(appPath(`/api/v1/children/1/english/plans/${planId}`))
    if(next.items.every(x=>x.status!=='pending')){await api(appPath(`/api/v1/children/1/english/plans/${planId}/finish`),{method:'POST'});next.plan.status='done'}
    setD(next)
  }
  if(d&&!item)return <PlayShell><section className="kid-stage kid-done"><h1>练习完成！</h1><p className="kid-note">{d.plan.doneCount} / {d.plan.targetCount}</p><Link className="kid-listen" to="/">回到首页</Link></section></PlayShell>
  return <PlayShell>
    <KidBar backTo="/" kind={item?.question.code==='picture'?'听音选图':'听音选词'} current={(d?.plan.doneCount??0)+1} total={d?.plan.targetCount??0}/>
    <section className="kid-stage" aria-label="当前题目">
      <div className="kid-board">
        <div className="kid-prompt">
          <h1>{item?.question.stem??'准备题目…'}</h1>
          {item&&<ListenButton onClick={()=>new Audio(appPath(`/api/v1/english/words/${item.kpId}/speech.mp3`)).play()}/>}
        </div>
        <div className="kid-work"><div className="kid-options">{item?.question.options.map((option,i)=><button className="kid-option" onClick={()=>answer(i)} key={`${option.kpId}-${i}`}>{option.assetUrl?<img src={option.assetUrl} alt=""/>:<strong>{option.label}</strong>}</button>)}</div></div>
      </div>
    </section>
  </PlayShell>
}

export default function App(){return <Routes><Route path="/" element={<HomePage/>}/><Route path="/types" element={<HomePage/>}/><Route path="/types/:id/result" element={<ResultPage/>}/><Route path="/types/:id/:n" element={<QuestionTypePreviewPage/>}/><Route path="/types/:id" element={<QuestionTypePreviewPage/>}/><Route path="/words/*" element={<Navigate to="/" replace/>}/><Route path="/practice/:planId" element={<PracticePage/>}/><Route path="*" element={<Navigate to="/" replace/>}/></Routes>}
