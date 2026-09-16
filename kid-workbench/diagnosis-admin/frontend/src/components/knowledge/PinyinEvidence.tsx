import { PinyinQuestion, type PinyinQuestionType, type PinyinQuestionView } from '@kid-workbench/pinyin-player'
import type { Evidence } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'
import './pinyinEvidence.css'

function frozenQuestion(e: Evidence): PinyinQuestionView | undefined {
 const q=e.question
 if(!q || !['instance_snapshot','frozen_version'].includes(e.questionFidelity) || !['listen','inword','shape','blend'].includes(e.questionType)) return
 const v=q.visual
 const type=e.questionType as PinyinQuestionType
 if(q.options.length!==4 || q.options.some(o=>!o.id) || new Set(q.options.map(o=>o.id)).size!==q.options.length) return
 if(!q.options.some(o=>o.id===q.answerOptionId) || (e.response.selectedOptionId && !q.options.some(o=>o.id===e.response.selectedOptionId))) return
 if((type==='listen'||type==='inword') && q.options.some(o=>!o.label?.trim())) return
 if((type==='shape'||type==='inword') && !v?.text) return
 if(type==='blend' && (!v?.initial||!v?.final)) return
 return {id:`attempt-${e.attemptId}`,type,stem:q.stem.text,visual:{kind:v?.kind||type,text:v?.text,initial:v?.initial,final:v?.final},speechUrl:q.stem.audioMediaId?mediaURL(e.attemptId,q.stem.audioMediaId):undefined,options:q.options.map(o=>({id:o.id,label:o.label,speechUrl:o.audioMediaId?mediaURL(e.attemptId,o.audioMediaId):undefined}))}
}
export function PinyinEvidence({evidence:e,compact=false}:{evidence:Evidence;compact?:boolean}) {
 const question=frozenQuestion(e)
 const historic=['instance_snapshot','frozen_version'].includes(e.questionFidelity)
 const selected=historic && e.selectionFidelity==='stable_option'?e.question?.options.find(o=>o.id===e.response.selectedOptionId):undefined
 const answer=historic?e.question?.options.find(o=>o.id===e.question?.answerOptionId):undefined
 return <div className={`answer-evidence pinyin-evidence${compact?' compact':''}`}>
  {question?<PinyinQuestion question={question} selectedOptionId={selected?.id} correct={e.isCorrect} readOnly allowSyntheticSpeech={false}/>:<p className="evidence-notice">{historic?'未保存完整拼音题面，无法还原当时画面。':'未保存可信的拼音题目快照，无法还原当时题面。'}对错记录仍然保留。</p>}
  <div className="pinyin-evidence-facts">
   <p>{selected?`孩子选了：${selected.label||'所选读音'}（当时第 ${(e.question?.options.indexOf(selected)??0)+1} 项）`:'未保存可核对的具体选项。'}</p>
   {answer&&<p>正确答案：{answer.label||'正确读音'}（当时第 {(e.question?.options.indexOf(answer)??0)+1} 项）</p>}
   {e.mediaFidelity==='mutable_reference'&&<p className="evidence-notice">题目和选项按当时记录展示；音频来自记录中的素材地址，未标注版本的音频可能已更新。</p>}
  </div>
 </div>
}
