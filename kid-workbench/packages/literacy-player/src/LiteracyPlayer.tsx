export {GlyphSenseQuestion} from './GlyphSenseQuestion'
import {GlyphSenseQuestion} from './GlyphSenseQuestion'
export {materialReasonLabel} from './materialReasons'
import {useEffect,useRef,useState,type ReactNode} from 'react'
import {HandwritingQuestion} from './HandwritingQuestion'
import './player.css'
export type InkPoint={x:number;y:number;t:number}
export type PlayerResponse={kind:'choice';selectedOptionId:string}|{kind:'handwriting';strokes:InkPoint[][];hintsUsed:number}
export type PlayerFeedback={correct:boolean;canRetry?:boolean;answerOptionId?:string}
export type PlayerQuestion={id:string|number;questionType?:string;interaction:'choice'|'handwriting';prompt?:string;stem:{text?:string;image?:unknown;audio?:unknown};options?:{id:string;text?:string;image?:unknown;audio?:unknown}[]}
export type PlayerProps={question:PlayerQuestion;mode?:'practice'|'preview';mediaResolver:(ref:unknown)=>string|undefined;onSubmit:(response:PlayerResponse)=>Promise<PlayerFeedback>;disabled?:boolean;children?:ReactNode}
export function LiteracyPlayer(props:PlayerProps){return <PlayerSession key={props.question.id} {...props}/>}
function PlayerSession({question:q,mode='practice',mediaResolver:resolve,onSubmit,disabled,children}:PlayerProps){
 const [feedback,setFeedback]=useState<PlayerFeedback>(),[pending,setPending]=useState(false),[error,setError]=useState(''),[picked,setPicked]=useState(''),[retry,setRetry]=useState(0),[round,setRound]=useState(0)
 const inFlight=useRef(false),last=useRef<PlayerResponse|undefined>(undefined)
 const audio=useRef<HTMLAudioElement|null>(null),playback=useRef(0)
 const locked=!!disabled||pending||!!(feedback?.correct||feedback&&!feedback.canRetry)
 async function submit(response:PlayerResponse){if(inFlight.current||locked)return;inFlight.current=true;setPending(true);setError('');setFeedback(undefined);last.current=response;if(response.kind==='choice')setPicked(response.selectedOptionId);try{setFeedback(await onSubmit(response))}catch(e){setError(e instanceof Error?e.message:'作答暂时无法保存')}finally{inFlight.current=false;setPending(false)}}
 function stopAudio(){playback.current++;const previous=audio.current;audio.current=null;if(previous){previous.pause();previous.currentTime=0}}
 async function play(ref:unknown){stopAudio();const request=playback.current;try{const url=resolve(ref);if(!url)throw new Error('读音素材缺失');const next=new Audio(url);audio.current=next;await next.play()}catch{if(request===playback.current)setError('读音无法播放，请重试')}}
 useEffect(()=>{if(mode==='practice'&&q.interaction==='handwriting')void play(q.stem.audio);return stopAudio},[q.id,mode])
 function reset(){if(inFlight.current||disabled)return;setFeedback(undefined);setPicked('');setError('');last.current=undefined;setRound(v=>v+1)}
 const feedbackView=<div className={`lp-feedback${feedback?.correct?' lp-success':feedback?' lp-miss':''}`} aria-live="polite">{q.interaction==='handwriting'?(pending?'正在看你写的字……':feedback?(feedback.correct?'答对啦':'不太像，再听一次再写'):'先听读音，再写到格子里'):(pending?'正在保存…':feedback?(feedback.correct?'答对啦':feedback.canRetry?'再试一次':'记住这个字'):'')}</div>
 const media=(ref:unknown)=>{const url=resolve(ref);return url?<img key={`${url}:${retry}`} src={url} alt="" onError={()=>setError('图片无法显示，请重试')}/>:<span role="alert">图片素材缺失</span>}
 return <section className={`literacy-player${q.questionType==='glyph_sense'?' lp-glyph-host':''}`} data-mode={mode} aria-label="识字练习">
 {q.prompt&&<p className="lp-prompt">{q.prompt}</p>}
 {q.questionType==='glyph_sense'?<GlyphSenseQuestion question={q} mediaResolver={resolve} selectedOptionId={picked} correct={feedback?.correct} disabled={locked} onPick={id=>void submit({kind:'choice',selectedOptionId:id})} feedback={pending?'正在检查…':feedback&&!feedback.correct&&!feedback.canRetry?'记住这个字':undefined}/>:q.interaction==='handwriting'?<HandwritingQuestion key={round} locked={locked} onSubmit={submit} onPlay={()=>play(q.stem.audio)} onClear={reset} feedback={feedbackView} listenIcon={<Sound/>}/>:<div className="lp-choice">
 <div className="lp-stage-visual"><div className="lp-tianzige">{q.stem.image?media(q.stem.image):q.stem.text?<strong>{q.stem.text}</strong>:<span role="alert">题干素材缺失</span>}</div>{!!q.stem.audio&&<button className="lp-listen" aria-label="播放读音" onClick={()=>play(q.stem.audio)}><Sound/></button>}</div>
 <div className="lp-choice-row">{feedbackView}<div className="lp-option-grid">{q.options?.map((o,i)=><div key={o.id} role="button" tabIndex={locked?-1:0} aria-label={`选项 ${i+1}`} aria-disabled={locked} aria-pressed={picked===o.id} className={`lp-option${o.image?' lp-picture':''}${(feedback?.answerOptionId===o.id||feedback?.correct&&picked===o.id)?' lp-correct':feedback&&!feedback.correct&&picked===o.id?' lp-wrong':''}`} onClick={()=>void submit({kind:'choice',selectedOptionId:o.id})} onKeyDown={e=>{if(e.target!==e.currentTarget)return;if(e.key==='Enter'||e.key===' '){e.preventDefault();void submit({kind:'choice',selectedOptionId:o.id})}}}>{o.image?media(o.image):<strong>{o.text}</strong>}{!!o.audio&&<button className="lp-option-play" aria-label={`播放选项 ${i+1} 读音`} onClick={e=>{e.stopPropagation();void play(o.audio)}}><Sound size={18}/></button>}</div>)}</div></div></div>}
 {error&&<div className="lp-error" role="alert">{error}<button onClick={()=>{setError('');setRetry(v=>v+1)}}>重试素材</button>{last.current&&<button disabled={pending} onClick={()=>void submit(last.current!)}>重试提交</button>}</div>}
 {mode==='preview'&&feedback&&<button className="lp-reset" disabled={pending||disabled} onClick={reset}>再试一次</button>}{children}
 </section>
}
// Same light SpeakerHigh geometry used by the original App, without a host icon dependency.
function Sound({size=22}:{size?:number}){return <svg width={size} height={size} viewBox="0 0 256 256" fill="currentColor" aria-hidden="true"><path d="M154.64,26.61a6,6,0,0,0-6.32.65L77.94,82H32A14,14,0,0,0,18,96v64a14,14,0,0,0,14,14H77.94l70.38,54.74A6,6,0,0,0,158,224V32A6,6,0,0,0,154.64,26.61ZM30,160V96a2,2,0,0,1,2-2H74v68H32A2,2,0,0,1,30,160Zm116,51.73L86,165.07V90.93l60-46.66Zm50.53-108.85a38,38,0,0,1,0,50.24,6,6,0,1,1-9-7.94,26,26,0,0,0,0-34.37,6,6,0,0,1,9-7.93ZM246,128a77.86,77.86,0,0,1-19.86,52,6,6,0,1,1-8.94-8,66,66,0,0,0,0-88,6,6,0,1,1,8.94-8A77.86,77.86,0,0,1,246,128Z"/></svg>}
