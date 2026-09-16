import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { ScienceGlyph } from './icons'
import type { ScienceExample } from './types'

export function MatchLab({
  example,
  readOnly = false,
  initialPairs,
  onChange,
  onSolved,
  onMark,
  resolveAssetUrl = (url) => url,
  showFeedback = true,
}: {
  example: ScienceExample
  readOnly?: boolean
  initialPairs?: Record<string, string>
  onChange?: (pairs: Record<string, string>) => void
  onSolved?: () => void
  onMark?: (correct: boolean) => void
  resolveAssetUrl?: (url: string) => string
  showFeedback?: boolean
}) {
  const boardRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef<string | null>(null)
  const dragOriginRef = useRef<{ x: number; y: number } | null>(null)
  const selectedRef = useRef<string | null>(null)
  const ignoreClickRef = useRef(false)
  const [selected, setSelected] = useState<string | null>(null)
  const [pairs, setPairs] = useState<Record<string, string>>(() => ({ ...initialPairs }))
  const [preview, setPreview] = useState<{ x1: number; y1: number; x2: number; y2: number } | null>(null)
  const [boardSize, setBoardSize] = useState({ width: 0, height: 0 })
  const [geom, setGeom] = useState<Array<{ from: string; to: string; x1: number; y1: number; x2: number; y2: number }>>([])
  const sources = example.matchSources ?? []
  const targets = example.matchTargets ?? []
  const answers = example.matchAnswers ?? {}
  const sourceGroup = example.matchSourceGroup ?? '项目'
  const targetGroup = example.matchTargetGroup ?? '选项'
  const [hoverTarget, setHoverTarget] = useState<string | null>(null)
  const [tracking, setTracking] = useState(false)

  useEffect(() => {
    setPairs({ ...(initialPairs ?? {}) })
  }, [example.prompt])

  const selectSource = (id: string | null) => {
    if (readOnly) return
    selectedRef.current = id
    setSelected(id)
  }

  const hitTarget = (clientX: number, clientY: number) => {
    const stack = typeof document.elementsFromPoint === 'function'
      ? document.elementsFromPoint(clientX, clientY)
      : [document.elementFromPoint?.(clientX, clientY)]
    for (const node of stack) {
      if (!(node instanceof Element)) continue
      const target = node.closest('[data-match-role="target"]')
      if (target instanceof HTMLElement && target.dataset.matchNode) return target.dataset.matchNode
    }
    const board = boardRef.current
    if (!board) return null
    const pad = 36
    let best: { id: string; dist: number } | null = null
    for (const node of board.querySelectorAll('[data-match-role="target"]')) {
      if (!(node instanceof HTMLElement) || !node.dataset.matchNode) continue
      const box = node.getBoundingClientRect()
      if (box.width < 8 || box.height < 8) continue
      if (clientX < box.left - pad || clientX > box.right + pad || clientY < box.top - pad || clientY > box.bottom + pad) continue
      const dist = Math.hypot(clientX - (box.left + box.width / 2), clientY - (box.top + box.height / 2))
      if (!best || dist < best.dist) best = { id: node.dataset.matchNode, dist }
    }
    return best?.id ?? null
  }

  const connect = (from: string, to: string) => {
    if (readOnly) return
    setPairs((current) => {
      const next = { ...current }
      for (const [source, target] of Object.entries(next)) {
        if (source === from || target === to) delete next[source]
      }
      next[from] = to
      onChange?.(next)
      return next
    })
    selectSource(null)
    setPreview(null)
    draggingRef.current = null
    dragOriginRef.current = null
  }

  const finishPointer = (event: { clientX: number; clientY: number }) => {
    const from = draggingRef.current
    const origin = dragOriginRef.current
    if (!from && !origin) return
    const sourceId = from
    const moved = origin ? Math.hypot(event.clientX - origin.x, event.clientY - origin.y) >= 8 : false
    const to = hitTarget(event.clientX, event.clientY)
    draggingRef.current = null
    dragOriginRef.current = null
    setHoverTarget(null)
    setPreview(null)
    setTracking(false)
    if (sourceId && to) {
      ignoreClickRef.current = true
      connect(sourceId, to)
      queueMicrotask(() => { ignoreClickRef.current = false })
      return
    }
    if (moved) selectSource(null)
  }
  const finishRef = useRef(finishPointer)
  finishRef.current = finishPointer

  const pointOn = (id: string, edge: 'top' | 'bottom') => {
    const board = boardRef.current
    const node = board?.querySelector(`[data-match-node="${id}"]`)
    if (!board || !(node instanceof HTMLElement)) return null
    const boardBox = board.getBoundingClientRect()
    const port = node.querySelector('.match-port')
    const box = (port instanceof HTMLElement ? port : node).getBoundingClientRect()
    return {
      x: box.left + box.width / 2 - boardBox.left,
      y: port instanceof HTMLElement
        ? box.top + box.height / 2 - boardBox.top
        : (edge === 'bottom' ? box.bottom - boardBox.top : box.top - boardBox.top),
    }
  }

  const refreshGeom = useCallback(() => {
    const board = boardRef.current
    if (board) setBoardSize({ width: board.clientWidth, height: board.clientHeight })
    setGeom(Object.entries(pairs).flatMap(([from, to]) => {
      const start = pointOn(from, 'bottom')
      const end = pointOn(to, 'top')
      if (!start || !end) return []
      return [{ from, to, x1: start.x, y1: start.y, x2: end.x, y2: end.y }]
    }))
  }, [pairs])

  useLayoutEffect(() => { refreshGeom() }, [refreshGeom])
  useEffect(() => {
    const board = boardRef.current
    if (!board) return
    const images = [...board.querySelectorAll('img')]
    const onLoad = () => refreshGeom()
    images.forEach((image) => image.addEventListener('load', onLoad))
    if (typeof ResizeObserver === 'undefined') return () => { images.forEach((image) => image.removeEventListener('load', onLoad)) }
    const observer = new ResizeObserver(() => refreshGeom())
    observer.observe(board)
    window.addEventListener('resize', refreshGeom)
    return () => {
      observer.disconnect()
      window.removeEventListener('resize', refreshGeom)
      images.forEach((image) => image.removeEventListener('load', onLoad))
    }
  }, [refreshGeom])

  useEffect(() => {
    if (!tracking || readOnly) return
    const onMove = (event: PointerEvent) => {
      const from = draggingRef.current
      if (!from) return
      const start = pointOn(from, 'bottom')
      const board = boardRef.current
      if (!start || !board) return
      const box = board.getBoundingClientRect()
      setPreview({ x1: start.x, y1: start.y, x2: event.clientX - box.left, y2: event.clientY - box.top })
      setHoverTarget(hitTarget(event.clientX, event.clientY))
    }
    const onUp = (event: PointerEvent) => finishRef.current(event)
    document.addEventListener('pointermove', onMove)
    document.addEventListener('pointerup', onUp)
    document.addEventListener('pointercancel', onUp)
    return () => {
      document.removeEventListener('pointermove', onMove)
      document.removeEventListener('pointerup', onUp)
      document.removeEventListener('pointercancel', onUp)
    }
  }, [readOnly, tracking])

  useEffect(() => {
    if (readOnly || sources.length === 0) return
    const completeNow = sources.every((item) => pairs[item.id])
    if (!completeNow) return
    const allRightNow = sources.every((item) => pairs[item.id] === answers[item.id])
    onMark?.(allRightNow)
    if (allRightNow) onSolved?.()
  }, [answers, onMark, onSolved, pairs, readOnly, sources])

  const complete = sources.length > 0 && sources.every((item) => pairs[item.id])
  const allRight = complete && sources.every((item) => pairs[item.id] === answers[item.id])
  const boardPoint = (event: { clientX: number; clientY: number }) => {
    const board = boardRef.current
    if (!board) return null
    const box = board.getBoundingClientRect()
    return { x: event.clientX - box.left, y: event.clientY - box.top }
  }
  const lineTone = (from: string, to: string) => answers[from] === to ? 'is-right' : 'is-wrong'
  const nodeFace = (item: { icon?: string; imageUrl?: string; label: string }) => item.imageUrl
    ? <img src={resolveAssetUrl(item.imageUrl)} alt="" />
    : item.icon ? <ScienceGlyph name={item.icon} /> : <span aria-hidden="true" />

  return <>
    <div className={`match-board${readOnly ? ' is-readonly' : ''}`} ref={boardRef}>
      <svg className="match-lines" aria-hidden="true" viewBox={boardSize.width > 0 ? `0 0 ${boardSize.width} ${boardSize.height}` : undefined} preserveAspectRatio="none">
        {geom.map((line) => {
          const tone = lineTone(line.from, line.to)
          return <g key={`${line.from}-${line.to}`} data-testid={`match-line-${line.from}-${line.to}`}>
            <line x1={line.x1} y1={line.y1} x2={line.x2} y2={line.y2} className="match-line-halo" />
            <line x1={line.x1} y1={line.y1} x2={line.x2} y2={line.y2} className={tone} />
            <circle cx={line.x1} cy={line.y1} r="8" className={tone} />
            <circle cx={line.x2} cy={line.y2} r="8" className={tone} />
          </g>
        })}
        {preview && <>
          <line x1={preview.x1} y1={preview.y1} x2={preview.x2} y2={preview.y2} className="match-line-halo" />
          <line x1={preview.x1} y1={preview.y1} x2={preview.x2} y2={preview.y2} className="is-preview" />
        </>}
      </svg>
      <div className="match-sources">
        {sources.map((item) => <button
          key={item.id}
          type="button"
          data-match-node={item.id}
          data-match-role="source"
          aria-label={`选择${sourceGroup}：${item.label}`}
          aria-pressed={selected === item.id}
          disabled={readOnly}
          className={`${selected === item.id ? 'is-selected' : ''} ${pairs[item.id] ? (answers[item.id] === pairs[item.id] ? 'right' : 'picked') : ''}`}
          onPointerDown={(event) => {
            if (readOnly) return
            if (event.pointerType === 'mouse' && event.button !== 0) return
            if (event.currentTarget.hasPointerCapture?.(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
            draggingRef.current = item.id
            dragOriginRef.current = { x: event.clientX, y: event.clientY }
            selectSource(item.id)
            setTracking(true)
            const start = pointOn(item.id, 'bottom')
            const next = boardPoint(event)
            if (start && next) setPreview({ x1: start.x, y1: start.y, x2: next.x, y2: next.y })
          }}
          onPointerMove={(event) => {
            if (draggingRef.current !== item.id) return
            const start = pointOn(item.id, 'bottom')
            const next = boardPoint(event)
            if (start && next) setPreview({ x1: start.x, y1: start.y, x2: next.x, y2: next.y })
            setHoverTarget(hitTarget(event.clientX, event.clientY))
          }}
          onPointerUp={(event) => finishRef.current(event)}
          onPointerCancel={() => {
            draggingRef.current = null
            dragOriginRef.current = null
            setPreview(null)
            setHoverTarget(null)
            setTracking(false)
          }}
          onClick={() => {
            if (readOnly) return
            if (ignoreClickRef.current) {
              ignoreClickRef.current = false
              return
            }
            selectSource(item.id)
          }}
        >{nodeFace(item)}<b>{item.label}</b><i className="match-port" aria-hidden="true" /></button>)}
      </div>
      <div className="match-targets">
        {targets.map((item) => <button
          key={item.id}
          type="button"
          data-match-node={item.id}
          data-match-role="target"
          aria-label={`选择${targetGroup}：${item.label}`}
          disabled={readOnly}
          className={`${Object.values(pairs).includes(item.id) ? (Object.entries(pairs).some(([from, to]) => to === item.id && answers[from] === to) ? 'right' : 'picked') : ''} ${hoverTarget === item.id ? 'is-drop' : ''}`}
          onPointerUp={(event) => {
            if (readOnly) return
            const from = draggingRef.current ?? selectedRef.current
            if (!from) return
            event.stopPropagation()
            ignoreClickRef.current = true
            connect(from, item.id)
            queueMicrotask(() => { ignoreClickRef.current = false })
            draggingRef.current = null
            dragOriginRef.current = null
            setPreview(null)
            setHoverTarget(null)
            setTracking(false)
          }}
          onClick={() => {
            if (readOnly) return
            const from = selectedRef.current
            if (from) connect(from, item.id)
          }}
        ><i className="match-port" aria-hidden="true" />{nodeFace(item)}<b>{item.label}</b></button>)}
      </div>
    </div>
    {showFeedback && complete && <div className={`demo-feedback ${allRight ? 'success' : 'again'}`} role="status"><b>{allRight ? '连对了！' : '再连一连'}</b><span>{allRight ? example.explanation : example.tip}</span></div>}
  </>
}
