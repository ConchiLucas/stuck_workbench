import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  batchGenerateGlyphs,
  batchGenerateQuestionSpeech,
  batchGenerateSpeech,
  glyphImageURL,
  getQuestionSpeechStatus,
  listMath,
  speechAudioURL,
} from '../../api/math'
import type { MathItem } from '../../api/mathTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'

const KIND_LABEL: Record<string, string> = {
  add: '加法',
  sub: '减法',
}

function kindLabel(kind: string) {
  if (!kind) return '图形'
  return KIND_LABEL[kind] ?? kind
}

async function playSpeech(kpId: number, speechUrl?: string) {
  const res = await fetch(speechAudioURL(kpId, speechUrl))
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(typeof body.error === 'string' ? body.error : `读音失败 ${res.status}`)
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const audio = new Audio(url)
  audio.onended = () => URL.revokeObjectURL(url)
  await audio.play()
}

export function MathPage() {
  const queryClient = useQueryClient()
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [busyGlyphModule, setBusyGlyphModule] = useState<string | null>(null)
  const [busySpeechModule, setBusySpeechModule] = useState<string | null>(null)
  const [speakingKp, setSpeakingKp] = useState<number | null>(null)
  const [speechError, setSpeechError] = useState('')


  const listQuery = useQuery({
    queryKey: ['math', 'items', 'groups'],
    queryFn: () => listMath({ view: 'groups' }),
  })

  const glyphBatchMutation = useMutation({
    mutationFn: (moduleCode: string) => batchGenerateGlyphs(moduleCode),
    onMutate: (moduleCode) => setBusyGlyphModule(moduleCode),
    onSettled: () => {
      setBusyGlyphModule(null)
      void queryClient.invalidateQueries({ queryKey: ['math'] })
    },
  })

  const speechBatchMutation = useMutation({
    mutationFn: (moduleCode: string) => batchGenerateSpeech(moduleCode),
    onMutate: (moduleCode) => setBusySpeechModule(moduleCode),
    onSettled: () => {
      setBusySpeechModule(null)
      void queryClient.invalidateQueries({ queryKey: ['math'] })
    },
  })

  const onPlaySpeech = async (item: MathItem) => {
    setSpeechError('')
    setSpeakingKp(item.kpId)
    try {
      await playSpeech(item.kpId, item.speechAudioUrl)
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingKp(null)
    }
  }

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (glyphBatchMutation.error instanceof Error && glyphBatchMutation.error.message) ||
    (speechBatchMutation.error instanceof Error && speechBatchMutation.error.message) ||
    ''

  return (
    <section className="literacy-page math-page" aria-label="算术素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">MATERIALS / MATH</p>
          <h1>算术素材</h1>
          <p className="page-description">
            加减法淡蓝方格纸字图；认识图形画真实几何图形；题干 TTS 读音可按组批量生成。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.math} />
        </div>
      </div>


      <h2>字图与读音素材</h2>
      <div className="literacy-toolbar">
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 项素材` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}
      {glyphBatchMutation.data ? (
        <div className="info-panel">
          批量字图：生成 {glyphBatchMutation.data.generated}，失败 {glyphBatchMutation.data.failed}
          {glyphBatchMutation.data.errors?.length
            ? ` · ${glyphBatchMutation.data.errors.join('；')}`
            : ''}
        </div>
      ) : null}
      {speechBatchMutation.data ? (
        <div className="info-panel">
          批量读音：生成 {speechBatchMutation.data.generated}，跳过 {speechBatchMutation.data.skipped}
          ，失败 {speechBatchMutation.data.failed}
          {speechBatchMutation.data.errors?.length
            ? ` · ${speechBatchMutation.data.errors.join('；')}`
            : ''}
        </div>
      ) : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。</div>
      ) : (
        <div className="group-list">
          {(listQuery.data.groups ?? [])
            .filter((g) => g.moduleCode === 'add10' || g.moduleCode === 'sub10' || g.moduleCode === 'shape')
            .map((g) => {
            const closed = collapsed[g.moduleCode]
            const glyphBusy = busyGlyphModule === g.moduleCode
            const speechBusy = busySpeechModule === g.moduleCode
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
                      {g.items.length} 项素材 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                  <button
                    type="button"
                    className="mini-btn"
                    disabled={glyphBusy || glyphBatchMutation.isPending}
                    onClick={() => glyphBatchMutation.mutate(g.moduleCode)}
                  >
                    {glyphBusy ? '生成中…' : '生成本组字图'}
                  </button>
                  <button
                    type="button"
                    className="mini-btn"
                    disabled={speechBusy || speechBatchMutation.isPending}
                    onClick={() => speechBatchMutation.mutate(g.moduleCode)}
                  >
                    {speechBusy ? '生成中…' : '生成本组读音'}
                  </button>
                  <QuestionSpeechControls moduleCode={g.moduleCode} />

                </div>
                {closed ? null : (
                  <div className="char-grid math-grid">
                    {g.items.map((item) => (
                      <MathCard
                        key={item.kpId}
                        item={item}
                        speaking={speakingKp === item.kpId}
                        onPlay={() => void onPlaySpeech(item)}
                      />
                    ))}
                  </div>
                )}
              </section>
            )
          })}
        </div>
      )}


    </section>
  )
}

function QuestionSpeechControls({ moduleCode }: { moduleCode: string }) {
  const queryClient = useQueryClient()
  const statusQuery = useQuery({
    queryKey: ['math', 'question-speech', moduleCode],
    queryFn: () => getQuestionSpeechStatus(moduleCode),
  })
  const mutation = useMutation({
    mutationFn: () => batchGenerateQuestionSpeech(moduleCode),
    onSuccess: () => queryClient.invalidateQueries({
      queryKey: ['math', 'question-speech', moduleCode],
    }),
  })

  return (
    <>
      <span className="muted">
        {statusQuery.data
          ? `题目音频 ${statusQuery.data.ready}/${statusQuery.data.total}`
          : statusQuery.isLoading ? '题目音频加载中…' : '题目音频状态不可用'}
      </span>
      <button
        type="button"
        className="mini-btn"
        disabled={mutation.isPending}
        onClick={() => mutation.mutate()}
      >
        {mutation.isPending ? '生成中…' : '生成本组题目音频'}
      </button>
      {mutation.data ? (
        <span className="muted">
          题目音频：生成 {mutation.data.generated}，跳过 {mutation.data.skipped}，失败 {mutation.data.failed}
        </span>
      ) : null}
      {mutation.error instanceof Error ? (
        <span className="muted" role="alert">{mutation.error.message}</span>
      ) : null}
    </>
  )
}

function MathCard({
  item,
  speaking,
  onPlay,
}: {
  item: MathItem
  speaking: boolean
  onPlay: () => void
}) {
  const hasSpeech = Boolean(item.speechAudioUrl)
  return (
    <article className="char-card math-card">
      {item.glyphImageUrl ? (
        <img
          className="glyph-preview"
          src={glyphImageURL(item.kpId, item.glyphImageUrl)}
          alt={item.title}
        />
      ) : (
        <div className="math-title" title={item.title}>
          {item.title}
        </div>
      )}
      <div className={`sense-tag${item.glyphImageUrl && hasSpeech ? ' yes' : ' no'}`}>
        {item.glyphImageUrl ? '字图已备' : '缺少字图'} · {hasSpeech ? '读音已备' : '缺少读音'}
      </div>
      <div className="char-meta">
        <span className="badge">{kindLabel(item.kind)}</span>
        <span className="badge muted-badge">难度 {item.difficulty}</span>
      </div>
      <div className="card-actions">
        <button
          type="button"
          className="mini-btn"
          disabled={speaking}
          onClick={onPlay}
        >
          {speaking ? '播放中…' : '读音'}
        </button>
      </div>
    </article>
  )
}
