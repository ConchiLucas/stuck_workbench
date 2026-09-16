import type { LogicObject } from './types'

const SHAPES = new Set([
  'circle', 'square', 'triangle', 'star', 'diamond', 'heart', 'apple', 'banana', 'pear', 'grape',
  'carrot', 'cat', 'dog', 'bird', 'fish', 'car', 'bus', 'bike', 'skate', 'tree', 'flower', 'sun',
  'leaf', 'snow', 'arrow', 'giraffe', 'mouse', 'rabbit', 'one', 'two', 'three', 'four',
])

export function isNamedGlyph(glyph: string) {
  return SHAPES.has(glyph)
}

export function LogicGlyph({ object, label }: { object: LogicObject; label?: string }) {
  const fill = object.fill || '#6b5a86'
  const count = object.count && object.count > 1 ? Math.min(object.count, 4) : 1
  const rotate = object.rotate || 0
  if (!isNamedGlyph(object.glyph)) {
    return <span className="logic-emoji" aria-hidden={label ? undefined : true}>{object.glyph}</span>
  }
  const inner = count === 1
    ? <g transform={rotate ? `rotate(${rotate} 32 32)` : undefined}>{shape(object.glyph, fill)}</g>
    : grouped(object.glyph, fill, count)
  return (
    <svg className="logic-glyph" viewBox="0 0 64 64" role="img" aria-hidden={label ? undefined : true} aria-label={label}>
      {inner}
    </svg>
  )
}

function grouped(glyph: string, fill: string, count: number) {
  const positions = count === 2
    ? [[20, 32], [44, 32]]
    : count === 4
      ? [[18, 18], [46, 18], [18, 46], [46, 46]]
      : [[16, 20], [48, 20], [32, 46]]
  return (
    <g>
      {positions.slice(0, count).map(([x, y], i) => (
        <g key={i} transform={`translate(${x} ${y}) scale(0.42) translate(-32 -32)`}>{shape(glyph, fill)}</g>
      ))}
    </g>
  )
}

