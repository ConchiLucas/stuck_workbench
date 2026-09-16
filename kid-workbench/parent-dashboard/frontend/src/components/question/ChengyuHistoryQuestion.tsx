import { ChengyuPlayer, type ChengyuAnswer, type ChengyuExample } from '@kid-workbench/chengyu-player'
import { appPath } from '../../appPath'
import '@kid-workbench/chengyu-player/player.css'
import './chengyuHistory.css'

export interface ChengyuReview {
  example?: ChengyuExample
  selected?: string
  response_kind?: string
  unavailable_reason?: string
}

export function ChengyuHistoryQuestion({ review, correct }: { review?: ChengyuReview; correct?: boolean }) {
  if (!review?.example) {
    return <p className="chengyu-review-unavailable">{review?.unavailable_reason || '这条早期记录未保存可还原的原题素材。'}</p>
  }
  const initial: ChengyuAnswer | undefined = review.selected || correct !== undefined
    ? { kind: 'choice', value: review.selected ?? '', correct: correct ?? null }
    : undefined
  const selectedLabel = review.example.options?.find(o => o.id === review.selected)?.label || review.selected
  const answerLabel = review.example.options?.find(o => o.id === review.example?.answerId)?.label
  return <details className="chengyu-review-question">
    <summary>查看当时题目</summary>
    <p className="chengyu-review-caption">按保存的题目、选项和媒体查看，不产生新的作答。缺失读音时不会改用浏览器朗读。</p>
    {review.unavailable_reason ? <p className="chengyu-review-unavailable">{review.unavailable_reason}</p> : null}
    <div className="chengyu-history-player">
      <ChengyuPlayer example={review.example} readOnly initialAnswer={initial} resolveAssetUrl={url => url ? appPath(url) : url} />
    </div>
    {review.selected ? <p>孩子当时的作答：{selectedLabel}</p> : <p className="chengyu-review-caption">未保存当时的具体作答。</p>}
    {answerLabel ? <p>正确答案：{answerLabel}</p> : null}
  </details>
}
