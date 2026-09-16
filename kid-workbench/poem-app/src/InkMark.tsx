const paths: Record<string, string> = {
  moon: 'M38 12c-8 4-18 14-18 28 0 16 14 28 30 28 10 0 18-4 24-10-18 4-36-8-36-26 0-8 4-14 10-20-4 0-8 0-10 0z',
  frost: 'M32 8v48M16 18l32 28M48 18 16 46M12 32h40',
  goose: 'M18 40c8-18 28-22 36-10 4 6 2 16-8 20-12 6-24 2-28-10zm28-12c6-8 16-6 14 4',
  hoe: 'M16 12h28v8H28v36h-8V20H16z',
  kite: 'M32 8 48 32 32 56 16 32z M32 32v24',
  peach: 'M32 16c12 0 20 16 16 28-4 12-28 12-32 0C12 32 20 16 32 16z',
  lotus: 'M32 52c12-6 20-18 20-28-8 4-16 8-20 16-4-8-12-12-20-16 0 10 8 22 20 28z',
  boat: 'M10 36h44l-8 12H18z M24 20h16v16H24z',
  sun: 'M32 16a16 16 0 1 1 0 32 16 16 0 0 1 0-32z',
  lamp: 'M24 12h16l8 20H16z M28 32v20h8V32',
  bird: 'M12 36c16-18 28-8 40-16-8 12-4 24-16 28-10 4-20-2-24-12z',
  school: 'M12 28 32 12 52 28v24H12z M28 36h8v16h-8z',
  home: 'M12 30 32 14 52 30v22H12z',
  market: 'M10 24h44v8H10z M16 32v20M32 32v20M48 32v20',
  palace: 'M8 24h48v8H8z M16 32v20M32 16v36M48 32v20',
  bell: 'M20 20c0-8 24-8 24 0v16H20z M18 36h28v6H18z',
  horse: 'M12 40 20 20h12l8 12 12 4v12H16z',
  drum: 'M12 24h40v8H12z M20 32v16M44 32v16',
  rain: 'M16 16h32c0 12-8 16-16 16S16 28 16 16z M22 40v12M32 38v14M42 40v12',
  song: 'M20 16h8v32h-8z M28 16 48 12v12L28 32z',
  star: 'M32 10 38 26h16l-13 10 5 16-14-9-14 9 5-16-13-10h16z',
  sail: 'M20 12h4v40h-4z M24 16l24 16H24z',
  leaf: 'M16 48c8-28 32-32 36-8-16 4-24 12-36 8z',
  flower: 'M32 20c8 0 12 12 0 16-12-4-8-16 0-16zm0 16c0 12-12 20-16 8 12 0 16-8 16-8zm0 0c0 12 12 20 16 8-12 0-16-8-16-8z',
  black: 'M16 16h32v32H16z',
  red: 'M16 16h32v32H16z',
  white: 'M16 16h32v32H16z',
  yellow: 'M16 16h32v32H16z',
  green: 'M16 16h32v32H16z',
  blue: 'M16 16h32v32H16z',
  straight: 'M32 10v44',
  curve: 'M16 48c4-28 28-28 32 0',
  none: 'M16 16l32 32M48 16 16 48',
  short: 'M20 32h24',
  poet: 'M24 16c8 0 12 8 8 16-8 2-16-4-16-12 0-2 4-4 8-4zm-8 36c0-12 8-16 16-16s16 4 16 16',
  child: 'M28 18c6 0 8 6 4 10-6 0-12-4-12-8 0-1 4-2 8-2zm-6 34c0-10 6-14 12-14s12 4 12 14',
  fisher: 'M20 40c8-16 24-8 28 4H18z M40 16l8 20',
  farmer: 'M18 44c6-18 22-18 28 0H18z M22 20h20v8H22z',
  sit: 'M18 20h10v24H18z M28 32h18v6H28z',
  sleep: 'M12 36h40M20 28h8M36 28h8',
  climb: 'M16 48 32 16 48 48 M24 36h16',
  close: 'M16 28c8 12 24 12 32 0',
  yangtze: 'M8 28c12 8 20-8 32 0s16 8 16 8',
  'yellow-river': 'M8 36c12-10 24 6 48-4',
  pond: 'M12 32c8 12 32 12 40 0-8-8-32-8-40 0z',
  well: 'M20 16h24v32H20z M16 28h32',
  gone: 'M16 32h32M32 16v32',
  cicada: 'M20 24h24c0 16-8 24-12 24s-12-8-12-24z',
  ship: 'M8 36h48L44 52H20z M24 16h16v20H24z',
  cart: 'M12 36h32l8-12H28L20 36z M20 44a6 6 0 1 0 0 0z',
  snow: 'M32 12v40M16 20l32 24M48 20 16 44',
  silent: 'M16 24h8v16h-8z M28 20 48 12v12L28 40z M40 28l12 12M52 28 40 40',
  color: 'M16 20h14v24H16z M34 20h14v24H34z',
  loud: 'M16 24h8v16h-8z M28 16l16-8v40l-16-8z',
  sing: 'M18 20h8v28h-8z M26 20l22-6v14L26 34z',
  hot: 'M24 16c0 12 16 12 16 24 0 8-8 12-16 12s-16-4-16-12c0-12 16-12 16-24z',
  garden: 'M16 44h32M20 44 32 16 44 44',
  zoo: 'M14 40c8-20 28-16 36-4-12 8-28 4-36 4z',
  painting: 'M14 12h36v40H14z M22 20h20v8H22z',
  bag: 'M22 16h20v8H22z M18 24h28v28H18z',
  fly: 'M12 32c16-16 24 0 40-8-8 16-16 20-40 8z',
  swim: 'M12 28c12 8 16-8 28 0s12 8 12 8',
  still: 'M20 16h24v32H20z',
  summer: 'M32 12a20 20 0 1 1 0 40',
  spring: 'M16 40c8-20 24-20 32 0',
  autumn: 'M18 20 32 48 46 20',
  winter: 'M32 12v40M18 22l28 20M46 22 18 42',
}

const fills: Record<string, string> = {
  red: '#9e2a2b',
  white: '#f7f1e4',
  yellow: '#c5a572',
  green: '#3d5a45',
  blue: '#3d5368',
  black: '#1c1814',
}

export function InkMark({ visual, label }: { visual: string; label: string }) {
  if (visual === 'line' || visual === 'char') {
    return <span className="ink-glyph">{label.slice(0, 1)}</span>
  }
  const d = paths[visual]
  if (!d) {
    return <span className="ink-seal">{label.slice(0, 1)}</span>
  }
  const stroke = visual === 'white' || visual === 'frost' || visual === 'snow' ? '#6a5e4e' : '#1c1814'
  return (
    <svg className="ink-mark" viewBox="0 0 64 64" aria-hidden="true">
      <path
        d={d}
        fill={fills[visual] ?? 'none'}
        stroke={stroke}
        strokeWidth={visual === 'frost' || visual === 'snow' || visual === 'straight' || visual === 'curve' ? 3.4 : fills[visual] ? 1.4 : 2.6}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
