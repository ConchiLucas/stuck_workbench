import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { LogicPlayer, logicTitles, type LogicAnswer, type LogicExample, type LogicKind } from '@kid-workbench/logic-player'
import { genAPI } from '../api/generation'
import { appPath } from '../appPath'
import '@kid-workbench/logic-player/player.css'
import './scienceTasks.css'

type Item = {
  id: string
  kind: LogicKind
  skillCode: string
  targetId: number
  sourceId?: number
  sourceTable: string
  moduleCode: string
  moduleName?: string
  sourceContentHash?: string
  example: LogicExample
  mediaSHA256: Record<string, string>
}
type Task = { id: number; title: string; count: number; types: LogicKind[]; groups?: string[]; createdAt: string; items?: Item[] }
const types: Record<LogicKind, string> = logicTitles
const endpoint = 'logic/question-tasks'
export function LogicTasksPage() {
  const qc = useQueryClient()
  const [openId, setOpenId] = useState<number | null>(null)
  const [creating, setCreating] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [title, setTitle] = useState('')
  const [count, setCount] = useState(12)
  const [selected, setSelected] = useState<LogicKind[]>(Object.keys(types) as LogicKind[])
  const submitting = useRef(false)
  const list = useQuery({ queryKey: ['logic-tasks'], queryFn: () => genAPI<{ items: Task[] }>(endpoint) })
  const detail = useQuery({ queryKey: ['logic-task', openId], queryFn: () => genAPI<Task>(`${endpoint}/${openId}`), enabled: openId !== null })
  async function generate() {
    if (submitting.current) return
    submitting.current = true
    setBusy(true)
    setError('')
    try {
      const task = await genAPI<Task>(endpoint, 'POST', { title, count, types: selected })
      setCreating(false)
      setOpenId(task.id)
      await qc.invalidateQueries({ queryKey: ['logic-tasks'] })
    } catch (e) {
      setError((e as Error).message)
    } finally {
      submitting.current = false
      setBusy(false)
    }
  }
  return <>
    <section className="gen-page" aria-label="逻辑出题任务列表">
      <div className="science-task-actions"><button className="mini-btn" onClick={() => { setError(''); setCreating(true) }}>生成逻辑题目</button></div>
      {list.isLoading && <p>正在读取任务…</p>}
      {list.error && <p role="alert" className="error-panel">{list.error.message}</p>}
      {!list.isLoading && !list.error && !list.data?.items.length && <div className="empty-panel">暂无逻辑出题任务。</div>}
      <div className="gen-task-list">{list.data?.items.map(task => <button className="gen-task-row science-task-row" key={task.id} onClick={() => setOpenId(task.id)}><span className="gen-task-kind">练习</span><div><strong>{task.title}</strong><small>{task.types.map(t => types[t]).join(' / ')} · {task.groups?.join(' / ')} · {task.count} 题 · {new Date(task.createdAt).toLocaleString('zh-CN')}</small></div><span>查看 →</span></button>)}</div>
    </section>
    {creating && <Dialog onClose={() => { if (!busy) setCreating(false) }} label="生成逻辑题目">
      <form className="gen-form science-task-form" onSubmit={event => { event.preventDefault(); void generate() }}>
        <label>任务名称<input maxLength={80} value={title} placeholder="逻辑练习" onChange={e => setTitle(e.target.value)} disabled={busy}/></label>
        <fieldset disabled={busy}><legend>题型</legend>{(Object.keys(types) as LogicKind[]).map(t => <label key={t}><input type="checkbox" checked={selected.includes(t)} onChange={() => setSelected(prev => prev.includes(t) ? prev.filter(x => x !== t) : [...prev, t])}/>{types[t]}</label>)}</fieldset>
        <label>题数<input type="number" min={Math.max(1, selected.length)} max={40} value={count} onChange={e => setCount(Number(e.target.value))} disabled={busy}/></label>
        <p className="muted">使用素材后台已有逻辑对象、规则和图形生成找规律、分类、排序、图形推理、找不同和比较题。生成后保存稳定编号、展示顺序、规则、答案和媒体。缺少素材时会说明原因。试答不会记入孩子学习记录。</p>
        {error && <p role="alert" className="error-panel">{error}</p>}
        <button className="mini-btn" disabled={busy || !selected.length || count < selected.length || count > 40 || !Number.isInteger(count)}>{busy ? '正在生成并保存…' : '生成并保存'}</button>
      </form>
    </Dialog>}
    {openId !== null && <Dialog onClose={() => setOpenId(null)} label="题目查看">
      {detail.isLoading ? <p>正在读取题目…</p> : detail.error ? <p role="alert" className="error-panel">{detail.error.message}</p> : !detail.data?.items?.length ? <p className="empty-panel">暂无可查看的题目。</p> : <div className="gen-question-list">{detail.data.items.map((q, i) => <article key={`${openId}:${q.id}`} className="gen-question-card"><header><b>第 {i + 1} 题 · {types[q.kind]}</b></header><Preview example={q.example}/><details><summary>素材与出题来源</summary><p>逻辑素材 #{q.sourceId || q.targetId} · 分组 {q.moduleName || q.moduleCode}{q.sourceContentHash ? ` · 内容摘要 ${q.sourceContentHash}` : ''}</p><p>题目、规则、选项、顺序和媒体按生成时保存；试答不会记入孩子学习记录。</p><ul>{Object.entries(q.mediaSHA256 || {}).map(([path, hash]) => <li className="science-media-source" key={path}>{path}<small>媒体摘要 {hash}</small></li>)}</ul></details></article>)}</div>}
    </Dialog>}
  </>
}
function Preview({ example }: { example: LogicExample }) {
  const [key, setKey] = useState(0)
  const [answer, setAnswer] = useState<LogicAnswer | undefined>()
  const done = Boolean(answer && (example.kind === 'order' ? (answer.sequence ?? []).length : answer.selectedId))
  return <div className="science-task-player">
    <LogicPlayer key={key} example={example} onAnswer={setAnswer} resolveAssetUrl={url => appPath(url)} />
    {done ? <button type="button" className="mini-btn" onClick={() => { setAnswer(undefined); setKey(value => value + 1) }}>重来</button> : null}
  </div>
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
  return <dialog ref={ref} className="task-preview-dialog" aria-label={label} onCancel={event => { event.preventDefault(); onClose() }}><div className="task-preview-bar"><button className="task-preview-close" aria-label="关闭弹窗" onClick={onClose}>×</button></div><div className="task-preview-content">{children}</div></dialog>
}
