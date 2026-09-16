import type { ShapeKey } from '../api/types'

const names: Record<ShapeKey, string> = {
  circle: 'circle 圆形', square: 'square 正方形', rect: 'rect 长方形', triangle: 'triangle 三角形',
  oval: 'oval 椭圆形', trapezoid: 'trapezoid 梯形', rhombus: 'rhombus 菱形', star: 'star 五角星',
}

export function ShapeGlyph({ shape, className = '' }: { shape: ShapeKey; className?: string }) {
  const common = { fill: 'currentColor', stroke: 'currentColor', strokeWidth: 3, strokeLinejoin: 'round' as const }
  return <svg className={`shape-glyph ${className}`} viewBox="0 0 120 120" role="img" aria-label={names[shape]}>
    {shape === 'circle' && <circle {...common} cx="60" cy="60" r="39" />}
    {shape === 'square' && <rect {...common} x="22" y="22" width="76" height="76" rx="5" />}
    {shape === 'rect' && <rect {...common} x="13" y="30" width="94" height="60" rx="5" />}
    {shape === 'triangle' && <path {...common} d="M60 16 107 99H13Z" />}
    {shape === 'oval' && <ellipse {...common} cx="60" cy="60" rx="48" ry="32" />}
    {shape === 'trapezoid' && <path {...common} d="M31 24h58l20 73H11Z" />}
    {shape === 'rhombus' && <path {...common} d="m60 10 49 50-49 50L11 60Z" />}
    {shape === 'star' && <path {...common} d="m60 9 14 32 35 4-26 23 8 35-31-18-31 18 8-35-26-23 35-4Z" />}
  </svg>
}
