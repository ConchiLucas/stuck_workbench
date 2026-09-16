import { GlyphSenseQuestion, type PlayerQuestion } from '@kid-workbench/literacy-player'
import { appPath } from '../../appPath'
import './literacyHistory.css'

export interface LiteracyReview {
  question?: PlayerQuestion
  selected_option_id?: string
  answer_option_id?: string
  unavailable_reason?: string
}

export function LiteracyHistoryQuestion({ review, correct }: { review?: LiteracyReview; correct?: boolean }) {
  if (!review?.question || review.question.questionType !== 'glyph_sense') {
    return <p className="literacy-history-unavailable">{review?.unavailable_reason || '这条早期记录未保存可还原的原题素材。'}</p>
  }
  return <details className="literacy-history-question">
    <summary>查看当时题目</summary>
    <p className="literacy-history-caption">按保存的题目和素材版本查看，不产生新的作答。</p>
    <GlyphSenseQuestion question={review.question} selectedOptionId={review.selected_option_id}
      correct={correct} readOnly mediaResolver={ref => typeof ref === 'string' ? appPath(ref) : undefined} />
    {!review.selected_option_id && <p className="literacy-history-caption">未保存当时的具体选项。</p>}
  </details>
}
