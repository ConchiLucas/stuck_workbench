import { useEffect, useState } from 'react'
import {
  ENGLISH_QUESTION_SCRIPTS,
  type EnglishQuestionScript,
} from './englishQuestionTypes'

export function EnglishQuestionTypeSheet({ onClose }: { onClose: () => void }) {
  const [activeId, setActiveId] = useState<string | null>(null)
  const activeScript = ENGLISH_QUESTION_SCRIPTS.find((script) => script.id === activeId) ?? null

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      if (activeId) setActiveId(null)
      else onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [activeId, onClose])

  return (
    <div className="fullscreen-sheet" role="dialog" aria-modal="true" aria-labelledby="english-question-sheet-title">
      <header className="fullscreen-sheet-header">
        <div>
          {activeScript ? (
            <button type="button" className="sheet-back" onClick={() => setActiveId(null)}>
              ← 返回题型
            </button>
          ) : null}
          <h2 id="english-question-sheet-title">
            {activeScript ? activeScript.title : '英语通用题型脚本'}
          </h2>
          {!activeScript ? (
            <p className="muted">只保留最值得开发的通用脚本；点击任意卡片可直接查看答题效果。</p>
          ) : null}
        </div>
        <button type="button" className="refresh-button group-preview-link" onClick={onClose}>关闭</button>
      </header>

      <div className="fullscreen-sheet-body">
        {activeScript ? (
          <EnglishScriptDetail script={activeScript} />
        ) : (
          <div className="pinyin-quiz-type-grid english-question-type-grid">
            {ENGLISH_QUESTION_SCRIPTS.map((script) => (
              <button
                key={script.id}
                type="button"
                className="pinyin-quiz-type-card english-question-type-card"
                onClick={() => setActiveId(script.id)}
                aria-label={`${script.title}，查看效果`}
              >
                <span className="pinyin-quiz-type-index">{String(script.ordinal).padStart(2, '0')} · {script.code}</span>
                <span className="pinyin-quiz-type-badge">优先开发</span>
                <strong>{script.title}</strong>
                <span>{script.action}</span>
                <span className="english-script-goal">学习判断：{script.goal}</span>
                <span className="pinyin-quiz-type-example">覆盖 {script.coveredTypes.length} 类题型 →</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function EnglishScriptDetail({ script }: { script: EnglishQuestionScript }) {
  return (
    <div className="english-script-detail">
      <section className="english-effect-panel" aria-label="题目效果预览">
        <div className="english-effect-heading">
          <span className="quiz-type-tag">效果预览 · 示例数据</span>
          <p>{script.category} / {script.code}</p>
          <h3>{script.action}</h3>
        </div>
        <QuestionEffect scriptId={script.id} />
      </section>

      <aside className="english-diagnosis-panel">
        <section>
          <span className="english-detail-label">训练目标</span>
          <h3>{script.goal}</h3>
        </section>
        <section>
          <h4>覆盖题型</h4>
          <div className="english-type-chips">
            {script.coveredTypes.map((type) => <span key={type}>{type}</span>)}
          </div>
        </section>
        <section>
          <h4>答错后怎么判断</h4>
          <ul>{script.diagnosis.map((item) => <li key={item}>{item}</li>)}</ul>
        </section>
        <section className="english-next-step">
          <h4>下一步怎么学</h4>
          <p>{script.nextStep}</p>
        </section>
      </aside>
    </div>
  )
}

function QuestionEffect({ scriptId }: { scriptId: string }) {
  const [selected, setSelected] = useState<string | null>(null)
  const [typed, setTyped] = useState('')
  const [checked, setChecked] = useState(false)

  const choose = (value: string) => {
    setSelected(value)
    setChecked(true)
  }

  if (scriptId === 'audio-choice') {
    return (
      <div className="english-demo-question">
        <button type="button" className="english-play-button" aria-label="播放示例音频" onClick={() => speak('apple')}>▶ 播放 apple</button>
        <p>听一听，选择正确的图片</p>
        <DemoOptions values={[['🍎', 'apple'], ['🍌', 'banana'], ['🐶', 'dog']]} selected={selected} answer="apple" onChoose={choose} />
        <Feedback checked={checked} correct={selected === 'apple'} />
      </div>
    )
  }
  if (scriptId === 'image-text') {
    return (
      <div className="english-demo-question">
        <div className="english-demo-hero" aria-label="小猫图片">🐱</div>
        <p>看图片，选择对应的英文</p>
        <DemoOptions values={[['cat', 'cat'], ['cap', 'cap'], ['dog', 'dog']]} selected={selected} answer="cat" onChoose={choose} />
        <Feedback checked={checked} correct={selected === 'cat'} />
      </div>
    )
  }
  if (scriptId === 'sound-discrimination') {
    return (
      <div className="english-demo-question">
        <button type="button" className="english-play-button" onClick={() => speak('cat')}>▶ 播放 cat</button>
        <p>哪个单词和 cat 有相同的尾音？</p>
        <DemoOptions values={[['hat', 'hat'], ['dog', 'dog'], ['sun', 'sun']]} selected={selected} answer="hat" onChoose={choose} />
        <Feedback checked={checked} correct={selected === 'hat'} />
      </div>
    )
  }
  if (scriptId === 'card-builder') {
    return (
      <div className="english-demo-question">
        <p>依次点击卡片，拼出 cat</p>
        <div className="english-card-answer">{selected || '＿ ＿ ＿'}</div>
        <div className="english-token-row">
          {['c', 'a', 't'].map((token) => <button key={token} type="button" onClick={() => { const next = `${selected ?? ''}${token}`; setSelected(next); setChecked(next.length === 3) }}>{token}</button>)}
          <button type="button" className="subtle" onClick={() => { setSelected(null); setChecked(false) }}>重来</button>
        </div>
        <Feedback checked={checked} correct={selected === 'cat'} />
      </div>
    )
  }
  if (scriptId === 'input-gap') {
    return (
      <div className="english-demo-question">
        <div className="english-demo-hero">🐶</div>
        <p>补全单词：d _ g</p>
        <div className="english-input-row">
          <input aria-label="输入缺少的字母" maxLength={1} value={typed} onChange={(event) => { setTyped(event.target.value); setChecked(false) }} />
          <button type="button" onClick={() => setChecked(true)}>检查</button>
        </div>
        <Feedback checked={checked} correct={typed.toLowerCase() === 'o'} />
      </div>
    )
  }
  if (scriptId === 'reading-qa') {
    return (
      <div className="english-demo-question">
        <blockquote>Tom has a red ball. He plays with it in the park.</blockquote>
        <p>What color is Tom&apos;s ball?</p>
        <DemoOptions values={[['red', 'red'], ['blue', 'blue'], ['green', 'green']]} selected={selected} answer="red" onChoose={choose} />
        <Feedback checked={checked} correct={selected === 'red'} />
      </div>
    )
  }
  return (
    <div className="english-demo-question english-path-demo">
      <p className="english-error-case">连续两次把 <strong>ship</strong> 听成 <strong>sheep</strong></p>
      <div><span>发现</span><strong>短元音 /ɪ/ 与长元音 /iː/ 分辨不稳</strong></div>
      <div><span>下一题</span><strong>降级到图片辅助的最小音对比</strong></div>
      <div><span>复测</span><strong>换一组词再次判断是否掌握</strong></div>
    </div>
  )
}

function DemoOptions({ values, selected, answer, onChoose }: {
  values: string[][]
  selected: string | null
  answer: string
  onChoose: (value: string) => void
}) {
  return (
    <div className="english-demo-options">
      {values.map(([label, value]) => {
        const state = selected && value === answer ? ' is-correct' : selected === value ? ' is-wrong' : ''
        return <button key={value} type="button" className={state} onClick={() => onChoose(value)}>{label}</button>
      })}
    </div>
  )
}

function Feedback({ checked, correct }: { checked: boolean; correct: boolean }) {
  if (!checked) return <p className="muted english-demo-hint">点击答案，查看即时反馈。</p>
  return <p className={`mock-quiz-feedback ${correct ? 'ok' : 'err'}`}>{correct ? '答对了！可以进入变式题。' : '还没掌握：记录错误类型，再安排下一步练习。'}</p>
}

function speak(text: string) {
  if (!('speechSynthesis' in window)) return
  window.speechSynthesis.cancel()
  const utterance = new SpeechSynthesisUtterance(text)
  utterance.lang = 'en-US'
  utterance.rate = 0.75
  window.speechSynthesis.speak(utterance)
}

