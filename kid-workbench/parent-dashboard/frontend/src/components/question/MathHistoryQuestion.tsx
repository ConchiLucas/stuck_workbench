import { MathPlayer, type KidAnswer, type MathExample } from '@kid-workbench/math-player'
import { appPath } from '../../appPath'
import '@kid-workbench/math-player/player.css'
import './mathHistory.css'

export interface MathReview {
  example?: MathExample
  selected?: string
  unavailable_reason?: string
  audio_mutable?: boolean
}

export function MathHistoryQuestion({ review, correct }: { review?: MathReview; correct?: boolean }) {
  if (!review?.example) {
    return <p className="math-review-unavailable">{review?.unavailable_reason || '这条早期记录未保存可还原的原题素材。'}</p>
  }
  const initial: KidAnswer | undefined = review.selected || correct !== undefined
    ? { selected: review.selected ?? '', placements: {}, correct: correct ?? null }
    : undefined
  return <details className="math-review-question">
    <summary>查看当时题目</summary>
    <p className="math-review-caption">按保存的题目和选项查看，不产生新的作答。{review.example.kind === 'audio-shape' ? (review.audio_mutable ? '音频来自当时记录的素材地址，可能已更新；缺失读音时不会改用浏览器朗读。' : '音频按当时保存的文件播放。') : '普通算式与情境题不播放读音。'}</p>
    <div className="math-history-player">
      <MathPlayer mode="kid" readOnly example={review.example} initialAnswer={initial} resolveAssetUrl={url => url ? appPath(url) : url} />
    </div>
    {!review.selected && <p className="math-review-caption">未保存当时的具体选项。</p>}
  </details>
}
