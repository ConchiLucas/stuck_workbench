import { PoemFacts, PoemPlayer, type PoemAnswer, type PoemExample } from '@kid-workbench/poem-player'
import { appPath } from '../../appPath'
import '@kid-workbench/poem-player/player.css'
import './poemHistory.css'

export interface PoemReview {
  example?: PoemExample
  selected?: string
  response_kind?: string
  facts?: string[]
  unavailable_reason?: string
}

function initialAnswer(review: PoemReview): PoemAnswer | undefined {
  if (!review.example || !review.selected) return undefined
  const kind = review.example.kind
  if (kind === 'recite') {
    try {
      const parsed = JSON.parse(review.selected) as PoemAnswer
      if (parsed && Array.isArray(parsed.sequence)) return { kind: 'recite', sequence: parsed.sequence }
    } catch {
      const sequence = review.selected.split(',').map((id) => id.trim()).filter(Boolean)
      if (sequence.length) return { kind: 'recite', sequence }
    }
    return undefined
  }
  return { kind, selectedId: review.selected }
}

export function PoemHistoryQuestion({ review, correct }: { review?: PoemReview; correct?: boolean }) {
  if (!review?.example) {
    return <p className="poem-review-unavailable">{review?.unavailable_reason || '这条早期记录未保存可还原的原题素材。'}</p>
  }
  const initial = initialAnswer(review)
  return <details className="poem-review-question">
    <summary>查看当时题目</summary>
    <p className="poem-review-caption">按保存的诗文、选项、诗行顺序和媒体查看，不产生新的作答。</p>
    {review.unavailable_reason ? <p className="poem-review-unavailable">{review.unavailable_reason}</p> : <div className="poem-history-player">
      <PoemPlayer example={review.example} readOnly initialAnswer={initial} resolveAssetUrl={url => url ? appPath(url) : url} />
    </div>}
    {review.facts?.length ? <PoemFacts example={review.example} answer={initial} /> : review.selected ? <p>孩子当时的作答已按当时记录展示。</p> : <p className="poem-review-caption">未保存当时的具体作答。</p>}
    {correct === false && !review.facts?.length ? <p>当时答错了，但没有保存可核对的结构化输入。</p> : null}
  </details>
}
