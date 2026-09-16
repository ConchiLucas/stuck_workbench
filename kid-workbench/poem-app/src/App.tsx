import { appPath } from './appPath'
import { Link, Navigate, Route, Routes, useParams } from 'react-router-dom'
import { PoemPlayer, type PoemAnswer } from '@kid-workbench/poem-player'
import '@kid-workbench/poem-player/player.css'
import {
  clueExample,
  clueHref,
  isPracticeType,
  nextHref,
  pavilionByCode,
  pickKey,
  prevHref,
  taskLabel,
  typeBanks,
  typeHref,
  typeNextHref,
  typePrevHref,
  typeTitles,
  practiceTypes,
  type Clue,
  type PracticeType,
} from './garden'
import { IconClose, IconNext, IconPrev } from './icons'
import { InkMark } from './InkMark'
import { usePoemStore } from './store'

const EMPTY_SEQUENCE: string[] = []

function TypeBoard({ type, n, clue }: { type: PracticeType; n: number; clue: Clue }) {
  const setPick = usePoemStore((s) => s.setPick)
  const recite = usePoemStore((s) => s.recite)
  const example = clueExample(clue)
  function onAnswer(answer: PoemAnswer) {
    if (clue.kind === 'recite' && clue.sequence) {
      const seq = answer.sequence ?? []
      const last = seq[seq.length - 1]
      if (last) recite(type, n, last, clue.sequence, clue.answerId)
      return
    }
    if (answer.selectedId) setPick(type, n, answer.selectedId)
  }
  return <PoemPlayer example={example} onAnswer={onAnswer} />
}

function HomePage() {
  return (
    <section className="type-gallery-page" aria-label="题型">
      <div className="type-gallery">
        {practiceTypes.map((type) => (
          <Link
            key={type}
            className={`type-card type-card-${type}`}
            to={typeHref(type)}
            aria-label={typeTitles[type]}
            onClick={() => usePoemStore.getState().clearType(type)}
          >
            <img src={appPath(`/cards/${type}.png`)} alt="" />
            <strong>{typeTitles[type]}</strong>
          </Link>
        ))}
      </div>
    </section>
  )
}

function KidBar({
  label,
  n,
  total,
  prev,
  next,
  nextLabel,
}: {
  label: string
  n: number
  total: number
  prev?: string
  next: string
  nextLabel: string
}) {
  return (
    <nav className="kid-top" aria-label="练习导航">
      <Link className="kid-circle" to="/" aria-label="返回"><IconClose /></Link>
      <div className="kid-progress" role="progressbar" aria-label={`${label}，第 ${n} / ${total} 题`} aria-valuemin={1} aria-valuemax={total} aria-valuenow={n}>
        <i style={{ width: `${(n / total) * 100}%` }} />
        <span className="kid-progress-label" aria-hidden="true">{n} / {total}</span>
      </div>
      <div className="kid-nav">
        {prev ? <Link className="kid-circle" to={prev} aria-label="上一题"><IconPrev /></Link> : <span className="kid-circle is-muted" aria-hidden="true"><IconPrev /></span>}
        <Link className="kid-circle is-next" to={next} aria-label={nextLabel}><IconNext /></Link>
      </div>
    </nav>
  )
}

function PlayPage() {
  const { code, n } = useParams()
  const pavilion = pavilionByCode(code)
  const current = n ? Number(n) : 1
  if (!pavilion || !Number.isInteger(current) || current < 1 || current > pavilion.clues.length) return <Navigate to="/" replace />
  const clue = pavilion.clues[current - 1]
  const last = current >= pavilion.clues.length
  return (
    <main className="play-shell">
      <KidBar
        label={pavilion.kidTitle}
        n={current}
        total={pavilion.clues.length}
        prev={prevHref(pavilion, current)}
        next={nextHref(pavilion, current)}
        nextLabel={last ? '查看结果' : '下一题'}
      />
      <ClueBoard code={pavilion.code} n={current} clue={clue} />
    </main>
  )
}

function TypePlayPage() {
  const { type, n } = useParams()
  if (!isPracticeType(type)) return <Navigate to="/" replace />
  const clues = typeBanks[type]
  const current = n ? Number(n) : 1
  if (!Number.isInteger(current) || current < 1 || current > clues.length) return <Navigate to={typeHref(type)} replace />
  const clue = clues[current - 1]
  const last = current >= clues.length
  return (
    <main className="play-shell">
      <KidBar
        label={typeTitles[type]}
        n={current}
        total={clues.length}
        prev={typePrevHref(type, current)}
        next={typeNextHref(type, current)}
        nextLabel={last ? '查看结果' : '下一题'}
      />
      <TypeBoard type={type} n={current} clue={clue} />
    </main>
  )
}

