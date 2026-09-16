import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { importSyllableRecordings, listSyllables, syllableAudioURL, updateSyllable, type PinyinSyllable } from '../../api/pinyin'

export function SyllableMaterials() {
 const client=useQueryClient()
 const query=useQuery({queryKey:['pinyin','syllables'],queryFn:listSyllables})
 const [closed,setClosed]=useState(false)
 const importing=useMutation({mutationFn:importSyllableRecordings,onSuccess:()=>client.invalidateQueries({queryKey:['pinyin']})})
 return <section className="literacy-group" aria-label="音节素材">
  <div className="group-header-row">
   <button type="button" className="group-header" aria-expanded={!closed} onClick={()=>setClosed(!closed)}><span>音节</span><span className="muted">{query.data?.total ?? 0} 音节 · {closed?'展开':'收起'}</span></button>
   <button type="button" className="mini-btn" disabled={importing.isPending} onClick={()=>importing.mutate()}>{importing.isPending?'导入中…':'导入音节真人包'}</button>
  </div>
  {importing.error ? <div role="alert" className="error-panel">{importing.error.message}</div>:null}
  {importing.data ? <div role="status" className="info-panel">导入 {importing.data.generated}，失败 {importing.data.failed}{importing.data.errors?.length?` · ${importing.data.errors.join('；')}`:''}</div>:null}
  {closed?null:query.isLoading?<div className="loading-panel">音节加载中…</div>:query.error?<div className="error-panel" role="alert">{query.error.message}<button type="button" className="mini-btn" onClick={()=>void query.refetch()}>重试</button></div>:!query.data?.items?.length?<div className="empty-panel">暂无音节素材。</div>:<div className="char-grid">{query.data.items.map(item=><SyllableCard key={item.id} item={item}/>)}</div>}
 </section>
}
function SyllableCard({item}:{item:PinyinSyllable}) {
 const client=useQueryClient()
 const [editing,setEditing]=useState(false)
 const [text,setText]=useState(item.speechText)
 const [playing,setPlaying]=useState(false)
 const [error,setError]=useState('')
 const audio=useRef<HTMLAudioElement|null>(null)
 const save=useMutation({mutationFn:({speechText,enabled}:{speechText:string;enabled:boolean})=>updateSyllable(item.id,speechText,enabled),onSuccess:()=>{setEditing(false);void client.invalidateQueries({queryKey:['pinyin','syllables']})}})
 useEffect(()=>()=>{audio.current?.pause()},[])
 async function play(){
  setError('');audio.current?.pause()
  const current=new Audio(syllableAudioURL(item));audio.current=current
  current.onended=()=>setPlaying(false)
  current.onerror=()=>{setPlaying(false);setError('录音播放失败，请重新导入真人包')}
  setPlaying(true)
  try {await current.play()}catch{setPlaying(false);setError('录音播放失败，请重新导入真人包')}
 }
 return <article className="char-card">
  <div className="char-glyph">{item.syllableText}</div>
  <div className="sense-tag">{item.initialText} + {item.finalText} · 第 {item.tone} 声</div>
  <div className="sense-tag">例字 {item.speechText} · {item.speechUrl?'真人录音已备':'缺少真人录音'} · {item.enabled?'已启用':'已停用'}</div>
  {editing?<form onSubmit={event=>{event.preventDefault();save.mutate({speechText:text,enabled:item.enabled})}}>
    <label>例字<input aria-label={`${item.syllableText}例字`} value={text} maxLength={20} required onChange={event=>setText(event.target.value)}/></label>
    <button type="submit" className="mini-btn" disabled={save.isPending}>保存</button><button type="button" className="mini-btn" onClick={()=>{setText(item.speechText);setEditing(false)}}>取消</button>
   </form>:null}
  <div className="card-actions">
   <button type="button" className="mini-btn" disabled={!item.speechUrl||playing} onClick={()=>void play()} aria-label={`试听音节 ${item.syllableText}`}>{playing?'播放中…':'试听录音'}</button>
   <button type="button" className="mini-btn" onClick={()=>{setText(item.speechText);setEditing(!editing)}}>编辑例字</button>
   <button type="button" className="mini-btn" disabled={save.isPending} onClick={()=>save.mutate({speechText:item.speechText,enabled:!item.enabled})}>{item.enabled?'停用':'审核启用'}</button>
  </div>
  {error||save.error?<div className="error-panel" role="alert">{error||save.error?.message}</div>:null}
 </article>
}
