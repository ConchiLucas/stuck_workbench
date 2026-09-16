import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { listPoem } from '../../api/poem'
import type { PoemItem } from '../../api/poemTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'
import { appPath } from '../../appPath'
import './poem.css'

const DIFF_LABEL: Record<number, string> = {
  1: '启蒙',
  2: '进阶',
  3: '提高',
}

function matchesQuery(item: PoemItem, needle: string) {
  if (!needle) return true
  const hay = [item.title, item.author, item.dynasty, item.workId, item.line1, item.line2, ...item.lines].join(' ').toLowerCase()
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

export function PoemPage() {
  const [q, setQ] = useState('')
  const [dynasty, setDynasty] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [open, setOpen] = useState<PoemItem | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [speaking, setSpeaking] = useState<string | null>(null)

  const listQuery = useQuery({
    queryKey: ['poem', 'items', 'groups'],
    queryFn: () => listPoem({ view: 'groups' }),
  })

  const needle = q.trim().toLowerCase()
  const dynasties = useMemo(() => {
    const set = new Set<string>()
    for (const group of listQuery.data?.groups ?? []) {
      for (const item of group.items) {
        if (item.dynasty) set.add(item.dynasty)
      }
    }
    return [...set]
  }, [listQuery.data])
  const groups = useMemo(() => (listQuery.data?.groups ?? [])
    .map((g) => ({
      ...g,
      items: g.items.filter((item) => (!dynasty || item.dynasty === dynasty) && matchesQuery(item, needle)),
    }))
    .filter((g) => g.items.length > 0), [listQuery.data, dynasty, needle])
  const error = speechError || (listQuery.error instanceof Error && listQuery.error.message) || ''

  return (
    <section className="literacy-page" aria-label="古诗素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">MATERIALS / POEM</p>
          <h1>古诗素材</h1>
          <p className="page-description">
            只读浏览作品身份、题名、作者、朝代、正文行序、版本和已有读音。缺素材通过开发流程补齐。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.poem} />
        </div>
      </div>

      <div className="literacy-toolbar">
        <input
          className="search-input"
          placeholder="搜索诗名 / 作者 / 诗句"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select aria-label="按朝代筛选" value={dynasty} onChange={(e) => setDynasty(e.target.value)}>
          <option value="">全部朝代</option>
          {dynasties.map((name) => <option key={name} value={name}>{name}</option>)}
        </select>
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 首` : ''}</span>
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
                    onClick={() => setCollapsed((prev) => ({ ...prev, [g.moduleCode]: !prev[g.moduleCode] }))}
                  >
                    <span>{g.moduleName}</span>
                    <span className="muted">{g.items.length} 首 · {closed ? '展开' : '收起'}</span>
                  </button>
                </div>
                {!closed ? (
                  <div className="poem-grid">
                    {g.items.map((item) => (
                      <PoemCard key={item.kpId} item={item} onOpen={() => { setSpeechError(''); setOpen(item) }} />
                    ))}
                  </div>
                ) : null}
              </section>
            )
          })}
        </div>
      )}

      {open ? (
        <div className="science-detail-overlay" role="dialog" aria-modal="true" aria-label={`${open.title}素材详情`}>
          <div className="science-detail">
            <header>
              <div>
                <p className="eyebrow">{open.workId || open.code} · {open.dynasty || '朝代未记'}〔{open.author}〕</p>
                <h2>{open.title}</h2>
              </div>
              <button type="button" className="mini-btn" onClick={() => setOpen(null)}>关闭</button>
            </header>
            <p className="muted">版本 {open.edition || '通行小学课文'}。诗行身份按 {open.workId || open.code}:L 行序保存。</p>
            <ol className="poem-line-list">
              {(open.lineItems?.length ? open.lineItems.map((line) => line.text) : open.lines).map((line, index) => {
                const ord = open.lineItems?.[index]?.ord || index + 1
                const speechUrl = appPath(`/api/v1/poem/items/${open.kpId}/speech/${ord}.wav`)
                return (
                  <li key={`${open.kpId}:${ord}`}>
                    <span>{line}</span>
                    <button
                      type="button"
                      className="mini-btn"
                      disabled={speaking === speechUrl}
                      onClick={() => {
                        setSpeechError('')
                        setSpeaking(speechUrl)
                        void playURL(speechUrl).catch((e) => setSpeechError((e as Error).message)).finally(() => setSpeaking(null))
                      }}
                    >{speaking === speechUrl ? '试听中…' : '试听'}</button>
                  </li>
                )
              })}
            </ol>
          </div>
        </div>
      ) : null}
    </section>
  )
}

function PoemCard({ item, onOpen }: { item: PoemItem; onOpen: () => void }) {
  return (
    <article className="poem-card">
      <button type="button" className="poem-card-open" onClick={onOpen}>
        <h2 className="poem-card-title">{item.title}</h2>
        <p className="poem-card-author">{item.dynasty ? `${item.dynasty}〔${item.author}〕` : `〔${item.author}〕`}</p>
        <div className="poem-card-lines">
          {item.lines.map((line, index) => (
            <p key={`${item.kpId}:${index}`}>{line}</p>
          ))}
        </div>
        <p className="poem-card-meta">
          {DIFF_LABEL[item.difficulty] ?? `难度 ${item.difficulty}`} · 选诗名 · 补字 · 选下一句 · 排顺序 · 查看详情
        </p>
      </button>
    </article>
  )
}
