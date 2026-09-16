import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { LogicPlayer, logicTitles, type LogicExample, type LogicKind } from '@kid-workbench/logic-player'
import { listLogic } from '../../api/logic'
import type { LogicItem } from '../../api/logicTypes'
import { KidAppLink } from '../../components/KidAppLink'
import { KID_APP_PORTS } from '../../content/kidApps'
import { appPath } from '../../appPath'
import '@kid-workbench/logic-player/player.css'
import '../science/science.css'

const KIND_LABEL: Record<string, string> = logicTitles
const KINDS: LogicKind[] = ['pattern', 'classify', 'order', 'shape_reason', 'diff', 'compare']

function matchesQuery(item: LogicItem, needle: string) {
  if (!needle) return true
  const hay = [
    item.title, item.prompt, item.answer, item.kind, item.moduleName, item.rule,
    ...item.seq, ...item.wrong,
  ].join(' ').toLowerCase()
  return hay.includes(needle)
}

export function LogicPage() {
  const [q, setQ] = useState('')
  const [kindFilter, setKindFilter] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [open, setOpen] = useState<LogicItem | null>(null)

  const listQuery = useQuery({
    queryKey: ['logic', 'items', 'groups'],
    queryFn: () => listLogic({ view: 'groups' }),
  })

  const needle = q.trim().toLowerCase()
  const error = (listQuery.error instanceof Error && listQuery.error.message) || ''
  const groups = useMemo(() => (listQuery.data?.groups ?? [])
    .map((g) => ({
      ...g,
      items: g.items.filter((item) => (!kindFilter || item.kind === kindFilter) && matchesQuery(item, needle)),
    }))
    .filter((g) => g.items.length > 0), [listQuery.data, kindFilter, needle])

  return (
    <section className="literacy-page" aria-label="逻辑素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">MATERIALS / LOGIC</p>
          <h1>逻辑素材</h1>
          <p className="page-description">
            找规律、分类、排序、图形推理、找不同、比较。只读浏览规则、图形和题面。
          </p>
        </div>
        <div className="heading-actions">
          <KidAppLink port={KID_APP_PORTS.logic} />
        </div>
      </div>

      <div className="literacy-toolbar">
        <input
          className="search-input"
          placeholder="搜索题目 / 序列 / 答案"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select aria-label="按题型筛选" value={kindFilter} onChange={(e) => setKindFilter(e.target.value)}>
          <option value="">全部题型</option>
          {KINDS.map((kind) => <option key={kind} value={kind}>{KIND_LABEL[kind]}</option>)}
        </select>
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 题` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。</div>
      ) : groups.length === 0 ? (
        <div className="empty-panel">没有匹配的逻辑素材。</div>
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
                    <span className="muted">{g.items.length} 题 · {closed ? '展开' : '收起'}</span>
                  </button>
                </div>
                {!closed ? (
                  <div className="logic-grid">
                    {g.items.map((item) => <LogicCard key={item.kpId} item={item} onOpen={() => setOpen(item)} />)}
                  </div>
                ) : null}
              </section>
            )
          })}
        </div>
      )}
      {open ? (
        <Dialog label="逻辑详情" onClose={() => setOpen(null)}>
          <LogicDetail item={open} />
        </Dialog>
      ) : null}
    </section>
  )
}

function LogicCard({ item, onOpen }: { item: LogicItem; onOpen: () => void }) {
  const preview = item.example?.objects[0]
  const previewURL = preview && item.glyphUrls?.[preview.id]
  return (
    <article className="logic-card">
      <h2 className="logic-card-title">{item.title}</h2>
      <p className="logic-card-prompt">{item.prompt}</p>
      {item.seq.length > 0 ? <p className="logic-card-seq">{item.seq.join(' ')}</p> : null}
      {previewURL ? <img className="sense-preview" src={appPath(previewURL)} alt="" /> : null}
      <div className="logic-card-choices">
        <span className="logic-answer">答案 {item.answer}</span>
        {item.wrong.map((w) => <span key={w} className="logic-wrong">{w}</span>)}
      </div>
      <p className="logic-card-meta">
        {item.moduleName} · {KIND_LABEL[item.kind] ?? item.kind}
      </p>
      <div className="phrase-card-actions">
        <button type="button" className="mini-btn" onClick={onOpen}>查看详情</button>
      </div>
    </article>
  )
}

function LogicDetail({ item }: { item: LogicItem }) {
  const example = item.example as LogicExample | undefined
  return (
    <div className="phrase-detail">
      <h2>{item.title}</h2>
      <p>{item.prompt}</p>
      {item.rule ? <p>{item.rule}</p> : null}
      <p className="muted">只读查看 App 同款题面。浏览不会改素材，也不会记入孩子学习记录。</p>
      {example ? (
        <article className="phrase-detail-example">
          <header>{KIND_LABEL[example.kind] ?? example.kind}</header>
          <div className="science-material-player">
            <LogicPlayer example={example} readOnly resolveAssetUrl={(url) => appPath(url)} />
          </div>
        </article>
      ) : (
        <p className="muted">这条早期素材没有可还原的结构化题面。</p>
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
