import clsx from 'clsx'

export function StatCard({ label, value, hint, color, onClick }: {
  label: string; value: number | string; hint?: string; color: string; onClick?: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={clsx('dash-card p-4 text-left transition',
        onClick && 'hover:border-brand-500/40')}
    >
      <div className="flex items-center gap-2 text-sm text-night-mute">
        <i className="inline-block h-3 w-3 rounded" style={{ backgroundColor: color }} />
        {label}
      </div>
      <div className="mt-1 text-3xl font-semibold text-white">{value}</div>
      {hint && <div className="mt-1 text-xs text-night-mute">{hint}</div>}
    </button>
  )
}
