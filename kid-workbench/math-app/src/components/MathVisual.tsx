import type { MathVisual } from '../api/types'
import { ShapeGlyph } from './ShapeGlyph'

const emoji: Record<string, string> = { apple: '🍎', strawberry: '🍓' }

function Objects({ count, testId, object }: { count: number; testId: string; object: string }) {
  return <>{Array.from({ length: count }, (_, index) => <span key={index} data-testid={testId} data-counting-object>{emoji[object] ?? '●'}</span>)}</>
}

export function MathVisualCard({ visual }: { visual: MathVisual }) {
  if (visual.kind === 'none') return null
  if (visual.kind === 'equation') return <div className="equation" aria-label="算式">{visual.a} {visual.operator} {visual.b} = ?</div>
  if (visual.kind === 'shape') return <div className="visual-shape"><ShapeGlyph shape={visual.shape} /></div>
  if (visual.kind === 'add') return <div className="object-equation" aria-label="合起来">
    <span className="object-group"><Objects count={visual.leftCount} testId="left-object" object={visual.object} /></span>
    <b>+</b>
    <span className="object-group"><Objects count={visual.rightCount} testId="right-object" object={visual.object} /></span>
  </div>
  const remaining = Math.max(0, visual.leftCount - visual.rightCount)
  return <div className="object-equation" aria-label="拿走一些">
    <span className="object-group"><Objects count={remaining} testId="remaining-object" object={visual.object} /></span>
    <span className="object-group removed"><Objects count={visual.rightCount} testId="removed-object" object={visual.object} /></span>
  </div>
}
