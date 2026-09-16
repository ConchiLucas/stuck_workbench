import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MathVisualCard } from './MathVisual'
import { ShapeGlyph } from './ShapeGlyph'

describe('math visuals', () => {
  it('renders an equation without counting objects', () => {
    const { container } = render(<MathVisualCard visual={{ kind: 'equation', a: 3, b: 2, operator: '+' }} />)
    expect(screen.getByText('3 + 2 = ?')).toBeInTheDocument()
    expect(container.querySelectorAll('[data-counting-object]')).toHaveLength(0)
  })

  it('renders addition as two visible groups', () => {
    render(<MathVisualCard visual={{ kind: 'add', leftCount: 2, rightCount: 3, object: 'apple' }} />)
    expect(screen.getAllByTestId('left-object')).toHaveLength(2)
    expect(screen.getAllByTestId('right-object')).toHaveLength(3)
  })

  it('marks removed objects for subtraction', () => {
    render(<MathVisualCard visual={{ kind: 'sub', leftCount: 5, rightCount: 2, object: 'strawberry' }} />)
    expect(screen.getAllByTestId('remaining-object')).toHaveLength(3)
    expect(screen.getAllByTestId('removed-object')).toHaveLength(2)
  })

  it.each(['circle', 'square', 'rect', 'triangle', 'oval', 'trapezoid', 'rhombus', 'star'] as const)(
    'renders an accessible %s shape', (shape) => {
      render(<ShapeGlyph shape={shape} />)
      expect(screen.getByRole('img', { name: new RegExp(shape) })).toBeInTheDocument()
    },
  )
})
