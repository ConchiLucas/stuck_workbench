import { SyllableMaterials } from './SyllableMaterials'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { glyphImageURL, batchGenerateSpeech, importHumanPack, listPinyin, speechAudioURL } from '../../api/pinyin'
import type { PinyinItem } from '../../api/pinyinTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'

function exampleWordsOf(item: PinyinItem): string[] {
  const listed = item.wordExamples?.filter(Boolean) ?? []
  if (listed.length > 0) return listed
  return item.wordText ? [item.wordText] : []
}

function stopSpokenText() {
  if ('speechSynthesis' in window) window.speechSynthesis.cancel()
}

async function playRecordedSpeech(kpId: number, kind: string, speechUrl?: string) {
  const res = await fetch(speechAudioURL(kpId, kind, speechUrl))
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(typeof body.error === 'string' ? body.error : `读音失败 ${res.status}`)
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  await new Promise<void>((resolve, reject) => {
    const audio = new Audio(url)
    audio.onended = () => {
      URL.revokeObjectURL(url)
      resolve()
    }
    audio.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('读音失败'))
    }
    void audio.play().catch((err) => {
      URL.revokeObjectURL(url)
      reject(err instanceof Error ? err : new Error('读音失败'))
    })
  })
}

function exampleSpeechKind(item: PinyinItem, word: string) {
  const index = exampleWordsOf(item).indexOf(word)
  return index <= 0 ? 'word' : `word-${index}`
}

function exampleSpeechUrl(item: PinyinItem, word: string) {
  return item.wordExampleSpeechUrls?.[word] || (word === item.wordText ? item.wordSpeechUrl : '')
}

async function playExampleWord(item: PinyinItem, word: string) {
  await playRecordedSpeech(item.kpId, exampleSpeechKind(item, word), exampleSpeechUrl(item, word))
}

