import { LiteracyHistoryQuestion } from '../question/LiteracyHistoryQuestion'
import clsx from 'clsx'
import type { Bucket, PlanReviewItem, QuestionOption } from '../../api/plans'
import { QuestionVisual } from '../question/QuestionVisual'

const BUCKET_LABEL: Record<Bucket, string> = {
  review: '复习',
  shaky: '易错',
  learning: '在学',
  new: '新知',
}

const BUCKET_STYLE: Record<Bucket, string> = {
  review: 'bg-sky-400/15 text-sky-300',
  shaky: 'bg-rose-400/15 text-rose-300',
  learning: 'bg-amber-400/15 text-amber-200',
  new: 'bg-emerald-400/15 text-emerald-300',
}

function statusLabel(item: PlanReviewItem) {
  if (item.status === 'pending') return '未作答'
  if (item.status === 'correct' && item.tries === 1) return '✓ 一次答对'
  if (item.status === 'correct') return '✓ 第二次答对'
  if (item.status === 'wrong') return '✗ 没答对'
  return item.status
}

function optionMark(i: number, item: PlanReviewItem): 'correct' | 'picked-wrong' | 'neutral' {
  if (i === item.question.answer_index) return 'correct'
  if (item.picks.includes(i) && i !== item.question.answer_index) return 'picked-wrong'
  return 'neutral'
}

function optionText(opt: QuestionOption) {
  if (opt.emoji) return opt.emoji
  if (opt.shape) return opt.shape
  return opt.label ?? ''
}

export function ReviewItemCard({ item }: { item: PlanReviewItem }) {
  const hasPicks = item.picks.length > 0
  const costSec = item.cost_ms > 0 ? Math.round(item.cost_ms / 1000) : null

  return (
    <article className="dash-card p-5">
      <header className="flex flex-wrap items-center gap-2 text-sm">
        <span className="font-semibold text-white">第 {item.seq} 题</span>
        <span className="text-night-mute">
          {item.subject_name} · {item.kp_title}
        </span>
        <span className={clsx('rounded-md px-1.5 py-0.5 text-xs', BUCKET_STYLE[item.bucket])}>
          {BUCKET_LABEL[item.bucket]}
        </span>
        <span
          className={clsx(
            'rounded-md px-1.5 py-0.5 text-xs',
            item.status === 'correct' && 'bg-emerald-400/15 text-emerald-300',
            item.status === 'wrong' && 'bg-rose-400/15 text-rose-300',
            item.status === 'pending' && 'bg-white/10 text-night-mute',
          )}
        >
          {statusLabel(item)}
        </span>
        {costSec !== null && (
          <span className="ml-auto text-xs text-night-mute">{costSec} 秒</span>
        )}
      </header>

      {item.literacy_review ? <LiteracyHistoryQuestion review={item.literacy_review} correct={item.status === 'correct' ? true : item.status === 'wrong' ? false : undefined} /> : <>
      <div className="mt-4 flex flex-col items-center gap-3">
        <QuestionVisual visual={item.question.visual} size="compact" />
        <p className="text-center text-base text-white">{item.question.stem}</p>
      </div>

      <ul className="mt-4 grid grid-cols-2 gap-2">
        {item.question.options.map((opt, i) => {
          const mark = optionMark(i, item)
          return (
            <li
              key={i}
              className={clsx(
                'flex items-center gap-2 rounded-xl2 border px-3 py-2.5 text-sm',
                mark === 'correct' && 'border-emerald-400/50 bg-emerald-400/10 text-emerald-200',
                mark === 'picked-wrong' && 'border-rose-400/50 bg-rose-400/10 text-rose-200',
                mark === 'neutral' && 'border-white/10 bg-white/5 text-night-mute',
              )}
            >
              <span className="flex-1">{optionText(opt)}</span>
              {mark === 'correct' && <span aria-label="正确答案">✓</span>}
              {mark === 'picked-wrong' && <span aria-label="选错了">✗</span>}
            </li>
          )
        })}
      </ul>
      </>}

      {!hasPicks && item.status !== 'pending' && (
        <p className="mt-3 text-xs text-night-mute">早期记录未保存选项</p>
      )}
    </article>
  )
}
