import { PinyinQuestion, type PinyinQuestionView } from '@kid-workbench/pinyin-player'
import { appPath } from '../../appPath'
import './pinyinHistory.css'

export interface PinyinReview {
  question?: PinyinQuestionView
  selected_option_id?: string
  unavailable_reason?: string
}

const pinyinTypes = new Set(['listen', 'inword', 'shape', 'blend'])

export function PinyinHistoryQuestion({ review, correct }: { review?: PinyinReview; correct?: boolean }) {
  if (!review?.question || !pinyinTypes.has(review.question.type)) {
    return <p className="pinyin-review-unavailable">{review?.unavailable_reason || '这条早期记录未保存可还原的原题素材。'}</p>
  }
  return <details className="pinyin-review-question">
    <summary>查看当时题目</summary>
    <p className="pinyin-review-caption">按保存的题目和选项查看，不产生新的作答。音频来自当时记录的素材地址；未标注版本的音频可能已更新；缺失读音时不会改用浏览器朗读。</p>
    <PinyinQuestion
      question={review.question}
      selectedOptionId={review.selected_option_id}
      correct={correct}
      readOnly
      resolveUrl={url => url ? appPath(url) : undefined}
    />
    {!review.selected_option_id && <p className="pinyin-review-caption">未保存当时的具体选项。</p>}
  </details>
}
