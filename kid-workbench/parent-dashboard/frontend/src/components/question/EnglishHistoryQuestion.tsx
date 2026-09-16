import { EnglishPlayer, type EnglishAnswer, type EnglishExample } from '@kid-workbench/english-player'
import { appPath } from '../../appPath'
import '@kid-workbench/english-player/player.css'
import './englishHistory.css'

export interface EnglishReview {
  example?: EnglishExample
  selected?: string
  response_kind?: string
  unavailable_reason?: string
}

export function EnglishHistoryQuestion({ review, correct }: { review?: EnglishReview; correct?: boolean }) {
  if (!review?.example) {
    return <p className="english-review-unavailable">{review?.unavailable_reason || '这条早期记录未保存可还原的原题素材。'}</p>
  }
  const initial: EnglishAnswer | undefined = review.selected || correct !== undefined
    ? {
        kind: review.example.kind === 'input-gap' ? 'input' : review.example.kind === 'card-builder' ? 'order' : 'choice',
        value: review.selected ?? '',
        correct: correct ?? null,
      }
    : undefined
  return <details className="english-review-question">
    <summary>查看当时题目</summary>
    <p className="english-review-caption">按保存的题目、选项和媒体查看，不产生新的作答。缺失读音时不会改用浏览器朗读。</p>
    {review.unavailable_reason ? <p className="english-review-unavailable">{review.unavailable_reason}</p> : <div className="english-history-player">
      <EnglishPlayer example={review.example} readOnly allowSyntheticSpeech={false} initialAnswer={initial} resolveAssetUrl={url => url ? appPath(url) : url} />
    </div>}
    {review.unavailable_reason && review.selected && <p>孩子当时的作答：{review.example.options?.find(o => o.id === review.selected)?.label || review.selected}</p>}
    {!review.selected && <p className="english-review-caption">未保存当时的具体作答。</p>}
  </details>
}
