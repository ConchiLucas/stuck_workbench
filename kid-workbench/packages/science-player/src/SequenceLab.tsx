import { useEffect, useRef, useState } from 'react'
import { ScienceGlyph } from './icons'
import type { ScienceExample } from './types'

export function SequenceLab({
  example,
  readOnly = false,
  initialOrder,
  onChange,
  onSolved,
  onMark,
  showFeedback = true,
}: {
  example: ScienceExample
  readOnly?: boolean
  initialOrder?: string[]
  onChange?: (order: string[]) => void
  onSolved?: () => void
  onMark?: (correct: boolean) => void
  showFeedback?: boolean
}) {
  const items = example.sequenceItems ?? []
  const byId = Object.fromEntries(items.map((item) => [item.id, item]))
  const start = initialOrder?.length ? initialOrder : (example.sequenceDisplayOrder?.length ? example.sequenceDisplayOrder : items.map((item) => item.id))
  const [order, setOrder] = useState<string[]>(() => [...start])
  const [dragging, setDragging] = useState<number | null>(null)
  const [dragOver, setDragOver] = useState<number | null>(null)
  const draggingRef = useRef<number | null>(null)

  const displayKey = (example.sequenceDisplayOrder ?? []).join('|')
  useEffect(() => {
    const next = initialOrder?.length ? initialOrder : (example.sequenceDisplayOrder?.length ? example.sequenceDisplayOrder : items.map((item) => item.id))
    setOrder([...next])
  }, [example.prompt, displayKey])

  const emit = (next: string[]) => {
    setOrder(next)
    onChange?.(next)
  }
  const move = (i: number, amount: number) => {
    if (readOnly) return
    const target = i + amount
    if (target < 0 || target >= order.length) return
    const next = [...order]
    ;[next[i], next[target]] = [next[target], next[i]]
    emit(next)
  }
  const reorder = (from: number, to: number) => {
    if (readOnly || from === to || from < 0 || to < 0) return
    const next = [...order]
    const [moved] = next.splice(from, 1)
    next.splice(to, 0, moved)
    emit(next)
  }
  const want = example.correctSequence ?? []
  const solved = want.length > 0 && want.length === order.length && want.every((id, i) => order[i] === id)
  useEffect(() => {
    if (readOnly || !solved) return
    onMark?.(true)
    onSolved?.()
  }, [onMark, onSolved, readOnly, solved])
  const finishDrag = () => { draggingRef.current = null; setDragging(null); setDragOver(null) }
  const startDrag = (i: number) => { if (readOnly) return; draggingRef.current = i; setDragging(i) }
  const dragTo = (from: number, to: number) => {
    if (from === to) return
    reorder(from, to)
    draggingRef.current = to
    setDragging(to)
    setDragOver(to)
  }
  return <>
    <p id="sequence-drag-hint" className="visually-hidden">按住卡片拖到正确位置，也可以用左右方向键调整顺序。</p>
    <div className={`sequence-workbench${readOnly ? ' is-readonly' : ''}`} role="list" aria-label={example.listLabel || '排序'} aria-describedby="sequence-drag-hint" onPointerUp={finishDrag} onPointerCancel={finishDrag} onMouseUp={finishDrag}>
      {order.map((id, i) => {
        const item = byId[id]
        if (!item) return null
        return <div
          key={id}
          role="listitem"
          tabIndex={readOnly ? -1 : 0}
          data-index={i}
          data-id={id}
          aria-label={`拖动排序：${item.label}`}
          aria-grabbed={dragging === i}
          className={`${dragging === i ? 'is-dragging' : ''} ${dragOver === i ? 'is-drag-over' : ''}`}
          onPointerDown={(event) => {
            if (readOnly || (event.pointerType === 'mouse' && event.button !== 0)) return
            startDrag(i)
            event.currentTarget.setPointerCapture?.(event.pointerId)
          }}
          onPointerMove={(event) => {
            const from = draggingRef.current
            if (from === null) return
            const target = document.elementFromPoint(event.clientX, event.clientY)?.closest('[data-index]')
            if (!(target instanceof HTMLElement)) return
            const to = Number(target.dataset.index)
            if (Number.isInteger(to)) dragTo(from, to)
          }}
          onPointerEnter={() => { if (draggingRef.current !== null && draggingRef.current !== i) dragTo(draggingRef.current, i) }}
          onPointerUp={finishDrag}
          onPointerCancel={finishDrag}
          onMouseDown={(event) => { if (!readOnly && event.button === 0) startDrag(i) }}
          onMouseEnter={() => { if (draggingRef.current !== null && draggingRef.current !== i) dragTo(draggingRef.current, i) }}
          onMouseUp={finishDrag}
          onKeyDown={(event) => {
            if (event.key === 'ArrowLeft') { event.preventDefault(); move(i, -1) }
            if (event.key === 'ArrowRight') { event.preventDefault(); move(i, 1) }
          }}
        ><i className="drag-grip" aria-hidden="true" /><span>{item.icon ? <ScienceGlyph name={item.icon} /> : null}</span><b>{i + 1}. {item.label}</b></div>
      })}
    </div>
    {showFeedback && solved && <div className="demo-feedback success" role="status"><b>顺序正确！</b><span>{example.explanation}</span></div>}
  </>
}
