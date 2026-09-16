import {gatewayLink} from '../appPath'
import {LiteracyPlayer,type PlayerFeedback} from '@kid-workbench/literacy-player'
import {questionTypes} from '@kid-workbench/literacy-contract'
import {useEffect,useState,useRef,type ReactNode} from 'react'
import {Link,useNavigate,useParams} from 'react-router-dom'
import {useQuery,useQueryClient} from '@tanstack/react-query'
import {genAPI,GenerationAPIError,mediaURL,type Snapshot,type Task} from '../api/generation'
const typeNames:Record<string,string>=Object.fromEntries(questionTypes.map(t=>[t.code,t.label]))
function QuestionCard({q,taskId,seq,revisionId}:{q:Snapshot;taskId:number;seq:number;revisionId?:number}){return <LiteracyPlayer mode="preview" question={{id:`${taskId}:${revisionId}:${seq}`,questionType:q.questionType,interaction:q.interaction??(q.questionType==='write_char'?'handwriting':'choice'),prompt:q.prompt,stem:q.stem,options:q.options?.map(option=>({...option,audio:q.questionType==='glyph_sense'?option.audio:undefined}))}} mediaResolver={ref=>mediaURL(ref as import('../api/generation').Media)} onSubmit={response=>genAPI<PlayerFeedback>(`question-tasks/${taskId}/items/${seq}/preview-answer`,'POST',{revisionId,response})}/>}
export function GenerationPage(){
 const pendingActions=useRef(new Map<string,{key:string;body:object}>()),actionBusy=useRef(false)
 const {id}=useParams(),nav=useNavigate(),qc=useQueryClient();const [error,setError]=useState(''),[busy,setBusy]=useState(false)
 useEffect(()=>{setError('')},[id])
 const list=useQuery({queryKey:['generation-tasks'],queryFn:()=>genAPI<Task[]>('question-tasks?subject=literacy')})
 const detail=useQuery({queryKey:['generation-task',id],queryFn:()=>genAPI<Task>('question-tasks/'+id),enabled:!!id})
 const t=detail.data;const questions=t?.items
 const refresh=async()=>{await qc.invalidateQueries({queryKey:['generation-task']});await qc.invalidateQueries({queryKey:['generation-tasks']})}
 async function action(path:string,body:unknown={},method='POST'){
  if(actionBusy.current)return
  actionBusy.current=true;setBusy(true);setError('')
  const endpoint='question-tasks/'+id+'/'+path
  const generates=path==='generate'||path.endsWith('/replace')
  const pending=(generates?pendingActions.current.get(endpoint):undefined)??{key:crypto.randomUUID(),body:{expectedRowVersion:t?.rowVersion,...(body as object)}}
  if(generates)pendingActions.current.set(endpoint,pending)
  try{await genAPI(endpoint,method,pending.body,pending.key);pendingActions.current.delete(endpoint);await refresh()}
  catch(e){if(e instanceof GenerationAPIError&&e.code!=='generation_running')pendingActions.current.delete(endpoint);setError((e as Error).message)}
  finally{actionBusy.current=false;setBusy(false)}
 }
 const close=()=>nav('/',{replace:true})
 return <><section className="gen-page" aria-label="出题任务列表">{list.error&&<div role="alert" className="error-panel">{list.error.message}</div>}{list.isLoading?<p>正在读取任务…</p>:!list.data?.length?<div className="empty-panel">暂无出题任务。</div>:<div className="gen-task-list">{list.data.map(t=><Link className="gen-task-row" key={t.id} to={'/tasks/'+t.id}><span className="gen-task-kind">{t.sourceMode==='legacy_pool'?'旧题包':t.kind==='review'?'复习':'练习'}</span><div><strong>{t.title}</strong><small>{t.moduleName} · {t.targetCount} 题 · {new Date(t.updatedAt).toLocaleString('zh-CN')}</small></div><span>查看 →</span></Link>)}</div>}</section>
 {id&&<TaskPreviewDialog onClose={close}>{detail.isLoading?<p>正在读取题目…</p>:detail.error?<p role="alert" className="error-panel">{detail.error.message}</p>:t&&<>
 {(error||t.lastError)&&<div role="alert" className="error-panel">{error||t.lastError}</div>}
 {t.sourceReviewSuggestionId&&<p className="muted">来自孩子知识库复习建议 · <a href={`${gatewayLink(19211,'http://localhost:19211').replace(/\/$/,'')}/reviews/${t.sourceReviewSuggestionId}`} target="_blank" rel="noreferrer">查看错题依据 ↗</a></p>}
 {t.sourceMode==='material_template'&&t.kind==='practice'&&<ReviewJobs taskId={t.id}/>}
 {t.kind==='review'&&<ReviewEvidence taskId={t.id}/>}
 {!questions?.length&&<div className="empty-panel">暂无可查看的题目。</div>}
 <div className="gen-question-list">{questions?.map((v,i)=><article key={v.id} className="gen-question-card"><header><b>第 {i+1} 题 · {v.snapshot.targetText}</b>{t.status==='draft'&&<div><button className="mini-btn" disabled={busy||t.kind==='review'} onClick={()=>action('items/'+(i+1)+'/replace')}>换一道</button><button className="mini-btn" disabled={busy||i===0} onClick={()=>{const order=t.items.map((_,n)=>n+1);[order[i-1],order[i]]=[order[i],order[i-1]];void action('order',{order},'PUT')}}>上移</button></div>}</header><QuestionCard q={v.snapshot} taskId={t.id} seq={i+1} revisionId={t.activeRevisionId}/><details><summary>素材与出题来源</summary><p>题目版本 #{v.id}{v.sourceQuestionVersionId&&<> · 原题版本 #{v.sourceQuestionVersionId}</>}</p><p className="gen-ids">{v.snapshot.materialRevisionIds?.join(' · ')}</p></details></article>)}</div>
 </>}</TaskPreviewDialog>}</>
}
function TaskPreviewDialog({onClose,children}:{onClose:()=>void;children:ReactNode}){
 const dialog=useRef<HTMLDialogElement>(null)
 useEffect(()=>{
  const previous=document.activeElement as HTMLElement|null
  const overflow=document.body.style.overflow
  document.body.style.overflow='hidden'
  dialog.current?.showModal()
  return ()=>{document.body.style.overflow=overflow;dialog.current?.close();previous?.focus()}
 },[])
 return <dialog ref={dialog} className="task-preview-dialog" aria-label="题目查看" onCancel={event=>{event.preventDefault();onClose()}}>
  <div className="task-preview-bar"><button type="button" className="task-preview-close" aria-label="关闭弹窗" onClick={onClose} autoFocus>×</button></div>
  <div className="task-preview-content">{children}</div>
 </dialog>
}

function ReviewJobs({taskId}:{taskId:number}){
 const [error,setError]=useState(''),[busy,setBusy]=useState(false)
 const jobs=useQuery({queryKey:['review-jobs',taskId],queryFn:()=>genAPI<{id:number;state:string;attempts:number;error:string;taskId?:number;sourcePlanId:number}[]>('question-tasks/'+taskId+'/review-jobs')})
 const names:Record<string,string>={pending:'等待生成',running:'生成中',done:'已生成草稿',failed:'生成失败',skipped:'无需复习'}
 if(!jobs.data?.length&&!jobs.error)return null
 return <details className="gen-form"><summary>自动复习生成记录</summary>{(error||jobs.error)&&<p role="alert">{error||jobs.error?.message}</p>}{jobs.data?.map(j=><div className="gen-toolbar" key={j.id}><span>练习 #{j.sourcePlanId} · {names[j.state]??j.state} · 尝试 {j.attempts} 次</span>{j.taskId&&<Link to={'/tasks/'+j.taskId}>查看复习草稿</Link>}{j.error&&<span>{j.error}</span>}{j.state==='failed'&&<button disabled={busy} className="refresh-button" onClick={async()=>{setBusy(true);setError('');try{await genAPI('review-generation-jobs/'+j.id+'/retry','POST',{});await jobs.refetch()}catch(e){setError((e as Error).message)}finally{setBusy(false)}}}>重新尝试</button>}</div>)}</details>
}

type Evidence={receiptId:number;planId:number;sourceQuestionVersionId:number;createdAt:string;targetText:string;questionType:string;selectedOptionId?:string;selectedOption?:Snapshot['options'][number];correctOption?:Snapshot['options'][number];responseKind?:string;answerPayload?:{kind?:string;strokes?:{x:number;y:number;t:number}[][];hintsUsed?:number};evaluation?:{outcome?:string;assistance?:string;evaluatorVersion?:string}}
function ReviewEvidence({taskId}:{taskId:number}){
 const evidence=useQuery({queryKey:['review-evidence',taskId],queryFn:()=>genAPI<Evidence[]>('question-tasks/'+taskId+'/review-evidence')})
 return <details className="gen-form"><summary>复习出题依据 · 实际作答</summary>{evidence.isLoading&&<p>正在读取作答依据…</p>}{evidence.error&&<p role="alert">{evidence.error.message}</p>}{evidence.data?.map(row=><article key={row.receiptId} className="gen-evidence"><p><strong>{row.targetText} · {typeNames[row.questionType]}</strong> · 练习 #{row.planId} · {new Date(row.createdAt).toLocaleString('zh-CN')}</p>{row.responseKind==='handwriting'||row.answerPayload?.kind==='handwriting'||row.questionType==='write_char'?<HandwritingEvidence row={row}/>:<div className="gen-evidence-options"><div>当时选了：{row.selectedOption?.text??'未保存具体选项'}{row.selectedOption?.image&&<img src={mediaURL(row.selectedOption.image)} alt={'当时错选：'+row.selectedOption.text}/>}</div><div>正确选项：{row.correctOption?.text??'来源答案缺失'}{row.correctOption?.image&&<img src={mediaURL(row.correctOption.image)} alt={'正确选项：'+row.correctOption.text}/>}</div></div>}<small className="muted">作答回执 #{row.receiptId} · 原题版本 #{row.sourceQuestionVersionId}</small></article>)}{evidence.data?.length===0&&<p>没有保存可展示的作答依据。</p>}</details>
}
function HandwritingEvidence({row}:{row:Evidence}){
 const strokes=row.answerPayload?.strokes??[],hinted=row.evaluation?.assistance==='hinted'||(row.answerPayload?.hintsUsed??0)>0
 return <div className="gen-evidence-writing"><svg aria-label="当时的书写笔迹" role="img" viewBox="0 0 1 1" width="220" height="220" style={{background:'#fffaf2',border:'1px solid #d4b896',maxWidth:'100%'}}><path d="M .5 0 V 1 M 0 .5 H 1" stroke="#d4b896" strokeWidth=".003"/>{strokes.map((stroke,i)=><polyline key={i} points={stroke.filter(p=>Number.isFinite(p.x)&&Number.isFinite(p.y)).map(p=>`${p.x},${p.y}`).join(' ')} fill="none" stroke="#1f1812" strokeWidth=".028" strokeLinecap="round" strokeLinejoin="round"/>)}</svg>{!strokes.length&&<p>这次提交没有笔迹。</p>}<p>标准字：{row.targetText}</p><p>{row.evaluation?.outcome==='passed'?'通过':row.evaluation?.outcome==='not_passed'?'未通过':'评估结果未保存'} · {hinted?'使用过提示':'未使用提示'}</p>{row.evaluation?.evaluatorVersion&&<small>评估版本：{row.evaluation.evaluatorVersion}</small>}</div>
}
