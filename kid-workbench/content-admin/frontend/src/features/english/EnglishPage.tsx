import { useQuery } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { EnglishPlayer, englishTitles, type EnglishExample, type EnglishKind } from '@kid-workbench/english-player'
import { listEnglish, speechAudioURL } from '../../api/english'
import type { EnglishPassage, EnglishSentence, EnglishWord } from '../../api/englishTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'
import { appPath } from '../../appPath'
import '@kid-workbench/english-player/player.css'

async function playURL(url: string) {
  const res = await fetch(url)
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(typeof body.error === 'string' ? body.error : `读音失败 ${res.status}`)
  }
  const blob = await res.blob()
  const objectURL = URL.createObjectURL(blob)
  const audio = new Audio(objectURL)
  audio.onended = () => URL.revokeObjectURL(objectURL)
  await audio.play()
}

async function playSpeech(kpId: number, speechAudioUrl?: string) {
  await playURL(speechAudioURL(kpId, speechAudioUrl))
}

export function EnglishPage() {
  const [q, setQ] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [speakingKp, setSpeakingKp] = useState<number | null>(null)
  const [speakingSentence, setSpeakingSentence] = useState<number | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [openWord, setOpen] = useState<EnglishWord | null>(null)
  const [openSentence, setOpenSentence] = useState<EnglishSentence | null>(null)
  const [openPassage, setOpenPassage] = useState<EnglishPassage | null>(null)

  const listQuery = useQuery({
    queryKey: ['english', 'words', 'groups'],
    queryFn: () => listEnglish({ view: 'groups' }),
  })

  const onPlaySpeech = async (word: EnglishWord) => {
    setSpeechError('')
    setSpeakingKp(word.kpId)
    try {
      await playSpeech(word.kpId, word.speechAudioUrl)
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingKp(null)
    }
  }

  const onPlaySentence = async (sentence: EnglishSentence) => {
    setSpeechError('')
    setSpeakingSentence(sentence.id)
    try {
      await playURL(appPath(`/api/v1/english/sentences/${sentence.id}/speech.mp3`))
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingSentence(null)
    }
  }

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    ''

  const groups = (listQuery.data?.groups ?? [])
    .map((g) => ({
      ...g,
      words: g.words.filter(
        (w) => !q.trim() || w.wordText.toLowerCase().includes(q.trim().toLowerCase()),
      ),
    }))
    .filter((g) => g.words.length > 0)
  const allWords = (listQuery.data?.groups ?? []).flatMap((g) => g.words)
  const byId = new Map(allWords.map((w) => [w.kpId, w]))
  const needle = q.trim().toLowerCase()
  const sentences = (listQuery.data?.sentences ?? []).filter((item) =>
    !needle || item.text.toLowerCase().includes(needle) || item.targetWord?.toLowerCase().includes(needle),
  )
  const passages = (listQuery.data?.passages ?? []).filter((item) =>
    !needle || item.passage.toLowerCase().includes(needle) || item.prompt.includes(needle),
  )

  return (
    <section className="literacy-page" aria-label="英语素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">MATERIALS / ENGLISH</p>
          <h1>英语素材</h1>
          <p className="page-description">
            浏览已有字图、义图、单词读音、完整句子和阅读短文。素材由开发流程准备，本页不生成、不编辑、不发布。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.english} />
        </div>
      </div>

      <div className="literacy-toolbar">
        <input
          className="search-input"
          placeholder="搜索单词、句子或短文"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 词` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。</div>
      ) : (
        <div className="group-list">
          {groups.map((g) => {
            const closed = collapsed[g.moduleCode]
            return (
              <section key={g.moduleCode} className="literacy-group">
                <div className="group-header-row">
                  <button
                    type="button"
                    className="group-header"
                    onClick={() =>
                      setCollapsed((prev) => ({ ...prev, [g.moduleCode]: !prev[g.moduleCode] }))
                    }
                  >
                    <span>{g.moduleName}</span>
                    <span className="muted">
                      {g.words.length} 词 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                </div>
                {!closed ? (
                  <div className="char-grid">
                    {g.words.map((w) => (
                      <WordCard
                        key={w.kpId}
                        word={w}
                        speaking={speakingKp === w.kpId}
                        onPlaySpeech={() => void onPlaySpeech(w)}
                        onOpen={() => setOpen(w)}
                      />
                    ))}
                  </div>
                ) : null}
              </section>
            )
          })}
          {sentences.length ? (
            <section className="literacy-group">
              <div className="group-header-row">
                <h2>组句子素材</h2>
                <span className="muted">{sentences.length} 句</span>
              </div>
              <div className="char-grid">
                {sentences.map((sentence) => (
                  <article className="char-card" key={sentence.id}>
                    <strong>{sentence.text}</strong>
                    <span className="muted">{sentence.tokens.join(' · ')}</span>
                    <div className="card-actions">
                      <button type="button" className="mini-btn" disabled={speakingSentence === sentence.id || !sentence.speechAudioUrl} onClick={() => void onPlaySentence(sentence)}>
                        {speakingSentence === sentence.id ? '…' : sentence.speechAudioUrl ? '试听整句' : '读音暂不可用'}
                      </button>
                      <button type="button" className="mini-btn" onClick={() => setOpenSentence(sentence)}>查看详情</button>
                    </div>
                  </article>
                ))}
              </div>
            </section>
          ) : null}
          {passages.length ? (
            <section className="literacy-group">
              <div className="group-header-row">
                <h2>读一读素材</h2>
                <span className="muted">{passages.length} 篇</span>
              </div>
              <div className="char-grid">
                {passages.map((passage) => (
                  <article className="char-card" key={passage.id}>
                    <strong>{passage.prompt}</strong>
                    <span className="muted">{passage.passage}</span>
                    <div className="card-actions">
                      <button type="button" className="mini-btn" onClick={() => setOpenPassage(passage)}>查看详情</button>
                    </div>
                  </article>
                ))}
              </div>
            </section>
          ) : null}
        </div>
      )}
      {openWord ? <WordDetail word={openWord} siblings={(listQuery.data?.groups ?? []).find((g) => g.moduleCode === openWord.moduleCode)?.words ?? []} sentences={listQuery.data?.sentences ?? []} passages={listQuery.data?.passages ?? []} wordsById={byId} onClose={() => setOpen(null)} onPlaySpeech={() => void onPlaySpeech(openWord)} speaking={speakingKp === openWord.kpId} /> : null}
      {openSentence ? <MaterialDetail title={openSentence.text} examples={[sentenceExample(openSentence)]} onClose={() => setOpenSentence(null)} onPlay={() => void onPlaySentence(openSentence)} speaking={speakingSentence === openSentence.id} canPlay={Boolean(openSentence.speechAudioUrl)} /> : null}
      {openPassage ? <MaterialDetail title={openPassage.prompt} examples={[passageExample(openPassage, byId)]} onClose={() => setOpenPassage(null)} /> : null}
    </section>
  )
}

function WordCard({
  word,
  speaking,
  onPlaySpeech,
  onOpen,
}: {
  word: EnglishWord
  speaking: boolean
  onPlaySpeech: () => void
  onOpen: () => void
}) {
  return (
    <article className="char-card">
      {word.glyphImageUrl ? (
        <img className="glyph-preview" src={word.glyphImageUrl} alt={word.wordText} />
      ) : (
        <div className="char-glyph">{word.wordText}</div>
      )}
      {word.senseImageUrl ? (
        <img className="sense-preview" src={word.senseImageUrl} alt={`${word.wordText}义图`} />
      ) : (
        <div className="sense-placeholder">义图未生成</div>
      )}
      <strong>{word.wordText}</strong>
      {word.meaningZh ? <span className="muted">{word.meaningZh}</span> : null}
      <div className="card-actions">
        <button type="button" className="mini-btn" disabled={speaking || !word.speechAudioUrl} onClick={onPlaySpeech}>
          {speaking ? '…' : word.speechAudioUrl ? '读音' : '读音暂不可用'}
        </button>
        <button type="button" className="mini-btn" onClick={onOpen}>查看详情</button>
      </div>
    </article>
  )
}

function WordDetail({
  word,
  siblings,
  sentences,
  passages,
  wordsById,
  onClose,
  onPlaySpeech,
  speaking,
}: {
  word: EnglishWord
  siblings: EnglishWord[]
  sentences: EnglishSentence[]
  passages: EnglishPassage[]
  wordsById: Map<number, EnglishWord>
  onClose: () => void
  onPlaySpeech: () => void
  speaking: boolean
}) {
  const examples = materialSamples(word, siblings, sentences, passages, wordsById)
  return (
    <MaterialDetail
      title={word.wordText}
      subtitle={word.meaningZh}
      glyph={word.glyphImageUrl}
      sense={word.senseImageUrl}
      examples={examples}
      onClose={onClose}
      onPlay={onPlaySpeech}
      speaking={speaking}
      canPlay={Boolean(word.speechAudioUrl)}
    />
  )
}

function MaterialDetail({
  title,
  subtitle,
  glyph,
  sense,
  examples,
  onClose,
  onPlay,
  speaking,
  canPlay,
}: {
  title: string
  subtitle?: string
  glyph?: string
  sense?: string
  examples: EnglishExample[]
  onClose: () => void
  onPlay?: () => void
  speaking?: boolean
  canPlay?: boolean
}) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null
    const overflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const dialog = ref.current
    dialog?.showModal()
    dialog?.querySelector<HTMLButtonElement>('.task-preview-close')?.focus()
    return () => {
      document.body.style.overflow = overflow
      dialog?.close()
      previous?.focus()
    }
  }, [])
  return (
    <dialog ref={ref} className="task-preview-dialog english-word-detail" aria-label={`${title}素材详情`} onCancel={(event) => { event.preventDefault(); onClose() }}>
      <div className="task-preview-bar">
        <button className="task-preview-close" aria-label="关闭弹窗" onClick={onClose}>×</button>
      </div>
      <div className="task-preview-content english-word-detail-body">
        <h2>{title}</h2>
        {subtitle ? <p>{subtitle}</p> : null}
        {(glyph || sense) ? (
          <div className="english-word-media">
            {glyph ? <img src={glyph} alt={`${title}字图`} /> : <p>字图暂不可用</p>}
            {sense ? <img src={sense} alt={`${title}义图`} /> : <p>义图未生成</p>}
          </div>
        ) : null}
        {onPlay ? (
          <p>
            <button type="button" className="mini-btn" disabled={speaking || !canPlay} onClick={onPlay}>
              {speaking ? '试听中…' : canPlay ? '试听已有读音' : '读音暂不可用'}
            </button>
          </p>
        ) : null}
        {examples.map((example) => (
          <div className="english-material-player" key={example.kind}>
            <h3>{englishTitles[example.kind as EnglishKind]}</h3>
            <EnglishPlayer example={example} readOnly allowSyntheticSpeech={false} resolveAssetUrl={(url) => appPath(url)} />
          </div>
        ))}
        {!examples.length ? <p className="muted">同组还没有足够的义图或句子/短文素材，无法展示完整题面示例。</p> : null}
      </div>
    </dialog>
  )
}

function sentenceExample(sentence: EnglishSentence): EnglishExample {
  return {
    kind: 'card-builder',
    prompt: '把单词排成一句话',
    speech: sentence.text,
    speechUrl: sentence.speechAudioUrl ? `/api/v1/english/sentences/${sentence.id}/speech.mp3` : undefined,
    bank: sentence.tokens,
    answer: sentence.text,
  }
}

function passageExample(passage: EnglishPassage, wordsById: Map<number, EnglishWord>): EnglishExample {
  const options = passage.optionKpIds.map((id) => {
    const word = wordsById.get(id)
    return {
      id: String(id),
      label: word?.meaningZh || word?.wordText || String(id),
      picture: word?.senseImageUrl,
    }
  })
  return {
    kind: 'reading-qa',
    prompt: passage.prompt,
    passage: passage.passage,
    options,
    answerId: String(passage.answerKpId),
  }
}

function materialSamples(
  word: EnglishWord,
  siblings: EnglishWord[],
  sentences: EnglishSentence[],
  passages: EnglishPassage[],
  wordsById: Map<number, EnglishWord>,
): EnglishExample[] {
  const pictured = siblings.filter((item) => item.senseImageUrl)
  const others = pictured.filter((item) => item.kpId !== word.kpId).slice(0, 3)
  const out: EnglishExample[] = []
  if (word.senseImageUrl && others.length >= 3) {
    const options = [word, ...others].map((item) => ({
      id: String(item.kpId),
      label: item.meaningZh || item.wordText,
      picture: item.senseImageUrl,
    }))
    const look = [word, ...others].map((item) => ({
      id: String(item.kpId),
      label: item.wordText,
      picture: item.senseImageUrl,
    }))
    out.push({
      kind: 'audio-choice',
      speech: word.wordText,
      speechUrl: word.speechAudioUrl ? `/api/v1/english/words/${word.kpId}/speech.mp3` : undefined,
      options,
      answerId: String(word.kpId),
    })
    out.push({
      kind: 'image-text',
      prompt: `哪一张图是 ${word.wordText}？`,
      options: look,
      answerId: String(word.kpId),
    })
  }
  const sentence = sentences.find((item) => item.targetKpId === word.kpId)
  if (sentence) out.push(sentenceExample(sentence))
  if (word.senseImageUrl && word.speechAudioUrl) {
    out.push({
      kind: 'input-gap',
      prompt: '写出这个单词',
      speech: word.wordText,
      speechUrl: `/api/v1/english/words/${word.kpId}/speech.mp3`,
      cue: word.senseImageUrl,
      answer: word.wordText,
    })
  }
  const passage = passages.find((item) => item.answerKpId === word.kpId)
  if (passage) out.push(passageExample(passage, wordsById))
  return out
}
