import { ScienceFacts, SciencePlayer, type ScienceAnswer, type ScienceExample } from '@kid-workbench/science-player'
import type { Evidence } from '../../api/knowledgeTypes'
import { mediaURL } from '../../api/knowledge'
import '@kid-workbench/science-player/player.css'
import './scienceEvidence.css'

function initialAnswer(e: Evidence, example: ScienceExample): ScienceAnswer | undefined {
  const selected = e.response.value || e.response.selectedOptionId || ''
  if (!selected) return undefined
  if (example.kind === 'choice') return { kind: 'choice', selectedId: selected }
  try {
    const parsed = JSON.parse(selected) as ScienceAnswer
    if (parsed && parsed.kind) return parsed
  } catch {
    return { kind: example.kind, selectedId: selected }
  }
  return { kind: example.kind, selectedId: selected }
}

function playableExample(e: Evidence): ScienceExample | undefined {
  const example = e.scienceExample
  if (!example) return undefined
  const mediaUnavailable = ['mutable_reference', 'missing'].includes(e.mediaFidelity)
  const proxy = (url: string | undefined, id: string) => {
    if (!url) return url
    if (mediaUnavailable) return undefined
    return mediaURL(e.attemptId, id)
  }
  return {
    ...example,
    imageUrl: proxy(example.imageUrl, 'stem-image'),
    options: example.options?.map((option, index) => ({ ...option, imageUrl: proxy(option.imageUrl, `option-${index}-image`) })),
    matchSources: example.matchSources?.map((node, index) => ({ ...node, imageUrl: proxy(node.imageUrl, `match-source-${index}`) })),
    matchTargets: example.matchTargets?.map((node, index) => ({ ...node, imageUrl: proxy(node.imageUrl, `match-target-${index}`) })),
    sequenceItems: example.sequenceItems?.map((node, index) => ({ ...node, imageUrl: proxy(node.imageUrl, `sequence-${index}`) })),
  }
}

export function ScienceEvidence({ evidence: e, compact = false }: { evidence: Evidence; compact?: boolean }) {
  const unlinked = (e.evidenceReasonCodes || []).includes('unlinked_plan_item')
  const historic = ['instance_snapshot', 'frozen_version'].includes(e.questionFidelity)
  const example = historic && !unlinked ? playableExample(e) : undefined
  const mediaUnavailable = ['mutable_reference', 'missing'].includes(e.mediaFidelity)
  const initial = example ? initialAnswer(e, example) : undefined
  const missingNotice = unlinked
    ? '这条记录没有关联到当时那一题，无法还原作答画面。'
    : historic ? '未保存完整科普题面，无法还原当时画面。' : '未保存可信的科普题目快照，无法还原当时题面。'
  return <div className={`answer-evidence science-evidence${compact ? ' compact' : ''}`}>
    {example && mediaUnavailable ? <p className="evidence-notice">当时的图片未冻结、缺失或校验失败，无法可靠还原图片；作答记录仍然保留。</p> : example ? <div className="science-evidence-player"><SciencePlayer example={example} readOnly initialAnswer={initial} resolveAssetUrl={url => url} /></div> : <p className="evidence-notice">{missingNotice}对错记录仍然保留。</p>}
    {example ? <ScienceFacts example={example} answer={initial} /> : null}
    <p className="science-evidence-note">作答按当时记录展示；只有已冻结且校验通过的媒体才能作为历史图片显示。浏览不会记入新的作答。</p>
  </div>
}
