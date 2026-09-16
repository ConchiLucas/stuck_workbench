import { useEffect, useState } from 'react'
import type { PinyinGeneratedQuiz, PinyinGeneratedQuizOption } from '../../api/pinyinTypes'

type AnswerState = 'idle' | 'correct' | 'wrong'

export function GeneratedQuizPreview({ question }: { question: PinyinGeneratedQuiz }) {
  const [selected, setSelected] = useState<number | null>(null)
  const [answer, setAnswer] = useState<AnswerState>('idle')
  const audioOnly = question.type === 'shape' || question.type === 'blend'

  useEffect(() => {
    setSelected(null)
    setAnswer('idle')
  }, [question.instanceId])

  function pick(option: PinyinGeneratedQuizOption, index: number) {
    if (answer !== 'idle') return
    if (audioOnly) void playQuizAudio(option.speechUrl, option.speechText)
    setSelected(option.id)
    setAnswer(index === question.answerIndex ? 'correct' : 'wrong')
  }

  return (
    <div className="generated-pinyin-quiz">
      <div className="generated-pinyin-stem">
        <span className="quiz-type-tag">随机生成 · 数据库素材</span>
        <h3>{question.stem}</h3>
        <QuestionVisual question={question} />
      </div>
      <div className={`generated-pinyin-options${audioOnly ? ' is-audio-only' : ''}`} aria-label="选择答案">
        {question.options.map((option, index) => {
          let className = audioOnly ? 'mock-audio-option' : 'generated-letter-option'
          if (selected !== null && index === question.answerIndex) className += ' is-correct'
          else if (selected === option.id) className += ' is-wrong'
          return (
            <button
              key={`${question.instanceId}-${option.id}`}
              type="button"
              className={className}
              aria-label={audioOnly ? `播放读音 ${index + 1}` : undefined}
              disabled={answer !== 'idle'}
              onClick={() => pick(option, index)}
            >
              {audioOnly ? (
                <><span aria-hidden="true">🔊</span><strong>读音 {index + 1}</strong></>
              ) : (
                <strong>{option.label}</strong>
              )}
            </button>
          )
        })}
      </div>
      {answer !== 'idle' ? (
        <div className={`mock-quiz-feedback ${answer === 'correct' ? 'ok' : 'err'}`}>
          {answer === 'correct' ? '答对了！' : '再想想，换一个答案试试。'}
        </div>
      ) : (
        <p className="muted mock-quiz-hint">
          {audioOnly ? '点击读音按钮试听并选择答案；选项不显示文字内容。' : '点击一个选项完成试答。'}
        </p>
      )}
    </div>
  )
}

function QuestionVisual({ question }: { question: PinyinGeneratedQuiz }) {
  const playStem = () => void playQuizAudio(question.speechUrl, question.speechText)
  if (question.type === 'shape') {
    return question.visual.imageUrl ? (
      <img className="generated-glyph-visual" src={question.visual.imageUrl} alt={question.visual.text ?? '拼音字形'} />
    ) : (
      <div className="mock-four-line-grid"><strong>{question.visual.text}</strong></div>
    )
  }
  if (question.type === 'blend') {
    return (
      <div className="mock-blend-stem">
        <strong>{question.visual.initial}</strong><span>+</span>
        <strong>{question.visual.final}</strong><span>=</span><strong>?</strong>
      </div>
    )
  }
  return (
    <div className="generated-speech-visual">
      {question.visual.text ? <strong>{question.visual.text}</strong> : null}
      <button type="button" className="quiz-speak-btn ghost" onClick={playStem}>🔊</button>
    </div>
  )
}

async function playQuizAudio(url?: string, speechText?: string) {
  if (url) {
    const res = await fetch(url)
    if (res.ok) {
      const blob = await res.blob()
      const objectURL = URL.createObjectURL(blob)
      const audio = new Audio(objectURL)
      audio.onended = () => URL.revokeObjectURL(objectURL)
      await audio.play()
      return
    }
  }
  if (!speechText || !('speechSynthesis' in window)) return
  window.speechSynthesis.cancel()
  const utterance = new SpeechSynthesisUtterance(speechText)
  utterance.lang = 'zh-CN'
  utterance.rate = 0.72
  window.speechSynthesis.speak(utterance)
}
