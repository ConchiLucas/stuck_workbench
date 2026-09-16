import { useState } from 'react'
import { PlantDiagram } from './icons'
import type { ScienceExample } from './types'

export function LabelLab({
  example,
  readOnly = false,
  initialLabels,
  onChange,
  onSolved,
  onMark,
  showFeedback = true,
}: {
  example: ScienceExample
  readOnly?: boolean
  initialLabels?: Record<string, string>
  onChange?: (labels: Record<string, string>) => void
  onSolved?: () => void
  onMark?: (correct: boolean) => void
  showFeedback?: boolean
}) {
  const targets = example.labelTargets ?? []
  const token = example.labelTokens?.[0]
  const answers = example.labelAnswers ?? {}
  const [labels, setLabels] = useState<Record<string, string>>(() => ({ ...initialLabels }))
  const [dragging, setDragging] = useState(false)

  const place = (targetId: string) => {
    if (readOnly || !token) return
    const next = { [targetId]: token.id }
    setLabels(next)
    onChange?.(next)
    const right = answers[targetId] === token.id && Object.keys(answers).every((id) => (id === targetId ? token.id : next[id]) === answers[id]) && Object.keys(next).length === Object.keys(answers).length
    onMark?.(right)
    if (right) onSolved?.()
  }

  const placed = Object.keys(labels)[0]
  const right = Boolean(placed) && answers[placed] === labels[placed] && Object.keys(answers).length === Object.keys(labels).length
  return <>
    <div className={`label-workbench${readOnly ? ' is-readonly' : ''}`}>
      <div className="plant-diagram" data-diagram={`${example.diagramKey ?? 'plant-structure'}@${example.diagramVersion ?? 1}`}>
        <PlantDiagram />
        {targets.map((target) => {
          const here = labels[target.id]
          return <button
            key={target.id}
            type="button"
            className={`hotspot hotspot-${target.id}${here ? (answers[target.id] === here ? ' right' : ' picked') : ''}${dragging ? ' is-drop' : ''}`}
            style={{ left: `${target.x * 100}%`, top: `${target.y * 100}%` }}
            aria-label={`标注植物的${target.label}`}
            disabled={readOnly}
            onClick={() => place(target.id)}
            onPointerUp={(event) => {
              if (!dragging) return
              event.preventDefault()
              place(target.id)
              setDragging(false)
            }}
          >{here ? (example.labelTokens?.find((item) => item.id === here)?.label ?? '✓') : '＋'}</button>
        })}
      </div>
      {token ? <div
        className={`label-token${readOnly ? ' is-readonly' : ''}`}
        onPointerDown={() => { if (!readOnly) setDragging(true) }}
        onPointerUp={() => setDragging(false)}
      >把这个放到正确位置<b>{token.label}</b></div> : null}
    </div>
    {showFeedback && placed && <div className={`demo-feedback ${right ? 'success' : 'again'}`} role="status"><b>{right ? '标对了！' : '位置不对'}</b><span>{right ? example.explanation : example.tip}</span></div>}
  </>
}
