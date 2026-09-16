import { LogicFacts, LogicPlayer, type LogicAnswer, type LogicExample } from '@kid-workbench/logic-player'
import { appPath } from '../../appPath'
import '@kid-workbench/logic-player/player.css'
import './scienceHistory.css'

export interface LogicReview {
  example?: LogicExample
  selected?: string
  response_kind?: string
  facts?: string[]
  unavailable_reason?: string
}

function initialAnswer(review: LogicReview): LogicAnswer | undefined {
  if (!review.example || !review.selected) return undefined
  const kind = review.example.kind
  if (kind === 'order') {
    try {
      const parsed = JSON.parse(review.selected) as LogicAnswer
      if (parsed && Array.isArray(parsed.sequence)) return { kind: 'order', sequence: parsed.sequence, rejected: parsed.rejected }
    } catch {
      const sequence = review.selected.split(',').map((id) => id.trim()).filter(Boolean)
      if (sequence.length) return { kind: 'order', sequence }
    }
    return undefined
  }
  try {
    const parsed = JSON.parse(review.selected) as LogicAnswer
    if (parsed?.selectedId) return { kind, selectedId: parsed.selectedId }
  } catch {
    return { kind, selectedId: review.selected }
  }
  return { kind, selectedId: review.selected }
}

export function LogicHistoryQuestion({ review, correct }: { review?: LogicReview; correct?: boolean }) {
  if (!review?.example) {
    return <p className="science-review-unavailable">{review?.unavailable_reason || '这条早期记录未保存可还原的原题素材。'}</p>
  }
  const initial = initialAnswer(review)
  return <details className="science-review-question">
    <summary>查看当时题目</summary>
    <p className="science-review-caption">按保存的题目、规则、选项顺序和图形查看，不产生新的作答。</p>
    {review.unavailable_reason ? <p className="science-review-unavailable">{review.unavailable_reason}</p> : <div className="science-history-player">
      <LogicPlayer example={review.example} readOnly initialAnswer={initial} resolveAssetUrl={url => url ? appPath(url) : url} />
    </div>}
    {review.facts?.length ? <LogicFacts example={review.example} answer={initial} /> : review.selected ? <p>孩子当时的作答已按当时记录展示。</p> : <p className="science-review-caption">未保存当时的具体作答。</p>}
    {correct === false && !review.facts?.length ? <p>当时答错了，但没有保存可核对的结构化输入。</p> : null}
  </details>
}