function shape(glyph: string, fill: string) {
  switch (glyph) {
    case 'circle': return <circle cx="32" cy="32" r="18" fill={fill} />
    case 'square': return <rect x="14" y="14" width="36" height="36" rx="4" fill={fill} />
    case 'triangle': return <polygon points="32,12 52,50 12,50" fill={fill} />
    case 'star': return <polygon points="32,10 38,26 56,26 42,36 48,52 32,42 16,52 22,36 8,26 26,26" fill={fill} />
    case 'diamond': return <polygon points="32,10 54,32 32,54 10,32" fill={fill} />
    case 'heart': return <path fill={fill} d="M32 52 C14 38 10 24 18 16 c6-6 14-2 14 6 0-8 8-12 14-6 8 8 4 22-14 36z" />
    case 'apple': return <g fill={fill}><ellipse cx="32" cy="36" rx="16" ry="18" /><path d="M32 18c0-8 8-10 10-4" fill="none" stroke="#3f7a3a" strokeWidth="3" /><ellipse cx="28" cy="22" rx="6" ry="3" fill="#3f7a3a" /></g>
    case 'banana': return <path fill={fill} d="M16 18c18 2 34 18 30 34-16-8-28-20-30-34z" />
    case 'pear': return <g fill={fill}><circle cx="32" cy="24" r="10" /><ellipse cx="32" cy="42" rx="16" ry="14" /><path d="M32 12v8" fill="none" stroke="#3f7a3a" strokeWidth="3" /></g>
    case 'grape': return <g fill={fill}><circle cx="24" cy="28" r="8" /><circle cx="40" cy="28" r="8" /><circle cx="32" cy="42" r="8" /></g>
    case 'carrot': return <g><polygon points="32,18 48,52 16,52" fill={fill} /><path d="M24 16h16" stroke="#3f7a3a" strokeWidth="4" /></g>
    case 'cat': return <g fill={fill}><circle cx="32" cy="36" r="16" /><polygon points="16,28 18,12 28,24" /><polygon points="48,28 46,12 36,24" /><circle cx="26" cy="34" r="3" fill="#fff" /><circle cx="38" cy="34" r="3" fill="#fff" /></g>
    case 'dog': return <g fill={fill}><ellipse cx="32" cy="36" rx="16" ry="14" /><ellipse cx="18" cy="28" rx="8" ry="10" /><ellipse cx="46" cy="28" rx="8" ry="10" /><circle cx="26" cy="36" r="3" fill="#fff" /><circle cx="38" cy="36" r="3" fill="#fff" /></g>
    case 'bird': return <g fill={fill}><ellipse cx="30" cy="34" rx="16" ry="10" /><polygon points="46,34 58,28 46,40" /><circle cx="22" cy="32" r="3" fill="#fff" /></g>
    case 'fish': return <g fill={fill}><ellipse cx="28" cy="32" rx="16" ry="10" /><polygon points="44,32 58,20 58,44" /><circle cx="20" cy="30" r="3" fill="#fff" /></g>
    case 'car': return <g fill={fill}><rect x="10" y="28" width="44" height="16" rx="4" /><rect x="18" y="18" width="24" height="12" rx="3" /><circle cx="20" cy="46" r="6" fill="#333" /><circle cx="44" cy="46" r="6" fill="#333" /></g>
    case 'bus': return <g fill={fill}><rect x="8" y="18" width="48" height="26" rx="4" /><rect x="14" y="22" width="10" height="10" fill="#fff" /><rect x="28" y="22" width="10" height="10" fill="#fff" /><rect x="42" y="22" width="8" height="10" fill="#fff" /><circle cx="20" cy="48" r="6" fill="#333" /><circle cx="46" cy="48" r="6" fill="#333" /></g>
    case 'bike': return <g fill="none" stroke={fill} strokeWidth="3"><circle cx="18" cy="42" r="10" /><circle cx="46" cy="42" r="10" /><path d="M18 42 L32 22 L46 42 M32 22 L28 16" /></g>
    case 'skate': return <g fill={fill}><ellipse cx="32" cy="28" rx="16" ry="10" /><rect x="12" y="40" width="40" height="6" rx="3" /><circle cx="20" cy="50" r="4" /><circle cx="32" cy="50" r="4" /><circle cx="44" cy="50" r="4" /></g>
    case 'tree': return <g><circle cx="32" cy="26" r="16" fill={fill} /><rect x="28" y="38" width="8" height="16" fill="#8b5a2b" /></g>
    case 'flower': return <g fill={fill}><circle cx="32" cy="20" r="8" /><circle cx="20" cy="32" r="8" /><circle cx="44" cy="32" r="8" /><circle cx="32" cy="44" r="8" /><circle cx="32" cy="32" r="6" fill="#f4d35e" /></g>
    case 'sun': return <g fill={fill}><circle cx="32" cy="32" r="12" /><g stroke={fill} strokeWidth="3"><path d="M32 8v8M32 48v8M8 32h8M48 32h8M14 14l6 6M44 44l6 6M14 50l6-6M44 20l6-6" /></g></g>
    case 'leaf': return <path fill={fill} d="M32 12c16 10 20 28 0 40C12 40 16 22 32 12z" />
    case 'snow': return <g fill="none" stroke={fill} strokeWidth="3"><path d="M32 10v44M14 20l36 24M50 20 14 44M18 32h28" /></g>
    case 'arrow': return <polygon points="52,32 24,16 24,24 8,24 8,40 24,40 24,48" fill={fill} />
    case 'giraffe': return <g fill={fill}><rect x="28" y="10" width="8" height="28" rx="3" /><circle cx="36" cy="12" r="8" /><ellipse cx="30" cy="48" rx="16" ry="10" /></g>
    case 'mouse': return <g fill={fill}><circle cx="22" cy="24" r="10" /><circle cx="42" cy="24" r="10" /><ellipse cx="32" cy="38" rx="16" ry="12" /><circle cx="26" cy="36" r="3" fill="#fff" /><circle cx="38" cy="36" r="3" fill="#fff" /></g>
    case 'rabbit': return <g fill={fill}><ellipse cx="24" cy="16" rx="6" ry="16" /><ellipse cx="40" cy="16" rx="6" ry="16" /><circle cx="32" cy="38" r="16" /></g>
    case 'one': return <text x="32" y="42" textAnchor="middle" fontSize="32" fontFamily="system-ui,sans-serif" fontWeight="700" fill={fill}>1</text>
    case 'two': return <text x="32" y="42" textAnchor="middle" fontSize="32" fontFamily="system-ui,sans-serif" fontWeight="700" fill={fill}>2</text>
    case 'three': return <text x="32" y="42" textAnchor="middle" fontSize="32" fontFamily="system-ui,sans-serif" fontWeight="700" fill={fill}>3</text>
    case 'four': return <text x="32" y="42" textAnchor="middle" fontSize="32" fontFamily="system-ui,sans-serif" fontWeight="700" fill={fill}>4</text>
    default: return <circle cx="32" cy="32" r="18" fill={fill} />
  }
}
