export function StudioRing({ value, total, size = 112 }: {
  value: number
  total: number
  size?: number
}) {
  const pct = total > 0 ? value / total : 0
  const strokeWidth = 13
  const r = size / 2 - strokeWidth
  const c = 2 * Math.PI * r
  const gradientId = 'pinkNeonRing'

  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90" aria-hidden="true">
        <defs>
          <linearGradient id={gradientId} x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#ff3b88" />
            <stop offset="100%" stopColor="#ff70a5" />
          </linearGradient>
        </defs>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="#32162a"
          strokeWidth={strokeWidth}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke={`url(#${gradientId})`}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c * (1 - Math.min(Math.max(pct, 0.01), 1))}
          style={{
            filter: 'drop-shadow(0 0 6px rgba(255, 78, 136, 0.45))',
            transition: 'stroke-dashoffset 0.8s ease-out',
          }}
        />
      </svg>
      <div className="pointer-events-none absolute inset-0 grid place-items-center text-2xl font-bold tracking-tight text-white">
        {Math.round(pct * 100)}%
      </div>
    </div>
  )
}

