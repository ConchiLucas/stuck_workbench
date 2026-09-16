import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { MathPlayer, type MathDetail } from '@kid-workbench/math-player'
import '@kid-workbench/math-player/player.css'
import { mathTasksAPI, type MathTaskItem } from '../api/mathTasks'
import { appPath } from '../appPath'
import './mathTasks.css'

const groups=[{id:'addition',name:'加法'},{id:'subtraction',name:'减法'},{id:'shape',name:'图形'}] as const
export function MathTasksPage(){
 const qc=useQueryClient()
 const [openId,setOpenId]=useState<number|null>(null),[creating,setCreating]=useState(false),[busy,setBusy]=useState(false),[error,setError]=useState('')
 const [title,setTitle]=useState(''),[count,setCount]=useState(8),[rangeMax,setRangeMax]=useState(10),[detailIds,setDetailIds]=useState<string[]>([])
 const submitting=useRef(false)
 const list=useQuery({queryKey:['math-tasks'],queryFn:()=>mathTasksAPI.list()})
 const detail=useQuery({queryKey:['math-task',openId],queryFn:()=>mathTasksAPI.get(openId!),enabled:openId!==null})
 const materials=useQuery({queryKey:['math-materials'],queryFn:()=>mathTasksAPI.materials(),enabled:creating})
 async function generate(){
  if(submitting.current)return
  submitting.current=true;setBusy(true);setError('')
  try{const task=await mathTasksAPI.create({title,detailIds,rangeMax,count});setCreating(false);setOpenId(task.id);await qc.invalidateQueries({queryKey:['math-tasks']})}
  catch(e){setError((e as Error).message)}finally{submitting.current=false;setBusy(false)}
 }
 return <>
  <section className="gen-page" aria-label="算术出题任务列表">
   <div className="math-task-actions"><button className="mini-btn" onClick={()=>{setError('');setCreating(true)}}>生成算术题目</button></div>
   {list.isLoading&&<p>正在读取任务…</p>}
   {list.error&&<p role="alert" className="error-panel">{list.error.message}</p>}
   {!list.isLoading&&!list.error&&!list.data?.items.length&&<div className="empty-panel">暂无算术出题任务。</div>}
   <div className="gen-task-list">{list.data?.items.map(task=><button className="gen-task-row math-task-row" key={task.id} onClick={()=>setOpenId(task.id)}><span className="gen-task-kind">练习</span><div><strong>{task.title}</strong><small>{(task.titles??[]).join(' / ')||(task.groups??[]).join(' / ')} · {task.groups?.join(' / ')} · {task.count} 题 · {new Date(task.createdAt).toLocaleString('zh-CN')}</small></div><span>查看 →</span></button>)}</div>
  </section>
  {creating&&<Dialog onClose={()=>{if(!busy)setCreating(false)}} label="生成算术题目">
   <form className="gen-form math-task-form" onSubmit={event=>{event.preventDefault();void generate()}}>
    <label>任务名称<input maxLength={80} value={title} placeholder="算术练习" onChange={e=>setTitle(e.target.value)} disabled={busy}/></label>
    <label>数值范围<select value={rangeMax} onChange={e=>setRangeMax(Number(e.target.value))} disabled={busy}><option value={5}>5 以内</option><option value={10}>10 以内</option><option value={20}>20 以内</option></select></label>
    <fieldset disabled={busy||materials.isLoading}>{groups.map(group=><div key={group.id} className="math-task-type-group"><legend>{group.name}</legend>{(materials.data?.items??[]).filter(d=>d.groupId===group.id).map(d=><label key={d.id}><input type="checkbox" checked={detailIds.includes(d.id)} onChange={()=>setDetailIds(prev=>prev.includes(d.id)?prev.filter(id=>id!==d.id):[...prev,d.id])}/>{d.title}</label>)}</div>)}</fieldset>
    <label>题数<input type="number" min={Math.max(1,detailIds.length)} max={100} value={count} onChange={e=>setCount(Number(e.target.value))} disabled={busy}/></label>
    <p className="muted">使用素材后台已发布的算术详情；生成后保存原题、稳定选项和媒体。听音图形缺少音频时会说明原因。试答不会记入孩子学习记录。</p>
    {materials.error&&<p role="alert" className="error-panel">{materials.error.message}</p>}
    {error&&<p role="alert" className="error-panel">{error}</p>}
    <button className="mini-btn" disabled={busy||!!materials.error||!detailIds.length||count<detailIds.length||count>100||!Number.isInteger(count)}>{busy?'正在生成并保存…':'生成并保存'}</button>
   </form>
  </Dialog>}
  {openId!==null&&<Dialog onClose={()=>setOpenId(null)} label="题目查看">
   {detail.isLoading?<p>正在读取题目…</p>:detail.error?<p role="alert" className="error-panel">{detail.error.message}</p>:!detail.data?.items?.length?<p className="empty-panel">暂无可查看的题目。</p>:<div className="gen-question-list">{detail.data.items.map((q,i)=><article key={`${openId}:${q.sequence}`} className="gen-question-card"><header><b>第 {i+1} 题 · {q.detail.title}</b></header><Preview item={q}/><details><summary>素材与出题来源</summary><p>{q.source.title} · {q.sourceId} · 修订 {q.sourceRevision}</p><p>题目与选项按生成时保存；试答不会记入孩子学习记录。</p><ul>{Object.entries(q.mediaSHA256??{}).map(([path,hash])=><li className="math-media-source" key={path}>{path}<small>媒体摘要 {hash}</small></li>)}</ul></details></article>)}</div>}
  </Dialog>}
 </>
}
function Preview({item}:{item:MathTaskItem}){
 return <div className="math-task-player"><MathPlayer mode="kid" detail={item.detail as MathDetail} resolveAssetUrl={url=>url?appPath(url):url}/></div>
}
function Dialog({onClose,label,children}:{onClose:()=>void;label:string;children:ReactNode}){
 const ref=useRef<HTMLDialogElement>(null)
 useEffect(()=>{const previous=document.activeElement as HTMLElement|null;const overflow=document.body.style.overflow;document.body.style.overflow='hidden';const dialog=ref.current;dialog?.showModal();dialog?.querySelector<HTMLButtonElement>('.task-preview-close')?.focus();return()=>{document.body.style.overflow=overflow;dialog?.close();previous?.focus()}},[])
 return <dialog ref={ref} className="task-preview-dialog" aria-label={label} onCancel={event=>{event.preventDefault();onClose()}}><div className="task-preview-bar"><button className="task-preview-close" aria-label="关闭弹窗" onClick={onClose}>×</button></div><div className="task-preview-content">{children}</div></dialog>
}
