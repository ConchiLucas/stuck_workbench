import { PoemFacts, PoemPlayer, type PoemAnswer, type PoemExample } from '@kid-workbench/poem-player'
import type { Evidence } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'
import '@kid-workbench/poem-player/player.css'
import './poemEvidence.css'

function initialAnswer(e: Evidence, example: PoemExample): PoemAnswer | undefined {
  const selected = e.response.value || e.response.selectedOptionId || ''
  if (!selected) return undefined
  if (example.kind === 'recite') {
    try {
      const parsed = JSON.parse(selected) as PoemAnswer
      if (parsed && Array.isArray(parsed.sequence)) return { kind: 'recite', sequence: parsed.sequence }
    } catch {
      const sequence = selected.split(',').map((id) => id.trim()).filter(Boolean)
      if (sequence.length) return { kind: 'recite', sequence }
    }
    return undefined
  }
  return { kind: example.kind, selectedId: selected }
}

function playableExample(e: Evidence): PoemExample | undefined {
  const example = e.poemExample
  if (!example) return undefined
  const mediaUnavailable = ['mutable_reference', 'missing'].includes(e.mediaFidelity)
  if (!example.speechUrl) return example
  if (mediaUnavailable) return { ...example, speechUrl: undefined, audioMissingReason: example.audioMissingReason || '当时的读音未冻结、缺失或校验失败。' }
  return { ...example, speechUrl: mediaURL(e.attemptId, 'stem-audio') }
}

export function PoemEvidence({ evidence: e, compact = false }: { evidence: Evidence; compact?: boolean }) {
  const unlinked = (e.evidenceReasonCodes || []).includes('unlinked_plan_item')
  const historic = ['instance_snapshot', 'frozen_version'].includes(e.questionFidelity)
  const example = historic && !unlinked ? playableExample(e) : undefined
  const mediaUnavailable = ['mutable_reference', 'missing'].includes(e.mediaFidelity)
  const initial = example ? initialAnswer(e, example) : undefined
  const missingNotice = unlinked
    ? '这条记录没有关联到当时那一题，无法还原作答画面。'
    : historic ? '未保存完整古诗题面，无法还原当时画面。' : '未保存可信的古诗题目快照，无法还原当时题面。'
  return <div className={`answer-evidence poem-evidence${compact ? ' compact' : ''}`}>
    {example && mediaUnavailable && example.kind !== 'fill' && example.kind !== 'recite' ? <p className="evidence-notice">当时的读音未冻结、缺失或校验失败，无法可靠还原读音；作答记录仍然保留。</p> : example ? <div className="poem-evidence-player"><PoemPlayer example={example} readOnly initialAnswer={initial} resolveAssetUrl={url => url} /></div> : <p className="evidence-notice">{missingNotice}对错记录仍然保留。</p>}
    {example ? <PoemFacts example={example} answer={initial} /> : null}
    <p className="poem-evidence-note">作答按当时记录展示；只有已冻结且校验通过的媒体才能作为历史读音播放。浏览不会记入新的作答。</p>
  </div>
}
