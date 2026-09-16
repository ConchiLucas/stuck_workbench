export function ProgressRing({ value, total, size = 96 }: {
  value: number; total: number; size?: number
}) {
  const pct = total > 0 ? value / total : 0
  const r = size / 2 - 8
  const c = 2 * Math.PI * r

  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90" aria-hidden="true">
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="#3d1a32" strokeWidth={10} />
        <circle
          cx={size / 2} cy={size / 2} r={r} fill="none" stroke="#F472A6" strokeWidth={10}
          strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - pct)}
        />
      </svg>
      <div className="pointer-events-none absolute inset-0 grid place-items-center text-sm font-semibold text-white">
        {Math.round(pct * 100)}%
      </div>
    </div>
  )
}
