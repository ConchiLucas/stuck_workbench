import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { PhrasePlayer, phraseTitles, type PhraseExample, type PhraseKind } from '@kid-workbench/phrase-player'
import { listPhrase } from '../../api/phrase'
import type { PhraseItem } from '../../api/phraseTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'
import { appPath } from '../../appPath'
import '@kid-workbench/phrase-player/player.css'

const DIFF_LABEL: Record<number, string> = {
  1: '启蒙',
  2: '进阶',
  3: '提高',
}

function matchesQuery(item: PhraseItem, needle: string) {
  if (!needle) return true
  const hay = [
    item.title,
    item.zh,
    item.scene,
    item.replyTo,
    item.moduleName,
    ...item.wrong,
  ].join(' ').toLowerCase()
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

function examplesFor(item: PhraseItem, siblings: PhraseItem[]): PhraseExample[] {
  const others = siblings.filter((row) => row.kpId !== item.kpId)
  const zhByText = new Map(siblings.map((row) => [row.zh, row]))
  const zhOptions = [item.zh, ...item.wrong].filter(Boolean).slice(0, 4).map((label) => {
    const match = zhByText.get(label)
    return { id: match ? String(match.kpId) : `label:${label}`, label }
  })
  const enPool = [item, ...others].slice(0, 4)
  const enOptions = enPool.map((row) => ({ id: String(row.kpId), label: row.title }))
  const speechUrl = item.hasSpeech ? appPath(`/api/v1/phrase/items/${item.kpId}/speech.mp3`) : ''
  const question = siblings.find((row) => row.title === item.replyTo)
  const replySpeech = question?.hasSpeech ? appPath(`/api/v1/phrase/items/${question.kpId}/speech.mp3`) : ''
  const out: PhraseExample[] = [
    { kind: 'listen_zh', stem: '听一听，选出中文意思', speech: item.title, speechUrl, options: zhOptions, answerId: String(item.kpId) },
    { kind: 'listen_en', stem: '听一听，点出这句英语', speech: item.title, speechUrl, options: enOptions, answerId: String(item.kpId) },
  ]
  if (item.scene) {
    out.push({ kind: 'scene', stem: '这种时候该说哪一句？', prompt: item.scene, scene: item.scene, options: enOptions, answerId: String(item.kpId) })
  }
  if (item.replyTo) {
    out.push({ kind: 'reply', stem: '对方说了这句话，你怎么答？', prompt: item.replyTo, replyTo: item.replyTo, speech: item.replyTo, speechUrl: replySpeech, options: enOptions, answerId: String(item.kpId) })
  }
  return out
}

export function PhrasePage() {
  const [q, setQ] = useState('')
  const [moduleFilter, setModuleFilter] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [open, setOpen] = useState<PhraseItem | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [speaking, setSpeaking] = useState<number | null>(null)

  const listQuery = useQuery({
    queryKey: ['phrase', 'items', 'groups'],
    queryFn: () => listPhrase({ view: 'groups' }),
  })

  const needle = q.trim().toLowerCase()
  const groups = useMemo(() => (listQuery.data?.groups ?? [])
    .filter((g) => !moduleFilter || g.moduleCode === moduleFilter)
    .map((g) => ({ ...g, items: g.items.filter((item) => matchesQuery(item, needle)) }))
    .filter((g) => g.items.length > 0), [listQuery.data, moduleFilter, needle])
  const catalog = listQuery.data?.groups ?? []
  const allItems = catalog.flatMap((g) => g.items)
  const error = speechError || (listQuery.error instanceof Error && listQuery.error.message) || ''

  return (
    <section className="literacy-page" aria-label="英语短句素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / PHRASE</p>
          <h1>英语短句素材</h1>
          <p className="page-description">
            问候、课堂、日常、感受。只读浏览短句、中文含义、场景、问答配对和已有整句读音。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.phrase} />
        </div>
      </div>

      <div className="literacy-toolbar">
        <input
          className="search-input"
          placeholder="搜索短句 / 中文 / 场景"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select aria-label="按分组筛选" value={moduleFilter} onChange={(e) => setModuleFilter(e.target.value)}>
          <option value="">全部分组</option>
          {catalog.map((g) => (
            <option key={g.moduleCode} value={g.moduleCode}>{g.moduleName}</option>
          ))}
        </select>
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 句` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : listQuery.isError ? null : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。</div>
      ) : groups.length === 0 ? (
        <div className="empty-panel">没有匹配的短句。</div>
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
                      {g.items.length} 句 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                </div>
                {!closed ? (
                  <div className="phrase-grid">
                    {g.items.map((item) => (
                      <PhraseCard
                        key={item.kpId}
                        item={item}
                        speaking={speaking === item.kpId}
                        onPlay={async () => {
                          if (!item.hasSpeech) return
                          setSpeechError('')
                          setSpeaking(item.kpId)
                          try {
                            await playURL(appPath(`/api/v1/phrase/items/${item.kpId}/speech.mp3`))
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
        <Dialog label="短句详情" onClose={() => setOpen(null)}>
          <PhraseDetail item={open} siblings={allItems.filter((row) => row.moduleCode === open.moduleCode)} />
        </Dialog>
      ) : null}
    </section>
  )
}

function PhraseCard({ item, speaking, onPlay, onOpen }: { item: PhraseItem; speaking: boolean; onPlay: () => void; onOpen: () => void }) {
  return (
    <article className="phrase-card">
      <h2 className="phrase-card-title">{item.title}</h2>
      <p className="phrase-card-zh">{item.zh}</p>
      {item.scene ? <p className="phrase-card-scene">{item.scene}</p> : null}
      {item.replyTo ? <p className="phrase-card-reply">应答 {item.replyTo}</p> : null}
      {item.wrong.length > 0 ? (
        <div className="phrase-card-wrong">
          {item.wrong.map((w) => (
            <span key={w}>{w}</span>
          ))}
        </div>
      ) : null}
      <p className="phrase-card-meta">
        {item.moduleName} · {DIFF_LABEL[item.difficulty] ?? `难度 ${item.difficulty}`} · 听一听 · 选句子 · 什么时候说
        {item.replyTo ? ' · 问与答' : ''}
      </p>
      <div className="phrase-card-actions">
        {item.hasSpeech ? (
          <button type="button" className="mini-btn" onClick={onPlay} disabled={speaking}>{speaking ? '正在播放…' : '试听整句'}</button>
        ) : (
          <span className="muted">整句读音暂不可用</span>
        )}
        <button type="button" className="mini-btn" onClick={onOpen}>查看详情</button>
      </div>
    </article>
  )
}

function PhraseDetail({ item, siblings }: { item: PhraseItem; siblings: PhraseItem[] }) {
  const examples = examplesFor(item, siblings)
  return (
    <div className="phrase-detail">
      <h2>{item.title}</h2>
      <p>{item.zh}</p>
      <p className="muted">只读查看 App 同款题面；试听已有整句，不会改素材，也不会记入孩子学习记录。</p>
      {examples.map((example) => (
        <article key={example.kind} className="phrase-detail-example">
          <header>{phraseTitles[example.kind as PhraseKind]}</header>
          <div className="phrase-material-player">
            <PhrasePlayer example={example} readOnly resolveAssetUrl={(url) => appPath(url)} />
          </div>
        </article>
      ))}
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
