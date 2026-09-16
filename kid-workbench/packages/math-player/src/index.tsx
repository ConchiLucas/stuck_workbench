import { useState } from 'react'
import { KidPlayer, type KidAnswer } from './KidPlayer'
export type { KidAnswer } from './KidPlayer'

export interface MathExample {
  kind: 'choice' | 'missing' | 'objects' | 'judgement' | 'audio-shape' | 'shape-name' | 'shape-feature' | 'shape-sort'
  prompt: string
  options?: string[]
  answer?: string
  groups?: string[]
  statements?: string[]
  buckets?: { label: string; items: string[] }[]
  operation?: 'add' | 'sub'
  counts?: number[]
  object?: string
  shapeKeys?: string[]
  objectImageUrl?: string
  shapeImageUrls?: Record<string, string>
  audioUrl?: string
  optionIds?: string[]
  answerOptionId?: string
}
export interface MathDetail {
  id: string
  groupId: 'addition' | 'subtraction' | 'shape'
  title: string
  moduleTitle: string
  learningGoal: string
  rules: string[]
  revision: number
  example: MathExample
}
export interface MathCatalog { schemaVersion: 1; items: MathDetail[]; stale?: boolean }
export const shapeNames: Record<string, string> = {circle:'圆形',square:'正方形',rect:'长方形',rectangle:'长方形',triangle:'三角形',oval:'椭圆形',trapezoid:'梯形',rhombus:'菱形',star:'五角星'}
const aliases: Record<string,string> = {'○':'circle','◯':'oval','△':'triangle','□':'square','◇':'rhombus','▭':'rect','圆形':'circle','椭圆形':'oval','三角形':'triangle','正方形':'square','菱形':'rhombus','长方形':'rect'}
export const shapeKey = (value:string) => aliases[value] ?? value
export function ShapeGlyph({shape,className=''}:{shape:string;className?:string}) {
  const key=shapeKey(shape)
  return <svg className={`math-player-shape ${className}`} viewBox="0 0 120 120" role="img" aria-label={shapeNames[key] ?? shape} fill="currentColor" stroke="currentColor" strokeWidth="3" strokeLinejoin="round">
    {key==='circle' && <circle cx="60" cy="60" r="39"/>}
    {key==='square' && <rect x="22" y="22" width="76" height="76" rx="3"/>}
    {(key==='rect'||key==='rectangle') && <rect x="13" y="30" width="94" height="60" rx="3"/>}
    {key==='triangle' && <path d="M60 16 107 99H13Z"/>}
    {key==='oval' && <ellipse cx="60" cy="60" rx="48" ry="32"/>}
    {key==='trapezoid' && <path d="M31 24h58l20 73H11Z"/>}
    {key==='rhombus' && <path d="m60 10 49 50-49 50L11 60Z"/>}
    {key==='star' && <path d="m60 9 14 32 35 4-26 23 8 35-31-18-31 18 8-35-26-23 35-4Z"/>}
  </svg>
}
function Symbol({value}:{value:string}) {
  return shapeNames[shapeKey(value)] ? <ShapeGlyph shape={value}/> : <span>{value}</span>
}
export function MathPlayer({detail,example,className='',resolveAssetUrl=(url:string)=>url,mode='preview',readOnly=false,initialAnswer,onAnswer}:{mode?:'preview'|'kid';readOnly?:boolean;initialAnswer?:KidAnswer;onAnswer?:(answer:KidAnswer)=>void;detail?:MathDetail;example?:MathExample;className?:string;resolveAssetUrl?:(url:string)=>string}) {
  const content=detail?.example ?? example
  if(!content) return <p role="alert">暂无可用题面</p>
  if(mode==='kid') return <KidPlayer key={JSON.stringify(content)} example={content} resolveAssetUrl={resolveAssetUrl} initialAnswer={initialAnswer} readOnly={readOnly} onAnswer={onAnswer}/>
  return <Player key={JSON.stringify(content)} example={content} className={className} resolveAssetUrl={resolveAssetUrl}/>
}
function Player({example:e,className,resolveAssetUrl}:{example:MathExample;className:string;resolveAssetUrl:(url:string)=>string}) {
  const [feedback,setFeedback]=useState<boolean|null>(null)
  const [selected,setSelected]=useState('')
  const [placements,setPlacements]=useState<Record<string,string>>({})
  const [audioError,setAudioError]=useState('')
  const candidates=(e.buckets??[]).flatMap((bucket,group)=>bucket.items.map((item,index)=>({item,group,id:`${group}-${index}`}))).sort((a,b)=>a.item.localeCompare(b.item))
  const options=e.options?.length ? e.options : e.kind==='judgement' ? (e.statements?.length===1 ? ['正确','错误'] : e.statements??[]) : []
  const answer=e.answer??''
  function choose(value:string) {
    setSelected(value)
    setFeedback(value===answer || (e.kind==='judgement' && answer.startsWith(value+'，')))
  }
  const object=({apple:'苹果',star:'星星'} as Record<string,string>)[e.object??'']??e.object??'物品'
  const objectGlyph=object==='苹果'?'🍎':object==='星星'?'★':'●'
  const original=e.counts?.[0]??0
  const removed=e.counts?.[1]??0
  return <div className={`math-player ${className}`}>
    <div className="math-player-prompt" aria-label={e.prompt}>{e.kind==='shape-name' && e.shapeKeys?.[0] ? <ShapeGlyph shape={e.shapeKeys[0]}/> : e.prompt}</div>
    {e.audioUrl ? <audio controls src={resolveAssetUrl(e.audioUrl)} aria-label="播放题目读音" onError={()=>setAudioError('读音暂时无法播放，请查看题目文字。')}/> : e.kind==='audio-shape'?<p>题干音频尚未准备好，可以先看文字试做。</p>:null}
    {audioError && <p role="alert">{audioError}</p>}
    {e.kind==='objects' && (e.counts?.length ? <div className="math-player-objects">
      {e.operation==='sub' ? <div className="math-player-object-group" aria-label={`原来 ${original} 个，拿走 ${removed} 个`}>
        {Array.from({length:Math.min(100,Math.max(0,original))},(_,i)=><span key={i} className={i>=original-removed?'math-player-removed':''} aria-label={`${i>=original-removed?'拿走':'剩下'}的${object}`}>{objectGlyph}</span>)}
        <small>拿走 {removed} 个</small>
      </div> : e.counts.map((count,index)=><div key={index} className="math-player-object-group" aria-label={`${count} 个${object}`}>
        {index>0 && <b>＋</b>}{Array.from({length:Math.min(100,Math.max(0,count))},(_,i)=><span key={i}>{objectGlyph}</span>)}
      </div>)}
    </div> : <div className="math-player-objects">{e.groups?.map((group,i)=><span key={i}>{i>0?'＋ ':''}{group}</span>)}</div>)}
    {e.kind==='judgement' && e.statements?.length===1 && <p className="math-player-statement">{e.statements[0]}</p>}
    {e.kind==='shape-sort' ? <>
      <div className="math-player-sorting">{candidates.map(({item,id})=><label key={id}><Symbol value={item}/><select aria-label={`${shapeNames[shapeKey(item)]??item}放在哪一组`} value={placements[id]??''} onChange={event=>{setPlacements({...placements,[id]:event.target.value});setFeedback(null)}}>
        <option value="">选择分组</option>{e.buckets?.map((bucket,i)=><option key={i} value={String(i)}>{bucket.label}</option>)}
      </select></label>)}</div>
      <button type="button" className="math-player-check" disabled={candidates.some(c=>!placements[c.id])} onClick={()=>setFeedback(candidates.every(c=>placements[c.id]===String(c.group)))}>检查分类</button>
    </> : <div className="math-player-options">{options.map((option,index)=><button type="button" key={`${index}-${option}`} aria-pressed={selected===option} className={selected===option?'is-selected':''} onClick={()=>choose(option)}>
      {(e.kind==='audio-shape'||e.kind==='shape-feature') ? <Symbol value={e.shapeKeys?.[index]??option}/> : option}
    </button>)}</div>}
    {feedback!==null && <p role="status" className={feedback?'math-player-correct':'math-player-retry'}>{feedback?'答对了！':'再想一想，再试一次。'}</p>}
    <button type="button" className="math-player-reset" onClick={()=>{setFeedback(null);setSelected('');setPlacements({})}}>重新试做</button>
  </div>
}
