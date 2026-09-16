import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ChengyuPlayer, chengyuTitles, type ChengyuExample, type ChengyuKind } from '@kid-workbench/chengyu-player'
import { listChengyu } from '../../api/chengyu'
import type { ChengyuItem } from '../../api/chengyuTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'
import { appPath } from '../../appPath'
import '@kid-workbench/chengyu-player/player.css'

const DIFF_LABEL: Record<number, string> = {
  1: '启蒙',
  2: '进阶',
  3: '提高',
}

function matchesQuery(item: ChengyuItem, needle: string) {
  if (!needle) return true
  const hay = [
    item.title,
    item.pinyin,
    item.meaning,
    item.example,
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

function firstBlank(full: string, target: string) {
  const runes = Array.from(full)
  const word = Array.from(target)
  let start = -1
  for (let i = 0; i + word.length <= runes.length; i += 1) {
    if (runes.slice(i, i + word.length).join('') === target) {
      start = i
      break
    }
  }
  if (start < 0) return undefined
  return {
    full,
    blanked: runes.slice(0, start).join('') + '____' + runes.slice(start + word.length).join(''),
    target,
    start,
    length: word.length,
  }
}

function examplesFor(item: ChengyuItem, siblings: ChengyuItem[]): ChengyuExample[] {
  const others = siblings.filter((row) => row.kpId !== item.kpId)
  const meaningOptions = [item.meaning, ...item.wrong].filter(Boolean).slice(0, 4).map((label) => ({ id: `label:${label}`, label }))
  const idiomOptions = [item, ...others].slice(0, 4).map((row) => ({ id: String(row.kpId), label: row.title }))
  const speechUrl = item.hasChengyuSpeech ? appPath(`/api/v1/chengyu/items/${item.kpId}/speech.mp3`) : ''
  const blank = firstBlank(item.example, item.title)
  const out: ChengyuExample[] = [
    { kind: 'meaning', stem: '这个成语是什么意思？', speech: item.title, speechUrl, options: meaningOptions, answerId: `label:${item.meaning}`, chengyu: item.title, meaning: item.meaning },
    { kind: 'pick', stem: '看意思，点出这个成语', prompt: item.meaning, meaning: item.meaning, options: idiomOptions, answerId: String(item.kpId), chengyu: item.title },
    { kind: 'pinyin', prompt: item.pinyin, pinyin: item.pinyin, options: idiomOptions, answerId: String(item.kpId), chengyu: item.title },
  ]
  if (blank) {
    out.push({ kind: 'example', prompt: blank.blanked, example: item.example, blank, options: idiomOptions, answerId: String(item.kpId), chengyu: item.title })
  }
  return out
}

export function ChengyuPage() {
  const [q, setQ] = useState('')
  const [moduleFilter, setModuleFilter] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [open, setOpen] = useState<ChengyuItem | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [speaking, setSpeaking] = useState<string | null>(null)

  const listQuery = useQuery({
    queryKey: ['chengyu', 'items', 'groups'],
    queryFn: () => listChengyu({ view: 'groups' }),
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
    <section className="literacy-page" aria-label="成语素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / CHENGYU</p>
          <h1>成语素材</h1>
          <p className="page-description">
            日常、动物、寓言、品德。只读浏览成语、拼音、释义、例句和已有读音。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.chengyu} />
        </div>
      </div>

      <div className="literacy-toolbar">
        <input
          className="search-input"
          placeholder="搜索成语 / 拼音 / 释义"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select aria-label="按分组筛选" value={moduleFilter} onChange={(e) => setModuleFilter(e.target.value)}>
          <option value="">全部分组</option>
          {catalog.map((g) => (
            <option key={g.moduleCode} value={g.moduleCode}>{g.moduleName}</option>
          ))}
        </select>
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 条` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : listQuery.isError ? null : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。</div>
      ) : groups.length === 0 ? (
        <div className="empty-panel">没有匹配的成语。</div>
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
                      {g.items.length} 条 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                </div>
                {!closed ? (
                  <div className="chengyu-grid">
                    {g.items.map((item) => (
                      <ChengyuCard
                        key={item.kpId}
                        item={item}
                        speaking={speaking === `chengyu:${item.kpId}`}
                        onPlay={async () => {
                          if (!item.hasChengyuSpeech) return
                          setSpeechError('')
                          setSpeaking(`chengyu:${item.kpId}`)
                          try {
                            await playURL(appPath(`/api/v1/chengyu/items/${item.kpId}/speech.mp3`))
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
        <Dialog label="成语详情" onClose={() => setOpen(null)}>
          <ChengyuDetail
            item={open}
            siblings={allItems.filter((row) => row.moduleCode === open.moduleCode)}
            speaking={speaking}
            onPlay={async (kind) => {
              const url = kind === 'chengyu' ? open.chengyuSpeechUrl : kind === 'meaning' ? open.meaningSpeechUrl : open.exampleSpeechUrl
              if (!url) return
              setSpeechError('')
              setSpeaking(`${kind}:${open.kpId}`)
              try {
                await playURL(appPath(url))
              } catch (e) {
                setSpeechError(e instanceof Error ? e.message : '读音失败')
              } finally {
                setSpeaking(null)
              }
            }}
          />
        </Dialog>
      ) : null}
    </section>
  )
}

function ChengyuCard({ item, speaking, onPlay, onOpen }: { item: ChengyuItem; speaking: boolean; onPlay: () => void; onOpen: () => void }) {
  return (
    <article className="chengyu-card">
      <h2 className="chengyu-card-title">{item.title}</h2>
      <p className="chengyu-card-pinyin">{item.pinyin}</p>
      <p className="chengyu-card-meaning">{item.meaning}</p>
      {item.example ? <p className="chengyu-card-example">{item.example}</p> : null}
      {item.wrong.length > 0 ? (
        <div className="chengyu-card-wrong">
          {item.wrong.map((w) => (
            <span key={w}>{w}</span>
          ))}
        </div>
      ) : null}
      <p className="chengyu-card-meta">
        {item.moduleName} · {DIFF_LABEL[item.difficulty] ?? `难度 ${item.difficulty}`} · 听释义 · 选成语 · 看拼音 · 看句子
      </p>
      <div className="chengyu-card-actions">
        {item.hasChengyuSpeech ? (
          <button type="button" className="mini-btn" onClick={onPlay} disabled={speaking}>{speaking ? '正在播放…' : '试听成语读音'}</button>
        ) : (
          <span className="muted">成语读音暂不可用</span>
        )}
        <button type="button" className="mini-btn" onClick={onOpen}>查看详情</button>
      </div>
    </article>
  )
}

function ChengyuDetail({ item, siblings, speaking, onPlay }: {
  item: ChengyuItem
  siblings: ChengyuItem[]
  speaking: string | null
  onPlay: (kind: 'chengyu' | 'meaning' | 'example') => void
}) {
  const examples = examplesFor(item, siblings)
  return (
    <div className="chengyu-detail">
      <h2>{item.title}</h2>
      <p>{item.pinyin}</p>
      <p>{item.meaning}</p>
      <p className="muted">只读查看 App 同款题面；试听已有音频，不会改素材，也不会记入孩子学习记录。</p>
      <div className="chengyu-detail-audio">
        <AudioChip label="成语读音" available={Boolean(item.hasChengyuSpeech)} busy={speaking === `chengyu:${item.kpId}`} onPlay={() => onPlay('chengyu')} />
        <AudioChip label="释义音频" available={Boolean(item.hasMeaningSpeech)} busy={speaking === `meaning:${item.kpId}`} onPlay={() => onPlay('meaning')} />
        <AudioChip label="例句音频" available={Boolean(item.hasExampleSpeech)} busy={speaking === `example:${item.kpId}`} onPlay={() => onPlay('example')} />
      </div>
      {examples.map((example) => (
        <article key={example.kind} className="chengyu-detail-example">
          <header>{chengyuTitles[example.kind as ChengyuKind]}</header>
          <div className="chengyu-material-player">
            <ChengyuPlayer example={example} readOnly resolveAssetUrl={(url) => appPath(url)} />
          </div>
        </article>
      ))}
    </div>
  )
}

function AudioChip({ label, available, busy, onPlay }: { label: string; available: boolean; busy: boolean; onPlay: () => void }) {
  if (!available) return <span className="muted">{label}暂不可用</span>
  return <button type="button" className="mini-btn" onClick={onPlay} disabled={busy}>{busy ? `正在播放${label}…` : `试听${label}`}</button>
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
