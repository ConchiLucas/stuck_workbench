export function ScienceGlyph({ name, label }: { name?: string; label?: string }) {
  const key = name || ''
  return (
    <svg className="science-glyph" viewBox="0 0 64 64" role="img" aria-hidden={label ? undefined : true} aria-label={label}>
      {iconPath(key)}
    </svg>
  )
}

function iconPath(key: string) {
  switch (key) {
    case 'duck':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="28" cy="28" r="12" /><path d="M40 28h12l-6 6M22 40c4 10 16 10 22 2" /></g>
    case 'cat':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M18 28 12 12l12 8 8-8 12-8-6 16" /><circle cx="32" cy="36" r="14" /></g>
    case 'rabbit':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M24 28c-2-16 8-18 8-6 0-12 10-10 8 6" /><circle cx="32" cy="38" r="12" /></g>
    case 'sun':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="32" cy="32" r="10" /><path d="M32 8v8M32 48v8M8 32h8M48 32h8M14 14l6 6M44 44l6 6M14 50l6-6M44 20l6-6" /></g>
    case 'lamp':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M24 18h16l4 14H20zM32 32v12M24 52h16" /></g>
    case 'firefly':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="32" cy="36" r="8" /><path d="M24 20l8 10 8-10" /></g>
    case 'ice':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M32 10 50 32 32 54 14 32z" /></g>
    case 'heat':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M24 48c0-10 16-10 16-22 4 6 8 12 8 22a16 16 0 0 1-32 0z" /></g>
    case 'cold':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M32 12v40M18 20l28 24M46 20 18 44" /></g>
    case 'heavy':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M16 24h32l-4 24H20zM32 12v12" /></g>
    case 'fish':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M12 32c10-12 30-12 32 0-2 12-22 12-32 0z" /><path d="M44 32l12-8v16z" /></g>
    case 'frog':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="24" cy="22" r="6" /><circle cx="40" cy="22" r="6" /><path d="M16 36c4 12 28 12 32 0" /></g>
    case 'bear':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="20" cy="18" r="8" /><circle cx="44" cy="18" r="8" /><circle cx="32" cy="36" r="16" /></g>
    case 'camel':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M10 44c8-18 12-8 20-8s10-12 22-6v18" /></g>
    case 'bee':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><ellipse cx="32" cy="36" rx="14" ry="10" /><path d="M24 20h16M22 36h20" /></g>
    case 'panda':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="32" cy="34" r="16" /><circle cx="22" cy="20" r="6" /><circle cx="42" cy="20" r="6" /></g>
    case 'giraffe':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M28 54V18h8v12l10 8" /></g>
    case 'bamboo':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M28 54V10M36 54V14M20 24h24M20 38h24" /></g>
    case 'nectar':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="32" cy="24" r="10" /><path d="M32 34v18M24 46h16" /></g>
    case 'leaf':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M16 40c8-22 28-22 32 0-12 14-24 14-32 0z" /><path d="M20 40c8 0 16-8 24-18" /></g>
    case 'ear':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M40 16c10 8 10 28 0 36-12 0-16-10-16-20s4-18 16-16z" /></g>
    case 'eye':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M8 32c8-14 40-14 48 0-8 14-40 14-48 0z" /><circle cx="32" cy="32" r="6" /></g>
    case 'nose':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M32 12c6 10 8 20 0 32-8-12-6-22 0-32z" /></g>
    case 'see':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M8 32c8-14 40-14 48 0" /><circle cx="32" cy="34" r="6" /></g>
    case 'hear':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M20 20v24M32 16v32M44 22v20" /></g>
    case 'smell':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="32" cy="24" r="8" /><path d="M32 32c-8 8 8 12 0 20" /></g>
    case 'seed':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><ellipse cx="32" cy="36" rx="12" ry="16" /></g>
    case 'sprout':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M32 54V28M32 28c-10-12-18-4-18 4 10 0 18 8 18 16 0-8 8-16 18-16 0-8-8-16-18-4z" /></g>
    case 'seedling':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M32 54V20M18 28c8 0 14 8 14 16M46 24c-8 4-14 12-14 20" /></g>
    case 'egg':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><ellipse cx="32" cy="34" rx="12" ry="18" /></g>
    case 'caterpillar':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><circle cx="18" cy="36" r="6" /><circle cx="32" cy="36" r="6" /><circle cx="46" cy="36" r="6" /></g>
    case 'butterfly':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M32 16v32M16 20c12 4 12 16 0 24M48 20c-12 4-12 16 0 24" /></g>
    case 'water':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M32 12c12 16 16 28 0 40-16-12-12-24 0-40z" /></g>
    case 'vapor':
      return <g fill="none" stroke="currentColor" strokeWidth="3"><path d="M18 44c8-8 8-16 0-24M32 48c8-8 8-20 0-28M46 44c8-8 8-16 0-24" /></g>
    default:
      return <circle cx="32" cy="32" r="14" fill="none" stroke="currentColor" strokeWidth="3" />
  }
}

export function PlantDiagram() {
  return (
    <svg className="plant-structure" viewBox="0 0 200 240" aria-hidden="true">
      <rect x="0" y="168" width="200" height="72" fill="#e8d9b8" />
      <path d="M100 168V72" stroke="#2f7a46" strokeWidth="8" fill="none" />
      <path d="M100 112C58 96 40 128 48 150" stroke="#2f7a46" strokeWidth="7" fill="none" />
      <path d="M100 128C142 108 164 132 156 154" stroke="#2f7a46" strokeWidth="7" fill="none" />
      <circle cx="100" cy="44" r="28" fill="#f2c14e" />
      <circle cx="100" cy="44" r="10" fill="#c47c16" />
      <path d="M70 176c8 18 20 28 30 32M130 176c-8 18-20 28-30 32M86 186c4 14 12 22 14 24M114 186c-4 14-12 22-14 24" stroke="#8a5a2b" strokeWidth="4" fill="none" />
    </svg>
  )
}
