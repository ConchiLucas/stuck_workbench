import {WritingTemplateEditor} from './WritingTemplateEditor'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import {
  listLiteracy,
  patchLiteracyChar,
  speechAudioURL,
} from '../../api/literacy'
import type { LiteracyChar } from '../../api/literacyTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'

async function playSpeech(kpId: number, speechAudioUrl?: string) {
  const res = await fetch(speechAudioURL(kpId, speechAudioUrl))
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

export function LiteracyPage() {
  const [templateKp,setTemplateKp]=useState<number|null>(null)
  const queryClient = useQueryClient()
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [busyKp, setBusyKp] = useState<number | null>(null)
  const [speakingKp, setSpeakingKp] = useState<number | null>(null)
  const [speechError, setSpeechError] = useState('')

  const listQuery = useQuery({
    queryKey: ['literacy', 'chars', 'groups'],
    queryFn: () => listLiteracy({ view: 'groups' }),
  })

  const overrideMutation = useMutation({
    mutationFn: ({ kpId, value }: { kpId: number; value: boolean }) =>
      patchLiteracyChar(kpId, value),
    onMutate: ({ kpId }) => setBusyKp(kpId),
    onSettled: () => {
      setBusyKp(null)
      void queryClient.invalidateQueries({ queryKey: ['literacy'] })
    },
  })

  const onPlaySpeech = async (char: LiteracyChar) => {
    setSpeechError('')
    setSpeakingKp(char.kpId)
    try {
      await playSpeech(char.kpId, char.speechAudioUrl)
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingKp(null)
    }
  }

  const groups = useMemo(() => listQuery.data?.groups ?? [], [listQuery.data])

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (overrideMutation.error instanceof Error && overrideMutation.error.message) ||
    ''

  return (
    <section className="literacy-page" aria-label="识字素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">MATERIALS / LITERACY</p>
          <h1>识字素材</h1>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.literacy} />
        </div>
      </div>

      {templateKp !== null && <section aria-label="书写素材"><WritingTemplateEditor key={templateKp} kpId={templateKp}/><button type="button" className="mini-btn" onClick={() => setTemplateKp(null)}>关闭书写素材</button></section>}
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
                    <span className="muted">{g.chars.length} 字 · {closed ? '展开' : '收起'}</span>
                  </button>
                </div>
                {!closed ? (
                  <div className="char-grid">
                    {g.chars.map((c) => (
                      <CharCard
                        key={c.kpId}
                        char={c}
                        onTemplate={()=>setTemplateKp(c.kpId)}
                        busy={busyKp === c.kpId}
                        speaking={speakingKp === c.kpId}
                        onPlaySpeech={() => void onPlaySpeech(c)}
                        onToggleSense={() =>
                          overrideMutation.mutate({
                            kpId: c.kpId,
                            value: !c.effectiveNeedsSenseImage,
                          })
                        }
                      />
                    ))}
                  </div>
                ) : null}
              </section>
            )
          })}
        </div>
      )}
    </section>
  )
}

function CharCard({
  char,
  busy,
  speaking,
  onPlaySpeech,
  onToggleSense,
  onTemplate,
}: {
  char: LiteracyChar
  busy: boolean
  speaking: boolean
  onPlaySpeech: () => void
  onToggleSense: () => void
  onTemplate:()=>void
}) {
  return (
    <article className={`char-card${char.effectiveNeedsSenseImage ? ' needs-sense' : ''}`}>
      {char.glyphImageUrl ? (
        <img className="glyph-preview" src={char.glyphImageUrl} alt={char.charText} />
      ) : (
        <div className="char-glyph">{char.charText}</div>
      )}
      {char.senseImageUrl ? (
        <img className="sense-preview" src={char.senseImageUrl} alt={`${char.charText}义图`} />
      ) : char.effectiveNeedsSenseImage ? (
        <div className="sense-placeholder">义图未生成</div>
      ) : null}
      <button
        type="button"
        className={`sense-tag${char.effectiveNeedsSenseImage ? ' yes' : ' no'}`}
        disabled={busy}
        title="点击切换要/不要义图"
        onClick={onToggleSense}
      >
        {char.effectiveNeedsSenseImage ? '要义图' : '不要义图'}
        {char.speechAudioUrl ? ' · 有读音' : ''}
      </button>
      <div className="card-actions">
        <button type="button" className="mini-btn" onClick={onTemplate}>书写素材</button>
        <button type="button" className="mini-btn" disabled={speaking} onClick={onPlaySpeech}>
          {speaking ? '…' : char.speechAudioUrl ? '读音✓' : '读音'}
        </button>
      </div>
    </article>
  )
}