export function PinyinPage() {
  const queryClient = useQueryClient()
  const playSeq = useRef(0)
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [speakingKey, setSpeakingKey] = useState<string | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [busySpeechModule, setBusySpeechModule] = useState('')
  const [busyHumanModule, setBusyHumanModule] = useState('')

  const listQuery = useQuery({
    queryKey: ['pinyin', 'items', 'groups'],
    queryFn: () => listPinyin({ view: 'groups' }),
  })

  const speechBatchMutation = useMutation({
    mutationFn: (moduleCode: string) => batchGenerateSpeech(moduleCode),
    onMutate: (moduleCode) => setBusySpeechModule(moduleCode),
    onSettled: () => setBusySpeechModule(''),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pinyin'] }),
  })

  const humanPackMutation = useMutation({
    mutationFn: (moduleCode: string) => importHumanPack(moduleCode),
    onMutate: (moduleCode) => setBusyHumanModule(moduleCode),
    onSettled: () => setBusyHumanModule(''),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pinyin'] }),
  })

  const onPlaySpeech = async (item: PinyinItem, kind: 'solo' | 'word', word?: string) => {
    setSpeechError('')
    stopSpokenText()
    const seq = ++playSeq.current
    const key = kind === 'solo' ? `${item.kpId}-solo` : `${item.kpId}-word-${word ?? item.wordText}`
    setSpeakingKey(key)
    try {
      if (kind === 'solo') {
        await playRecordedSpeech(item.kpId, 'solo', item.soloSpeechUrl)
      } else if (word) {
        await playExampleWord(item, word)
        if (seq === playSeq.current) {
          await queryClient.invalidateQueries({ queryKey: ['pinyin'] })
        }
      }
    } catch (e) {
      if (seq === playSeq.current) {
        setSpeechError(e instanceof Error ? e.message : '读音失败')
      }
    } finally {
      if (seq === playSeq.current) setSpeakingKey(null)
    }
  }

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (speechBatchMutation.error instanceof Error && speechBatchMutation.error.message) ||
    (humanPackMutation.error instanceof Error && humanPackMutation.error.message) ||
    ''

  return (
    <section className="literacy-page pinyin-page" aria-label="拼音素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">MATERIALS / PINYIN</p>
          <h1>拼音素材</h1>
          <p className="page-description">
            管理字母、例字与音节录音。点例字可试听；音节使用真人录音。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.pinyin} />
        </div>
      </div>

      <div className="literacy-toolbar">
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 音` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}
      {speechBatchMutation.data ? (
        <div className="info-panel">
          批量读音：生成 {speechBatchMutation.data.generated}，跳过 {speechBatchMutation.data.skipped}
          ，失败 {speechBatchMutation.data.failed}
          {speechBatchMutation.data.errors?.length
            ? ` · ${speechBatchMutation.data.errors.join('；')}`
            : ''}
        </div>
      ) : null}
      {humanPackMutation.data ? (
        <div className="info-panel">
          真人包：导入 {humanPackMutation.data.generated}，跳过 {humanPackMutation.data.skipped}
          ，失败 {humanPackMutation.data.failed}
          {humanPackMutation.data.errors?.length
            ? ` · ${humanPackMutation.data.errors.join('；')}`
            : ''}
        </div>
      ) : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。</div>
      ) : (
        <div className="group-list">
          {(listQuery.data.groups ?? []).map((g) => {
            const closed = collapsed[g.moduleCode]
            const speechBusy = busySpeechModule === g.moduleCode
            const humanBusy = busyHumanModule === g.moduleCode
            return (
              <section key={g.moduleCode} className="literacy-group">
                <div className="group-header-row">
                  <button
                    type="button"
                    className="group-header"
                    aria-expanded={!closed}
                    onClick={() =>
                      setCollapsed((prev) => ({ ...prev, [g.moduleCode]: !prev[g.moduleCode] }))
                    }
                  >
                    <span>{g.moduleName}</span>
                    <span className="muted">
                      {g.items.length} 音 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                  <button
                    type="button"
                    className="mini-btn"
                    disabled={humanBusy || humanPackMutation.isPending}
                    onClick={() => humanPackMutation.mutate(g.moduleCode)}
                  >
                    {humanBusy ? '导入中…' : '导入真人包'}
                  </button>
                  <button
                    type="button"
                    className="mini-btn"
                    disabled={speechBusy || speechBatchMutation.isPending}
                    onClick={() => speechBatchMutation.mutate(g.moduleCode)}
                  >
                    {speechBusy ? '生成中…' : '生成本组读音'}
                  </button>
                </div>
                {closed ? null : (
                  <div className="char-grid">
                    {g.items.map((item) => (
                      <PinyinCard
                        key={item.kpId}
                        item={item}
                        speakingKey={speakingKey}
                        onPlay={(kind, word) => void onPlaySpeech(item, kind, word)}
                      />
                    ))}
                  </div>
                )}
              </section>
            )
          })}
        </div>
      )}

      <SyllableMaterials />
    </section>
  )
}

function PinyinCard({
  item,
  speakingKey,
  onPlay,
}: {
  item: PinyinItem
  speakingKey: string | null
  onPlay: (kind: 'solo' | 'word', word?: string) => void
}) {
  const soloKey = `${item.kpId}-solo`
  const hasSolo = Boolean(item.soloText)
  const exampleWords = exampleWordsOf(item)

  return (
    <article className="char-card">
      {item.glyphImageUrl ? (
        <img
          className="glyph-preview"
          src={glyphImageURL(item.kpId, item.glyphImageUrl)}
          alt={item.letter}
        />
      ) : (
        <div className="char-glyph">{item.letter}</div>
      )}
      <div className="sense-tag">
        {hasSolo ? `单读 ${item.soloText}` : '无单读'}
        {item.soloSpeechUrl || item.wordSpeechUrl ? ' · 有读音' : ''}
      </div>
      {exampleWords.length > 0 ? (
        <div className="pinyin-example-words">
          <span className="muted">例字</span>
          {exampleWords.map((word) => {
            const wordKey = `${item.kpId}-word-${word}`
            const hasSpeech = Boolean(exampleSpeechUrl(item, word))
            return (
              <button
                key={wordKey}
                type="button"
                className="mini-btn"
                aria-label={`读例字 ${word}`}
                disabled={speakingKey === wordKey}
                onClick={() => onPlay('word', word)}
              >
                {speakingKey === wordKey ? '…' : hasSpeech ? `${word}✓` : word}
              </button>
            )
          })}
        </div>
      ) : null}
      <div className="card-actions">
        <button
          type="button"
          className="mini-btn"
          disabled={!hasSolo || speakingKey === soloKey}
          onClick={() => onPlay('solo')}
        >
          {speakingKey === soloKey ? '…' : item.soloSpeechUrl ? '读单读✓' : '读单读'}
        </button>
      </div>
    </article>
  )
}
