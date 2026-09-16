import { LogicFacts, LogicPlayer, type LogicAnswer, type LogicExample } from '@kid-workbench/logic-player'
import type { Evidence } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'
import '@kid-workbench/logic-player/player.css'
import './scienceEvidence.css'

function initialAnswer(e: Evidence, example: LogicExample): LogicAnswer | undefined {
  const selected = e.response.value || e.response.selectedOptionId || ''
  if (!selected) return undefined
  if (example.kind === 'order') {
    try {
      const parsed = JSON.parse(selected) as LogicAnswer
      if (parsed && Array.isArray(parsed.sequence)) return { kind: 'order', sequence: parsed.sequence, rejected: parsed.rejected }
    } catch {
      const sequence = selected.split(',').map((id) => id.trim()).filter(Boolean)
      if (sequence.length) return { kind: 'order', sequence }
    }
    return undefined
  }
  try {
    const parsed = JSON.parse(selected) as LogicAnswer
    if (parsed?.selectedId) return { kind: example.kind, selectedId: parsed.selectedId }
  } catch {
    return { kind: example.kind, selectedId: selected }
  }
  return { kind: example.kind, selectedId: selected }
}

function playableExample(e: Evidence): LogicExample | undefined {
  const example = e.logicExample
  if (!example) return undefined
  const mediaUnavailable = ['mutable_reference', 'missing'].includes(e.mediaFidelity)
  const urls: Record<string, string> = {}
  for (const object of example.objects ?? []) {
    const raw = example.imageUrls?.[object.id]
    if (!raw) continue
    if (mediaUnavailable) continue
    urls[object.id] = mediaURL(e.attemptId, `glyph-${object.id}`)
  }
  return { ...example, imageUrls: urls }
}

export function LogicEvidence({ evidence: e, compact = false }: { evidence: Evidence; compact?: boolean }) {
  const unlinked = (e.evidenceReasonCodes || []).includes('unlinked_plan_item')
  const historic = ['instance_snapshot', 'frozen_version'].includes(e.questionFidelity)
  const example = historic && !unlinked ? playableExample(e) : undefined
  const mediaUnavailable = ['mutable_reference', 'missing'].includes(e.mediaFidelity)
  const initial = example ? initialAnswer(e, example) : undefined
  const missingNotice = unlinked
    ? '这条记录没有关联到当时那一题，无法还原作答画面。'
    : historic ? '未保存完整逻辑题面，无法还原当时画面。' : '未保存可信的逻辑题目快照，无法还原当时题面。'
  return <div className={`answer-evidence science-evidence${compact ? ' compact' : ''}`}>
    {example && mediaUnavailable ? <p className="evidence-notice">当时的图片未冻结、缺失或校验失败，无法可靠还原图片；作答记录仍然保留。</p> : example ? <div className="science-evidence-player"><LogicPlayer example={example} readOnly initialAnswer={initial} resolveAssetUrl={url => url} /></div> : <p className="evidence-notice">{missingNotice}对错记录仍然保留。</p>}
    {example ? <LogicFacts example={example} answer={initial} /> : null}
    <p className="science-evidence-note">作答按当时记录展示；只有已冻结且校验通过的媒体才能作为历史图片显示。浏览不会记入新的作答。</p>
  </div>
}
