import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { SciencePlayer, scienceTitles, type ScienceExample, type ScienceKind } from '@kid-workbench/science-player'
import { listScience, speechAudioURL } from '../../api/science'
import type { ScienceItem } from '../../api/scienceTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'
import { appPath } from '../../appPath'
import '@kid-workbench/science-player/player.css'
import './science.css'

const KIND_LABEL: Record<string, string> = scienceTitles

function matchesQuery(item: ScienceItem, needle: string) {
  if (!needle) return true
  const hay = [item.title, item.prompt, item.kind, item.moduleName, item.summary, item.explanation].join(' ').toLowerCase()
  return hay.includes(needle)
}

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

export function SciencePage() {
  const [q, setQ] = useState('')
  const [moduleFilter, setModuleFilter] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [open, setOpen] = useState<ScienceItem | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [speaking, setSpeaking] = useState<number | null>(null)

  const listQuery = useQuery({
    queryKey: ['science', 'items', 'groups'],
    queryFn: () => listScience({ view: 'groups' }),
  })

  const needle = q.trim().toLowerCase()
  const groups = useMemo(() => (listQuery.data?.groups ?? [])
    .filter((g) => !moduleFilter || g.moduleCode === moduleFilter)
    .map((g) => ({ ...g, items: g.items.filter((item) => matchesQuery(item, needle)) }))
    .filter((g) => g.items.length > 0), [listQuery.data, moduleFilter, needle])
  const catalog = listQuery.data?.groups ?? []
  const error = speechError || (listQuery.error instanceof Error && listQuery.error.message) || ''

  return (
    <section className="literacy-page" aria-label="科普素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / SCIENCE</p>
          <h1>科普素材</h1>
          <p className="page-description">
            观察、过程、结构。只读浏览知识点、结构图、环境图和已有读音。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.science} />
        </div>
      </div>

      <div className="literacy-toolbar">
        <input
          className="search-input"
          placeholder="搜索知识点 / 题干 / 题型"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select aria-label="按分组筛选" value={moduleFilter} onChange={(e) => setModuleFilter(e.target.value)}>
          <option value="">全部分组</option>
          {catalog.map((g) => (
            <option key={g.moduleCode} value={g.moduleCode}>{g.moduleName}</option>
          ))}
        </select>
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 项` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : listQuery.isError ? null : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。</div>
      ) : groups.length === 0 ? (
        <div className="empty-panel">没有匹配的科普素材。</div>
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
                    onClick={() => setCollapsed((prev) => ({ ...prev, [g.moduleCode]: !prev[g.moduleCode] }))}
                  >
                    <span>{g.moduleName}</span>
                    <span className="muted">
                      {g.items.length} 项 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                </div>
                {!closed ? (
                  <div className="phrase-grid">
                    {g.items.map((item) => (
                      <ScienceCard
                        key={item.kpId}
                        item={item}
                        speaking={speaking === item.kpId}
                        onPlay={async () => {
                          if (!item.hasSpeech && !item.speechAudioUrl) return
                          setSpeechError('')
                          setSpeaking(item.kpId)
                          try {
                            await playURL(speechAudioURL(item.kpId, item.speechAudioUrl))
                          } catch (e) {
                            setSpeechError(e instanceof Error ? e.message : '读音失败')
                          } finally {
                            setSpeaking(null)
                          }
                        }}
                        onOpen={() => setOpen(item)}
                      />
                    ))}
                  </div>
                ) : null}
              </section>
            )
          })}
        </div>
      )}
      {open ? (
        <Dialog label="科普详情" onClose={() => setOpen(null)}>
          <ScienceDetail item={open} />
        </Dialog>
      ) : null}
    </section>
  )
}

function ScienceCard({ item, speaking, onPlay, onOpen }: { item: ScienceItem; speaking: boolean; onPlay: () => void; onOpen: () => void }) {
  const kindLabel = item.kind ? (KIND_LABEL[item.kind as ScienceKind] ?? item.kind) : ''
  const image = item.diagramUrl || item.glyphImageUrl || item.senseImageUrl
  return (
    <article className="phrase-card">
      <h2 className="phrase-card-title">{item.title}</h2>
      {item.prompt ? <p className="phrase-card-zh">{item.prompt}</p> : item.summary ? <p className="phrase-card-zh">{item.summary}</p> : null}
      {image ? <img className="sense-preview" src={appPath(item.diagramUrl || item.glyphImageUrl || item.senseImageUrl)} alt={item.title} /> : null}
      <p className="phrase-card-meta">
        {item.moduleName}{kindLabel ? ` · ${kindLabel}` : ''}
      </p>
      <div className="phrase-card-actions">
        {item.hasSpeech || item.speechAudioUrl ? (
          <button type="button" className="mini-btn" onClick={onPlay} disabled={speaking}>{speaking ? '正在播放…' : '试听读音'}</button>
        ) : (
          <span className="muted">读音暂不可用</span>
        )}
        <button type="button" className="mini-btn" onClick={onOpen}>查看详情</button>
      </div>
    </article>
  )
}

function ScienceDetail({ item }: { item: ScienceItem }) {
  const example = item.example as ScienceExample | undefined
  return (
    <div className="phrase-detail">
      <h2>{item.title}</h2>
      {item.prompt ? <p>{item.prompt}</p> : null}
      {item.explanation ? <p>{item.explanation}</p> : null}
      {item.rationale ? <p className="muted">{item.rationale}</p> : null}
      {item.citation ? <p className="muted">依据：{item.citation}</p> : null}
      <p className="muted">只读查看 App 同款题面；试听已有读音，不会改素材，也不会记入孩子学习记录。</p>
      {item.hasDiagram && item.diagramUrl ? <img className="sense-preview" src={appPath(item.diagramUrl)} alt={`${item.title}结构图`} /> : null}
      {example ? (
        <article className="phrase-detail-example">
          <header>{KIND_LABEL[example.kind] ?? example.kind}</header>
          <div className="science-material-player">
            <SciencePlayer example={example} readOnly resolveAssetUrl={(url) => appPath(url)} />
          </div>
        </article>
      ) : (
        <p className="muted">这条素材没有可还原的结构化题面。</p>
      )}
    </div>
  )
}

function Dialog({ onClose, label, children }: { onClose: () => void; label: string; children: ReactNode }) {
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
    <dialog ref={ref} className="task-preview-dialog" aria-label={label} onCancel={(event) => { event.preventDefault(); onClose() }}>
      <div className="task-preview-bar">
        <button type="button" className="task-preview-close" aria-label="关闭弹窗" onClick={onClose}>×</button>
      </div>
      <div className="task-preview-content">{children}</div>
    </dialog>
  )
}