function ClueBoard({ code, n, clue }: { code: string; n: number; clue: Clue }) {
  const key = pickKey(code, n)
  const pick = usePoemStore((s) => s.picks[key])
  const sequence = usePoemStore((s) => s.sequence[key]) ?? EMPTY_SEQUENCE
  const wrong = usePoemStore((s) => s.wrongs[key])
  const setPick = usePoemStore((s) => s.setPick)
  const recite = usePoemStore((s) => s.recite)
  const pictureOptions = clue.options.some((option) => option.visual !== 'char' && option.visual !== 'line')
  const feedback = clue.kind === 'recite'
    ? (pick === clue.answerId ? '答对了' : wrong ? '再试一次' : '')
    : !pick ? '' : pick === clue.answerId ? '答对了' : '再试一次'

  function choose(id: string) {
    if (clue.kind === 'recite' && clue.sequence?.length) {
      recite(code, n, id, clue.sequence, clue.answerId)
      return
    }
    setPick(code, n, id)
  }

  return (
    <section className={`clue-board kind-${clue.kind}`}>
      <aside className="poem-scroll">
        <p className="clue-line">{clue.line}</p>
      </aside>
      <div className="clue-main">
        <div className="clue-prompt">
          <p className="poem-audio-missing" role="status">读音素材暂不可用</p>
          <div>
            <p className="clue-kicker">{taskLabel(clue.kind)}</p>
            <h1>{clue.prompt}</h1>
          </div>
        </div>
        {clue.kind === 'recite' && clue.sequence ? (
          <ol className="order-slots">
            {clue.sequence.map((line, index) => (
              <li key={line} aria-label={`第 ${index + 1} 句`}>
                <b>{index + 1}</b>
                <span>{sequence[index] ?? '？'}</span>
              </li>
            ))}
          </ol>
        ) : null}
        <div className={`option-grid cols-${Math.min(clue.options.length, 4)}${clue.kind === 'recite' || clue.kind === 'couplet' || clue.kind === 'title' || clue.kind === 'author' || clue.kind === 'fill' ? ' is-lines' : ''}`}>
          {clue.options.map((option) => {
            const selected = clue.kind === 'recite' ? sequence.includes(option.id) : pick === option.id
            const right = clue.kind === 'recite'
              ? pick === clue.answerId && sequence.includes(option.id)
              : pick === option.id && option.id === clue.answerId
            const isWrong = clue.kind === 'recite'
              ? wrong === option.id && pick !== clue.answerId
              : pick === option.id && option.id !== clue.answerId
            return (
              <button
                key={option.id}
                className={`option-card${selected ? ' is-picked' : ''}${right ? ' is-right' : ''}${isWrong ? ' is-wrong' : ''}`}
                aria-label={option.label}
                aria-pressed={selected}
                onClick={() => choose(option.id)}
              >
                {pictureOptions ? <InkMark visual={option.visual} label={option.label} /> : null}
                <strong>{option.label}</strong>
              </button>
            )
          })}
        </div>
        {clue.kind === 'recite' && clue.sequence ? <p className="tap-count" aria-live="polite">已排 {sequence.length} / {clue.sequence.length} 句</p> : null}
        {feedback ? <p className="kid-feedback" role="status">{feedback}</p> : null}
      </div>
    </section>
  )
}

function pickedLabel(clue: Clue, pick: string | undefined, sequence: string[]) {
  if (clue.kind === 'recite') return sequence.length ? sequence.join(' → ') : '未选'
  if (!pick) return '未选'
  return clue.options.find((option) => option.id === pick)?.label ?? pick
}

function ResultPage() {
  const { code } = useParams()
  const pavilion = pavilionByCode(code)
  const picks = usePoemStore((s) => s.picks)
  const sequences = usePoemStore((s) => s.sequence)
  if (!pavilion) return <Navigate to="/" replace />
  const found = pavilion.clues.filter((clue, index) => picks[pickKey(pavilion.code, index + 1)] === clue.answerId).length
  return (
    <main className="play-shell result-shell">
      <nav className="kid-top" aria-label="结果导航">
        <Link className="kid-circle" to="/" aria-label="返回"><IconClose /></Link>
      </nav>
      <section className="result-board">
        <h1>答题结果</h1>
        <p>答对 {found} / {pavilion.clues.length} 题</p>
        <ol>
          {pavilion.clues.map((clue, index) => {
            const pick = picks[pickKey(pavilion.code, index + 1)]
            const sequence = sequences[pickKey(pavilion.code, index + 1)] ?? []
            const state = !pick ? '未答' : pick === clue.answerId ? '答对了' : '再想想'
            return (
              <li key={clue.id}>
                <span>{clue.prompt}<em>你选了 {pickedLabel(clue, pick, sequence)}</em></span>
                <b>{state}</b>
              </li>
            )
          })}
        </ol>
        <div className="result-actions">
          <Link to={clueHref(pavilion.code)}>再做一次</Link>
          <Link to="/">回到首页</Link>
        </div>
      </section>
    </main>
  )
}

function TypeResultPage() {
  const { type } = useParams()
  const picks = usePoemStore((s) => s.picks)
  const sequences = usePoemStore((s) => s.sequence)
  const clearType = usePoemStore((s) => s.clearType)
  if (!isPracticeType(type)) return <Navigate to="/" replace />
  const clues = typeBanks[type]
  const found = clues.filter((clue, index) => picks[pickKey(type, index + 1)] === clue.answerId).length
  return (
    <main className="play-shell result-shell">
      <nav className="kid-top" aria-label="结果导航">
        <Link className="kid-circle" to="/" aria-label="返回"><IconClose /></Link>
      </nav>
      <section className="result-board">
        <h1>答题结果</h1>
        <p>答对 {found} / {clues.length} 题</p>
        <ol>
          {clues.map((clue, index) => {
            const pick = picks[pickKey(type, index + 1)]
            const sequence = sequences[pickKey(type, index + 1)] ?? []
            const state = !pick ? '未答' : pick === clue.answerId ? '答对了' : '再想想'
            return (
              <li key={clue.id}>
                <span>{clue.prompt}<em>你选了 {pickedLabel(clue, pick, sequence)}</em></span>
                <b>{state}</b>
              </li>
            )
          })}
        </ol>
        <div className="result-actions">
          <Link to={typeHref(type)} onClick={() => clearType(type)}>再做一次</Link>
          <Link to="/">回到首页</Link>
        </div>
      </section>
    </main>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/practice/type/:type/result" element={<TypeResultPage />} />
      <Route path="/practice/type/:type/:n?" element={<TypePlayPage />} />
      <Route path="/pavilions/:code/result" element={<ResultPage />} />
      <Route path="/pavilions/:code/:n" element={<PlayPage />} />
      <Route path="/pavilions/:code" element={<PlayPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
