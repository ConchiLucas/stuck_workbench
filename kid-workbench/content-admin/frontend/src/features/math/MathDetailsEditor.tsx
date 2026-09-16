import { appPath } from '../../appPath'
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { MathPlayer, shapeKey, shapeNames, type MathExample } from '@kid-workbench/math-player'
import '@kid-workbench/math-player/player.css'
import { generateMathDetailAudio, listMathDetails, publishMathDetail, saveMathDetail, type AdminMathCatalog, type AdminMathDetail } from '../../api/mathDetails'
import './mathDetails.css'

const groupLabels={addition:'加法',subtraction:'减法',shape:'图形'}
export function MathDetailsEditor(){
 const query=useQuery({queryKey:['math','details'],queryFn:listMathDetails,retry:false})
 const client=useQueryClient()
 const [selected,setSelected]=useState<AdminMathDetail|null>(null)
 function update(detail:AdminMathDetail){
  setSelected(detail)
  client.setQueryData<AdminMathCatalog>(['math','details'],old=>old?{...old,items:old.items.map(item=>item.id===detail.id?detail:item)}:old)
 }
 return <section className="math-details-editor" aria-label="题型详情素材">
  <div className="math-details-heading"><div><h2>题型详情素材</h2><p>编辑示例、保存草稿，再发布到孩子端与题目后台。</p></div><span>{query.data?.items.length??0} 种题型</span></div>
  {query.isPending&&<p>正在加载详情素材…</p>}
  {query.isError&&<div role="alert">{query.error.message}<button onClick={()=>void query.refetch()}>重新加载</button></div>}
  {query.data?.items.length===0&&<p>暂无详情素材，请检查素材初始化。</p>}
  <div className="math-detail-catalog">{query.data?.items.map(item=><button key={item.id} className={selected?.id===item.id?'active':''} onClick={()=>setSelected(item)}><strong>{groupLabels[item.groupId]} · {item.title}</strong><small>草稿 v{item.revision} · {item.publishedRevision?`已发布 v${item.publishedRevision}`:'尚未发布'}</small></button>)}</div>
  {selected&&<DetailForm key={selected.id} initial={selected} onSaved={update} onReload={async()=>{const fresh=await query.refetch();const detail=fresh.data?.items.find(item=>item.id===selected.id);if(!detail)throw new Error('无法重新加载已保存版本');update(detail);return detail}}/>}
 </section>
}
function StringList({label,values,onChange}:{label:string;values:string[];onChange:(values:string[])=>void}){
 return <fieldset><legend>{label}</legend>{values.map((value,index)=><div className="math-detail-array" key={index}><input aria-label={`${label} ${index+1}`} value={value} onChange={event=>onChange(values.map((old,i)=>i===index?event.target.value:old))}/><button type="button" aria-label={`删除${label} ${index+1}`} onClick={()=>onChange(values.filter((_,i)=>i!==index))}>删除</button></div>)}<button type="button" onClick={()=>onChange([...values,''])}>添加{label}</button></fieldset>
}
function DetailForm({initial,onSaved,onReload}:{initial:AdminMathDetail;onSaved:(value:AdminMathDetail)=>void;onReload:()=>Promise<AdminMathDetail>}){
 const [draft,setDraft]=useState(initial)
 const [saved,setSaved]=useState(initial)
 const [busy,setBusy]=useState(false)
 const [error,setError]=useState('')
 const [notice,setNotice]=useState('')
 const dirty=JSON.stringify(draft)!==JSON.stringify(saved)
 const e=draft.example
 const example=(change:Partial<MathExample>)=>{setDraft({...draft,example:{...e,...change}});setNotice('')}
 async function act(kind:'save'|'publish'|'audio'|'reload'){
  setBusy(true);setError('');setNotice('')
  try{
   const value=kind==='reload'?await onReload():kind==='save'?await saveMathDetail(draft):kind==='audio'?await generateMathDetailAudio(saved):await publishMathDetail(saved)
   const next={...value,publishedRevision:kind==='publish'?value.revision:value.publishedRevision??saved.publishedRevision}
   setDraft(next);setSaved(next);onSaved(next)
   setNotice(kind==='save'?'草稿已保存':kind==='publish'?`版本 ${next.revision} 已发布`:kind==='audio'?'题干音频已生成，请预览后发布':'已加载保存的版本')
  }catch(cause){setError(cause instanceof Error?cause.message:'操作失败')}finally{setBusy(false)}
 }
 const answerOptions=e.options?.length?e.options:e.kind==='judgement'?e.statements?.length===1?['对','错']:e.statements??[]:[]
 function updateShape(index:number,key:string){
  if(e.kind==='shape-name'){
   const glyphs:Record<string,string>={circle:'○',oval:'◯',square:'□',triangle:'△',rect:'▭',rhombus:'◇',star:'☆',trapezoid:'梯形'}
   const answer=shapeNames[key]
   example({shapeKeys:[key],prompt:glyphs[key]??answer,answer,options:[...new Set((e.options??[]).map(value=>value===e.answer?answer:value))],audioUrl:undefined})
   return
  }
  const shapeKeys=[...(e.shapeKeys??[])];shapeKeys[index]=key
  const options=[...(e.options??[])];const wasAnswer=options[index]===e.answer;options[index]=key
  example({shapeKeys,options,...(wasAnswer?{answer:key,...(e.kind==='audio-shape'?{prompt:`听到：${shapeNames[key]}`,audioUrl:undefined}:{})}:{})})
 }
 function updateAnswer(answer:string){
  example({answer,...(e.kind==='audio-shape'?{prompt:`听到：${shapeNames[shapeKey(answer)]??answer}`,audioUrl:undefined}:{})})
 }

 return <div className="math-detail-workspace">
  <form onSubmit={event=>{event.preventDefault();void act('save')}} className="math-detail-form">
   <h3>{groupLabels[draft.groupId]} · {draft.title}</h3><p>草稿版本 {saved.revision} · {saved.publishedRevision?`已发布版本 ${saved.publishedRevision}`:'尚未发布'}{dirty?' · 有未保存修改':''}</p>
   <label>标题<input value={draft.title} onChange={event=>setDraft({...draft,title:event.target.value})}/></label>
   <label>模块名称<input value={draft.moduleTitle} onChange={event=>setDraft({...draft,moduleTitle:event.target.value})}/></label>
   <label>学习目标<textarea value={draft.learningGoal} onChange={event=>setDraft({...draft,learningGoal:event.target.value})}/></label>
   <StringList label="规则" values={draft.rules} onChange={rules=>setDraft({...draft,rules})}/>
   <label>题干<textarea value={e.prompt} onChange={event=>example({prompt:event.target.value,audioUrl:undefined})}/></label>
   {e.kind==='judgement'&&<StringList label="判断算式" values={e.statements??[]} onChange={statements=>example({statements})}/>}
   {e.kind!=='shape-sort'&&<>{!['audio-shape','shape-feature'].includes(e.kind)&&<StringList label="选项" values={e.options??[]} onChange={options=>example({options})}/>}<label>正确答案<select value={e.answer??''} onChange={event=>updateAnswer(event.target.value)}><option value="">请选择</option>{answerOptions.map((option,index)=><option key={index} value={option}>{option}</option>)}</select></label></>}
   {draft.groupId!=='shape'&&<fieldset><legend>数量参数</legend><label>运算<select value={e.operation??(draft.groupId==='addition'?'add':'sub')} onChange={event=>example({operation:event.target.value as 'add'|'sub'})}><option value="add">合起来（加法）</option><option value="sub">拿走（减法）</option></select></label>{[0,1].map(index=><label key={index}>{index===0?'第一个数量':'第二个数量'}<input type="number" min="0" max="20" value={e.counts?.[index]??0} onChange={event=>{const counts=[...(e.counts??[0,0])];counts[index]=Number(event.target.value);example({counts})}}/></label>)}{e.kind==='objects'&&<label>物品<select value={e.object??'dot'} onChange={event=>example({object:event.target.value})}><option value="apple">苹果</option><option value="star">星星</option><option value="dot">圆点</option></select></label>}</fieldset>}
   {['audio-shape','shape-name','shape-feature'].includes(e.kind)&&<fieldset><legend>图形</legend>{Array.from({length:e.kind==='shape-name'?1:(e.options?.length??0)},(_,index)=><label key={index}>{e.kind==='shape-name'?'目标图形':`选项图形 ${index+1}`}<select value={e.shapeKeys?.[index]??''} onChange={event=>updateShape(index,event.target.value)}><option value="">请选择图形</option>{Object.entries(shapeNames).filter(([key])=>key!=='rectangle').map(([key,name])=><option key={key} value={key}>{name}</option>)}</select></label>)}</fieldset>}
   {e.kind==='shape-sort'&&<fieldset><legend>分类目标</legend>{e.buckets?.map((bucket,index)=><div key={index} className="math-detail-bucket"><label>分组名称 {index+1}<input value={bucket.label} onChange={event=>example({buckets:e.buckets?.map((old,i)=>i===index?{...old,label:event.target.value}:old)})}/></label>{bucket.items.map((item,j)=><label key={j}>图形 {j+1}<select value={shapeKey(item)} onChange={event=>example({buckets:e.buckets?.map((old,i)=>i===index?{...old,items:old.items.map((value,k)=>k===j?event.target.value:value)}:old)})}>{Object.entries(shapeNames).filter(([key])=>key!=='rectangle').map(([key,name])=><option key={key} value={key}>{name}</option>)}</select><button type="button" onClick={()=>example({buckets:e.buckets?.map((old,i)=>i===index?{...old,items:old.items.filter((_,k)=>k!==j)}:old)})}>删除图形</button></label>)}<button type="button" onClick={()=>example({buckets:e.buckets?.map((old,i)=>i===index?{...old,items:[...old.items,'circle']}:old)})}>添加图形</button><button type="button" onClick={()=>example({buckets:e.buckets?.filter((_,i)=>i!==index)})}>删除分组</button></div>)}<button type="button" onClick={()=>example({buckets:[...(e.buckets??[]),{label:'新分组',items:[]}]})}>添加分组</button></fieldset>}
   <label>题干音频地址<input value={e.audioUrl??''} onChange={event=>example({audioUrl:event.target.value||undefined})}/></label>
   {error&&<div role="alert" className="error-panel">{error}</div>}{notice&&<p className="info-panel">{notice}</p>}
   <div className="math-detail-actions"><button disabled={busy} type="submit">保存草稿</button><button disabled={busy||dirty} type="button" onClick={()=>void act('audio')}>生成题干音频</button><button disabled={busy||dirty} type="button" onClick={()=>void act('publish')}>发布此版本</button><button disabled={busy} type="button" onClick={()=>void act('reload')}>重新加载已保存版本</button></div>
  </form>
  <aside className="math-detail-preview"><h3>同版题面预览</h3><p>此处试做不会记录孩子作答。</p><MathPlayer detail={draft} resolveAssetUrl={appPath}/></aside>
 </div>
}
