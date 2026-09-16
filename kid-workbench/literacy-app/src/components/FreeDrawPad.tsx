import { useEffect, useRef } from 'react'
import type { InkStroke } from '../lib/handwriting'

export function FreeDrawPad({
  disabled,
  resetToken,
  onChange,
}: {
  disabled?: boolean
  resetToken: number
  onChange: (strokes: InkStroke[]) => void
}) {
  const canvas = useRef<HTMLCanvasElement>(null)
  const strokes = useRef<InkStroke[]>([])
  const drawing = useRef<InkStroke | null>(null)
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange

  useEffect(() => {
    strokes.current = []
    drawing.current = null
    const node = canvas.current
    if (!node) return
    const ctx = node.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, node.width, node.height)
    onChangeRef.current([])
  }, [resetToken])

  useEffect(() => {
    const node = canvas.current
    if (!node) return
    const fit = () => {
      const box = node.getBoundingClientRect()
      const ratio = window.devicePixelRatio || 1
      node.width = Math.max(1, Math.round(box.width * ratio))
      node.height = Math.max(1, Math.round(box.height * ratio))
      const ctx = node.getContext('2d')
      if (!ctx) return
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0)
      ctx.lineCap = 'round'
      ctx.lineJoin = 'round'
      ctx.strokeStyle = '#1f1812'
      ctx.lineWidth = Math.max(10, box.width * 0.028)
      ctx.clearRect(0, 0, box.width, box.height)
      for (const stroke of strokes.current) replay(ctx, stroke)
    }
    fit()
    const observer = new ResizeObserver(fit)
    observer.observe(node)
    return () => observer.disconnect()
  }, [resetToken])

  const point = (event: React.PointerEvent<HTMLCanvasElement>) => {
    const box = event.currentTarget.getBoundingClientRect()
    return { x: event.clientX - box.left, y: event.clientY - box.top }
  }

  const replay = (ctx: CanvasRenderingContext2D, stroke: InkStroke) => {
    if (stroke.length < 2) return
    ctx.beginPath()
    ctx.moveTo(stroke[0].x, stroke[0].y)
    for (let i = 1; i < stroke.length; i += 1) ctx.lineTo(stroke[i].x, stroke[i].y)
    ctx.stroke()
  }

  const start = (event: React.PointerEvent<HTMLCanvasElement>) => {
    if (disabled) return
    event.currentTarget.setPointerCapture(event.pointerId)
    drawing.current = [point(event)]
  }

  const move = (event: React.PointerEvent<HTMLCanvasElement>) => {
    if (!drawing.current || disabled) return
    const next = point(event)
    const last = drawing.current[drawing.current.length - 1]
    if (Math.hypot(next.x - last.x, next.y - last.y) < 1.2) return
    drawing.current.push(next)
    const ctx = event.currentTarget.getContext('2d')
    if (!ctx) return
    ctx.beginPath()
    ctx.moveTo(last.x, last.y)
    ctx.lineTo(next.x, next.y)
    ctx.stroke()
  }

  const end = () => {
    if (!drawing.current) return
    if (drawing.current.length > 1) strokes.current = [...strokes.current, drawing.current]
    drawing.current = null
    onChangeRef.current(strokes.current)
  }

  return (
    <div className="write-wrap">
      <div className="tianzige stage">
        <canvas
          ref={canvas}
          className="write-canvas"
          aria-label="写字板"
          onPointerDown={start}
          onPointerMove={move}
          onPointerUp={end}
          onPointerCancel={end}
        />
      </div>
    </div>
  )
}
