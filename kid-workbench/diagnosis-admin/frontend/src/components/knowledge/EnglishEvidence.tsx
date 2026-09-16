import { EnglishPlayer, type EnglishAnswer, type EnglishExample } from '@kid-workbench/english-player'
import type { Evidence } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'
import '@kid-workbench/english-player/player.css'
import './englishEvidence.css'

function playableExample(e: Evidence): EnglishExample | undefined {
  const example = e.englishExample
  if (!example) return undefined
  const q = e.question
  return {
    ...example,
    speechUrl: q?.stem.audioMediaId ? mediaURL(e.attemptId, q.stem.audioMediaId) : example.speechUrl,
    cue: q?.stem.imageMediaId ? mediaURL(e.attemptId, q.stem.imageMediaId) : example.cue,
    options: example.options?.map((option, index) => {
      const imageId = q?.options[index]?.imageMediaId
      return { ...option, picture: imageId ? mediaURL(e.attemptId, imageId) : option.picture }
    }),
  }
}

export function EnglishEvidence({ evidence: e, compact = false }: { evidence: Evidence; compact?: boolean }) {
  const unlinked = (e.evidenceReasonCodes || []).includes('unlinked_plan_item')
  const historic = ['instance_snapshot', 'frozen_version'].includes(e.questionFidelity)
  const example = historic && !unlinked ? playableExample(e) : undefined
  const mediaUnavailable = ["mutable_reference", "missing"].includes(e.mediaFidelity)
  const selected = e.response.value || e.response.selectedOptionId || ''
  const initial: EnglishAnswer | undefined = example && selected
    ? { kind: example.kind === 'input-gap' ? 'input' : example.kind === 'card-builder' ? 'order' : 'choice', value: selected, correct: e.isCorrect }
    : undefined
  const answerLabel = example?.kind === 'input-gap' || example?.kind === 'card-builder'
    ? example.answer
    : example?.options?.find(o => o.id === example.answerId)?.label
  const selectedLabel = example && ['audio-choice', 'image-text', 'reading-qa'].includes(example.kind)
    ? example.options?.find(o => o.id === selected)?.label || selected
    : selected
  const missingNotice = unlinked
    ? '这条记录没有关联到当时那一题，无法还原作答画面。'
    : historic ? '未保存完整英语题面，无法还原当时画面。' : '未保存可信的英语题目快照，无法还原当时题面。'
  return <div className={`answer-evidence english-evidence${compact ? ' compact' : ''}`}>
    {example && mediaUnavailable ? <p className="evidence-notice">当时的图片或读音未冻结、缺失或校验失败，无法可靠还原图音；作答记录仍然保留。</p> : example ? <div className="english-evidence-player"><EnglishPlayer example={example} readOnly allowSyntheticSpeech={false} initialAnswer={initial} resolveAssetUrl={url => url}/></div> : <p className="evidence-notice">{missingNotice}对错记录仍然保留。</p>}
    <div className="english-evidence-facts">
      <p>{selected ? `孩子当时的作答：${selectedLabel}` : '未保存可核对的具体作答。'}</p>
      {answerLabel && <p>正确答案：{answerLabel}</p>}
      <p>作答按当时记录展示；只有已冻结且校验通过的媒体才能作为历史图音播放。</p>
    </div>
  </div>
}
