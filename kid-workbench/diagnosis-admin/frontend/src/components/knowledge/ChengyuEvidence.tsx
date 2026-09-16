import { ChengyuPlayer, type ChengyuAnswer, type ChengyuExample } from '@kid-workbench/chengyu-player'
import type { Evidence } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'
import '@kid-workbench/chengyu-player/player.css'
import './chengyuEvidence.css'

function playableExample(e: Evidence): ChengyuExample | undefined {
  const example = e.chengyuExample
  if (!example) return undefined
  const q = e.question
  return {
    ...example,
    speechUrl: ['mutable_reference', 'missing'].includes(e.mediaFidelity)
      ? ''
      : q?.stem.audioMediaId ? mediaURL(e.attemptId, q.stem.audioMediaId) : example.speechUrl,
  }
}

export function ChengyuEvidence({ evidence: e, compact = false }: { evidence: Evidence; compact?: boolean }) {
  const unlinked = (e.evidenceReasonCodes || []).includes('unlinked_plan_item')
  const historic = ['instance_snapshot', 'frozen_version'].includes(e.questionFidelity)
  const example = historic && !unlinked ? playableExample(e) : undefined
  const mediaUnavailable = ['mutable_reference', 'missing'].includes(e.mediaFidelity)
  const selected = e.response.value || e.response.selectedOptionId || ''
  const initial: ChengyuAnswer | undefined = example && selected
    ? { kind: 'choice', value: selected, correct: e.isCorrect }
    : undefined
  const answerLabel = example?.options?.find(o => o.id === example.answerId)?.label
  const selectedLabel = example?.options?.find(o => o.id === selected)?.label || selected
  const missingNotice = unlinked
    ? '这条记录没有关联到当时那一题，无法还原作答画面。'
    : historic ? '未保存完整成语题面，无法还原当时画面。' : '未保存可信的成语题目快照，无法还原当时题面。'
  return <div className={`answer-evidence chengyu-evidence${compact ? ' compact' : ''}`}>
    {example ? <>
      {mediaUnavailable ? <p className="evidence-notice">当时的读音未冻结、缺失或校验失败，无法可靠还原音频；作答记录仍然保留。</p> : null}
      <div className="chengyu-evidence-player"><ChengyuPlayer example={example} readOnly initialAnswer={initial} resolveAssetUrl={url => url}/></div>
    </> : <p className="evidence-notice">{missingNotice}对错记录仍然保留。</p>}
    <div className="chengyu-evidence-facts">
      <p>{selected ? `孩子当时的作答：${selectedLabel}` : '未保存可核对的具体作答。'}</p>
      {answerLabel && <p>正确答案：{answerLabel}</p>}
      <p>作答按当时记录展示；只有已冻结且校验通过的媒体才能作为历史读音播放。</p>
    </div>
  </div>
}
