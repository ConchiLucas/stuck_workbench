import { useState } from 'react'
import { PinyinEvidence } from './PinyinEvidence'
import { EnglishEvidence } from './EnglishEvidence'
import { PhraseEvidence } from './PhraseEvidence'
import { ChengyuEvidence } from './ChengyuEvidence'
import { ScienceEvidence } from './ScienceEvidence'
import { PoemEvidence } from './PoemEvidence'
import { LogicEvidence } from './LogicEvidence'
import { MathEvidence } from './MathEvidence'
import type { Evidence, Option } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'

function EvidenceImage({attempt,id,alt}:{attempt:number;id:string;alt:string}){const[failed,setFailed]=useState(false);return failed?<span className="media-missing">图片暂不可用</span>:<img src={mediaURL(attempt,id)} alt={alt} loading="lazy" onError={()=>setFailed(true)}/>}
function EvidenceAudio({attempt,id}:{attempt:number;id:string}){const[failed,setFailed]=useState(false);return failed?<span className="media-missing">音频暂不可用</span>:<audio controls preload="none" src={mediaURL(attempt,id)} aria-label="播放题目读音" onError={()=>setFailed(true)}/>}
function Handwriting({strokes}:{strokes:unknown}){const paths=Array.isArray(strokes)?strokes.map(s=>Array.isArray(s)?s:(s&&typeof s==='object'&&'points' in s&&Array.isArray(s.points)?s.points:[])):[];return <svg className="handwriting-record" viewBox="0 0 300 300" role="img" aria-label="孩子当时的书写笔迹"><path d="M150 0V300M0 150H300M0 0L300 300M300 0L0 300" className="writing-grid"/>{paths.map((points,i)=><polyline key={i} points={points.filter((p:{x:number;y:number})=>p&&Number.isFinite(p.x)&&Number.isFinite(p.y)).map((p:{x:number;y:number})=>`${Math.max(0,Math.min(1,p.x))*300},${Math.max(0,Math.min(1,p.y))*300}`).join(' ')} fill="none" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round"/>)}</svg>}
export function AnswerEvidence({evidence:e,compact=false}:{evidence:Evidence;compact?:boolean}){
 if(e.subjectCode==='pinyin') return <PinyinEvidence evidence={e} compact={compact}/>
 if(e.subjectCode==='math') return <MathEvidence evidence={e} compact={compact}/>
 if(e.subjectCode==='english') return <EnglishEvidence evidence={e} compact={compact}/>
 if(e.subjectCode==='phrase') return <PhraseEvidence evidence={e} compact={compact}/>
 if(e.subjectCode==='chengyu') return <ChengyuEvidence evidence={e} compact={compact}/>
 if(e.subjectCode==='science') return <ScienceEvidence evidence={e} compact={compact}/>
 if(e.subjectCode==='poem') return <PoemEvidence evidence={e} compact={compact}/>
 if(e.subjectCode==='logic') return <LogicEvidence evidence={e} compact={compact}/>
 const q=e.question;const historic=['frozen_version','instance_snapshot'].includes(e.questionFidelity)
 return <div className={`answer-evidence${compact?' compact':''}`}>
 {!historic&&<p className="evidence-notice">{e.questionFidelity==='current_reference'?'当前内容参考；未保存当时的选项，无法还原孩子的具体错选。':'旧记录未保存完整现场；下面仅显示可以核对的内容。'}</p>}
 {q?<><div className="question-stem"><strong>{q.stem.text||q.targetText||e.title}</strong>{q.visual?.initial&&q.visual?.final&&<div className="pinyin-blend" aria-label="拼读现场">{q.visual.initial} + {q.visual.final} = ?</div>}{q.visual?.letter&&<div className="pinyin-letter">{q.visual.letter}</div>}{q.stem.imageMediaId&&<EvidenceImage attempt={e.attemptId} id={q.stem.imageMediaId} alt="题目图片"/>}{q.stem.audioMediaId&&<EvidenceAudio attempt={e.attemptId} id={q.stem.audioMediaId}/>}</div>
 {e.response.kind==='handwriting'?<div className="writing-evidence"><Handwriting strokes={e.response.strokes}/><div><strong>{e.isCorrect?'本次通过':'本次尚未通过'}</strong><p>{e.assistance==='hinted'?`使用提示 ${e.response.hintsUsed??1} 次`:e.assistance==='none'?'未使用提示':'提示使用情况未记录'}</p><small>保留当时的笔迹与结果</small></div></div>:<div className="evidence-options">{q.options.map((o:Option)=>{const selected=e.response.selectedOptionId===o.id;const correct=q.answerOptionId===o.id;return <div key={o.id} className={`evidence-option${selected?' selected':''}${correct&&historic?' correct':''}`}><span className="option-caption">{selected?'孩子选了':correct&&historic?'正确答案':'选项'}</span>{o.imageMediaId&&<EvidenceImage attempt={e.attemptId} id={o.imageMediaId} alt={o.label||'选项图片'}/>}<span className="option-label">{o.label}</span>{o.audioMediaId&&<EvidenceAudio attempt={e.attemptId} id={o.audioMediaId}/>}</div>})}</div>}</>:<p className="empty">这次作答没有保存题目画面，对错记录仍然保留。</p>}
 </div>
}
