import { MathPlayer, type KidAnswer, type MathExample } from '@kid-workbench/math-player'
import type { Evidence } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'
import '@kid-workbench/math-player/player.css'
import './mathEvidence.css'

export function MathEvidence({evidence:e,compact=false}:{evidence:Evidence;compact?:boolean}) {
 const historic=['instance_snapshot','frozen_version'].includes(e.questionFidelity)
 const example=historic?e.mathExample:undefined
 const selected=historic && e.selectionFidelity==='stable_option'?e.question?.options.find(o=>o.id===e.response.selectedOptionId):undefined
 const answer=historic?e.question?.options.find(o=>o.id===e.question?.answerOptionId):undefined
 const playable:MathExample|undefined=example?{...example,audioUrl:example.kind==='audio-shape'&&e.question?.stem.audioMediaId?mediaURL(e.attemptId,e.question.stem.audioMediaId):undefined}:undefined
 const initial:KidAnswer|undefined=playable?{selected:selected?.label??'',placements:{},correct:e.isCorrect}:undefined
 return <div className={`answer-evidence math-evidence${compact?' compact':''}`}>
  {playable?<div className="math-evidence-player"><MathPlayer mode="kid" readOnly example={playable} initialAnswer={initial} resolveAssetUrl={url=>url}/></div>:<p className="evidence-notice">{historic?'未保存完整算术题面，无法还原当时画面。':'未保存可信的算术题目快照，无法还原当时题面。'}对错记录仍然保留。</p>}
  <div className="math-evidence-facts">
   <p>{selected?`孩子选了：${selected.label}（当时第 ${(e.question?.options.indexOf(selected)??0)+1} 项）`:'未保存可核对的具体选项。'}</p>
   {answer&&<p>正确答案：{answer.label}（当时第 {(e.question?.options.indexOf(answer)??0)+1} 项）</p>}
   {e.mediaFidelity==='mutable_reference'&&playable?.kind==='audio-shape'&&<p className="evidence-notice">题目和选项按当时记录展示；音频来自记录中的素材地址，未标注版本的音频可能已更新。</p>}
  </div>
 </div>
}
