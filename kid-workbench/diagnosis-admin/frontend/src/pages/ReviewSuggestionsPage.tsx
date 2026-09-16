import { useEffect,useRef,useState } from 'react'
import { useInfiniteQuery,useQuery } from '@tanstack/react-query'
import { Link,useLocation,useNavigate,useSearchParams } from 'react-router-dom'
import { knowledge,queryString,skillLabel,KnowledgeError,reasonLabel,timeLabel } from '../api/knowledge'
import { requestKey,saveSuggestion } from '../api/reviewSuggestions'
import type { Candidate,Page,Suggestion } from '../api/knowledgeTypes'
import { getChildId } from '../store/childStore'
import { LoadState,MoreButton } from '../components/knowledge/Shared'
import { compactChoices,readReviewDraft,refreshReviewChoices,writeReviewDraft,type ReviewChoice } from '../lib/reviewDraft'
import { BackToList, useListReturn } from '../lib/listNavigation'
export const generationLabel=(s:string|null)=>({queued:'等待生成',running:'正在生成',succeeded:'草稿已生成',partial:'部分生成',blocked:'暂不能生成',failed:'生成失败',cancelled:'已取消'}[s||'']||'尚未生成')
const stageLabel:Record<string,string>={pending:'待生成',draft:'草稿待发布',awaiting:'已发布 · 待作答',answered:'已有真实作答'}
type Pending={body:unknown;key:string}
function readPending(child:number):Pending|null {try{const p=JSON.parse(sessionStorage.getItem(`knowledge-review-pending:${child}`)||'null');return p&&typeof p.key==='string'&&p.body&&Array.isArray(p.body.targets)?p:null}catch{return null}}
export function ReviewSuggestionsPage(){
 const location=useLocation()
 const source=(location.state as {listReturn?:{to:string}}|null)?.listReturn?.to
 const sourceParams=new URLSearchParams(source?.startsWith('/wrongs?')?source.split('?')[1]:'')
 const child=getChildId(),navigate=useNavigate(),returnState=useListReturn(),[params]=useSearchParams(),subject=params.get('subject')||'',kpId=params.get('kpId')||''
 const [chosen,setChosen]=useState<Record<string,ReviewChoice>>({}),[busy,setBusy]=useState(false),[error,setError]=useState(''),[notice,setNotice]=useState(''),[ready,setReady]=useState(false),[restoreError,setRestoreError]=useState(''),[restoreAttempt,setRestoreAttempt]=useState(0),[pending,setPending]=useState<Pending|null>(()=>readPending(child))
 const savedSuccessfully=useRef(false)
 const candidates=useInfiniteQuery({queryKey:['knowledge','candidates',child,subject,kpId],initialPageParam:'',queryFn:({pageParam})=>knowledge<Page<Candidate>>('/review-candidates'+queryString({subject,kpId,cursor:pageParam})),getNextPageParam:p=>p.hasMore?p.nextCursor:undefined})
 const saved=useInfiniteQuery({queryKey:['knowledge','suggestions',child],initialPageParam:'',queryFn:({pageParam})=>knowledge<Page<Suggestion>>('/review-suggestions'+queryString({cursor:pageParam})),getNextPageParam:p=>p.hasMore?p.nextCursor:undefined})
 const savedRows=saved.data?.pages.flatMap(p=>p.items)||[], ids=savedRows.filter(s=>s.lifecycle!=='archived').map(s=>s.id)
 const stages=useQuery({queryKey:['knowledge','review-stages',child,ids.join(',')],enabled:ids.length>0,queryFn:async()=>{const chunks=[];for(let i=0;i<ids.length;i+=100)chunks.push(ids.slice(i,i+100));const results=await Promise.all(chunks.map(batch=>knowledge<{items:{id:number;stage:string}[]}>(`/review-stages?ids=${batch.join(',')}`)));return results.flatMap(r=>r.items)}})
 useEffect(()=>{
  let active=true;setReady(false);setRestoreError('');setChosen({});savedSuccessfully.current=false;setPending(readPending(child))
  const items=readReviewDraft(child)
  if(!items.length){setReady(true);return}
  refreshReviewChoices(child,items).then(result=>{if(!active)return;setChosen(result.choices);setReady(true);setNotice(result.missing?`已恢复可用选择，${result.missing} 项候选已变化，请重新确认。`:'已恢复上次未保存的复习选择。')}).catch(e=>{if(active)setRestoreError((e as Error).message)})
  return()=>{active=false}
 },[child,restoreAttempt])
 useEffect(()=>{if(ready&&!savedSuccessfully.current)writeReviewDraft(child,compactChoices(chosen))},[child,chosen,ready])
 useEffect(()=>{try{const key=`knowledge-review-pending:${child}`;if(pending)sessionStorage.setItem(key,JSON.stringify(pending));else sessionStorage.removeItem(key)}catch{/* Keep retry state in memory. */}},[child,pending])
 const selected=Object.values(chosen),total=selected.reduce((n,t)=>n+t.count,0),selectedSubject=selected[0]?.candidate.subjectCode
 const disabledReason=(c:Candidate)=>chosen[c.key]?'':selectedSubject&&selectedSubject!==c.subjectCode?'本次已选择其他学科，请先清空再切换。':total>=20?'本次已达到 20 题上限。':''
 const toggle=(c:Candidate)=>{setError('');setPending(null);if(disabledReason(c))return;setChosen(old=>{const n={...old};if(n[c.key])delete n[c.key];else n[c.key]={candidate:c,count:Math.min(3,20-total),mode:'mixed'};return n})}
 const save=async()=>{
  if(!selected.length&&!pending)return
  setError('');setBusy(true)
  try{
   let request=pending
   if(!request){
    const fresh=await refreshReviewChoices(child,compactChoices(chosen));setChosen(fresh.choices)
    if(fresh.missing){setNotice('部分候选已变化，已更新选择，请确认后再保存。');return}
    const current=Object.values(fresh.choices);if(new Set(current.map(x=>x.candidate.subjectCode)).size!==1)throw new Error('每份建议请选择同一学科的内容。')
    const n=current.reduce((v,x)=>v+x.count,0);if(n<1||n>20)throw new Error('每份建议总题数需在 1–20 题之间。')
    const body={schemaVersion:1,title:`${current.map(t=>t.candidate.title).join('、')}复习`,analysisVersion:'knowledge-analysis-v1',analysisAsOf:fresh.asOf,subjectCode:current[0].candidate.subjectCode,targets:current.map(({candidate:c,count,mode})=>({key:c.key,kpId:c.kpId,questionType:c.questionType,reasonCode:c.reasonCode,mode,requestedCount:count,preferredDistractorKpIds:c.preferredDistractorKpIds,evidence:c.evidence}))}
    request={body,key:requestKey()};setPending(request)
   }
   const s=await saveSuggestion(request.body,request.key)
   savedSuccessfully.current=true;writeReviewDraft(child,[]);try{sessionStorage.removeItem(`knowledge-review-pending:${child}`)}catch{/* Saving succeeded even when browser storage is unavailable. */}setPending(null);navigate(`/reviews/${s.id}`)
  }catch(e){setError((e as Error).message);if(e instanceof KnowledgeError&&e.status>=400&&e.status<500)setPending(null)}finally{setBusy(false)}
 }
 return <article>{source?.startsWith('/wrongs')&&<p className="notice"><BackToList fallback="/wrongs" label="返回来源错题"/> · {sourceParams.get('from')||'不限开始日期'} 至 {sourceParams.get('to')||'不限结束日期'}{sourceParams.get('followUpState')==='needs_practice'?' · 错误后尚无独立答对':''}。复习分析仍基于最近 30 天的完整依据。</p>}<header className="page-head"><div><p className="eyebrow">让每次复习有依据</p><h1>复习建议</h1><p className="lede">根据最近 30 天的练习，选出需要再练的能力。保存后，再交给题目后台生成草稿。</p></div></header>
 {restoreError&&<div className="error-box" role="alert">{restoreError}<button onClick={()=>setRestoreAttempt(x=>x+1)}>重试恢复选择</button></div>}{notice&&<p className="notice" role="status">{notice}</p>}
 <div className="review-layout"><section><h2>建议再练</h2><p className="muted">每种能力最多分析最近 20 个可核验题目实例。一次错误只作为观察。</p><LoadState loading={candidates.isPending} error={candidates.error}/>{candidates.data?.pages[0].items.length===0&&<div className="empty-panel">当前没有符合条件的复习候选。仍可在错题页查看全部历史记录。</div>}{candidates.data?.pages.flatMap(p=>p.items).map(c=><section key={c.key} className={`candidate-card${chosen[c.key]?' selected':''}`}><label className="candidate-heading" title={disabledReason(c)}><input type="checkbox" checked={!!chosen[c.key]} onChange={()=>toggle(c)} disabled={busy||!ready||!!disabledReason(c)}/><strong>{c.title}</strong><span>{skillLabel(c.questionType)}</span></label><p>{c.reasonText}</p><div className="evidence-links">{c.evidence.slice(0,4).map(e=><Link state={returnState()} to={`/attempts/${e.attemptId}`} key={e.attemptId}>作答 #{e.attemptId} ↗</Link>)}{c.evidence.length>4&&<span>等 {c.evidence.length} 条依据</span>}</div>{!c.reviewEligible&&<p className="notice">可保存分析；{(c.reviewBlockReasons||[]).map(reasonLabel).join('，')}</p>}</section>)}<MoreButton more={!!candidates.hasNextPage} loading={candidates.isFetchingNextPage} onClick={()=>void candidates.fetchNextPage()}/></section>
 <aside className="review-composer"><h2>本次复习</h2><p className="muted">每份同一学科，最多 20 题；还可选 {20-total} 题。</p>{!ready&&!restoreError?<p role="status">正在恢复复习选择…</p>:selected.length===0?<p className="muted">从左侧选择需要复习的内容。</p>:selected.map(({candidate:c,count,mode})=><div className="composer-target" key={c.key}><strong>{c.title}</strong><label>题数<input type="number" aria-label={`${c.title}题数`} min={1} max={Math.min(10,20-total+count)} value={count} disabled={busy} onChange={e=>{const value=Number(e.target.value);if(!Number.isInteger(value)||value<1||value>Math.min(10,20-total+count))return;setPending(null);setChosen(o=>({...o,[c.key]:{...o[c.key],count:value}}))}}/></label><select aria-label={`${c.title}复习方式`} value={mode} disabled={busy} onChange={e=>{setPending(null);setChosen(o=>({...o,[c.key]:{...o[c.key],mode:e.target.value}}))}}><option value="mixed">原题与变式</option><option value="original_only">只练原题</option></select></div>)}<p className="composer-total">{selected.length} 项能力 · {total} 题</p>{selected.length>0&&<button disabled={busy} onClick={()=>{setChosen({});setPending(null);setNotice('')}}>清空本次选择</button>}{error&&<p className="error-box" role="alert">{error}</p>}<button className="primary-button" disabled={busy||!ready||(!selected.length&&!pending)||total>20} onClick={()=>void save()}>{busy?'正在保存…':pending?'重试保存建议':'保存复习建议'}</button><small>保存建议不会发布题包或改变孩子的掌握状态。</small></aside></div>
 <section className="saved-reviews"><h2>已保存的建议</h2><LoadState loading={saved.isPending} error={saved.error}/>{savedRows.map(s=><Link state={returnState()} className="saved-review-row" to={`/reviews/${s.id}`} key={s.id}><strong>{s.title}</strong><span>{s.lifecycle==='archived'?'已归档':generationLabel(s.generationStatus)}</span><span>{s.lifecycle==='archived'?'':stages.isError?'发布／作答状态暂不可用':stages.data?.find(t=>t.id===s.id)?stageLabel[stages.data.find(t=>t.id===s.id)!.stage]:stages.isPending?'正在读取阶段…':'阶段信息已变化，请刷新'}</span><span>{s.generatedCount}/{s.requestedCount} 题</span><time>{timeLabel(s.createdAt)}</time></Link>)}{saved.data?.pages[0].items.length===0&&<p className="muted">还没有保存过复习建议。</p>}<MoreButton more={!!saved.hasNextPage} loading={saved.isFetchingNextPage} onClick={()=>void saved.fetchNextPage()}/></section></article>
}
