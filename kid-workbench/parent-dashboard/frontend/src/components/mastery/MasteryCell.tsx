import { STATUS_STYLE, type MasteryStatus } from '../../theme'
import type { MatrixPoint, MatrixSkill } from '../../api/types'
import { useChildStore } from '../../store/childStore'

export function MasteryCell({
  point,
  skillMode,
  skillCodes,
  skillLabel,
  showSkillLabels,
}: {
  point: MatrixPoint
  skillMode?: boolean
  skillCodes?: string[]
  skillLabel?: Record<string, string>
  showSkillLabels?: boolean
}) {
  const openKp = useChildStore((s) => s.openKp)
  const style = STATUS_STYLE[point.status]
  const tip = point.attempts > 0
    ? `${point.title} · ${style.label} · 正确率 ${Math.round(point.accuracy * 100)}%（练了 ${point.attempts} 次）`
    : `${point.title} · ${style.label}`

  if (skillMode && skillCodes?.length) {
    const skills = ensureSkills(point.skills, skillCodes)
    const fully = skills.every((s) => s.status === 'mastered' || s.status === 'review_due')
    return (
      <button
        type="button"
        title={tip + (fully ? ' · 认写均已过' : ' · 认和写都过才算完全掌握')}
        onClick={() => openKp(point.id)}
        className="flex flex-col items-center gap-1 rounded-xl px-1.5 py-1.5 min-w-10
                   transition-transform hover:scale-105 hover:ring-2 focus:outline-none focus:ring-2"
        style={{
          backgroundColor: style.bg,
          color: style.text,
          boxShadow: fully
            ? `inset 0 0 0 2px ${style.ring}`
            : `inset 0 0 0 1px ${style.ring}33`,
        }}
      >
        <span className="text-sm font-semibold leading-none">{point.title}</span>
        <span className="flex gap-0.5" aria-label="题型掌握">
          {skills.map((s) => {
            const st = STATUS_STYLE[s.status]
            const label = skillLabel?.[s.code] ?? s.code
            if (showSkillLabels) {
              return (
                <span
                  key={s.code}
                  title={`${label} · ${st.label}`}
                  className="rounded px-0.5 text-[9px] font-semibold leading-tight text-white"
                  style={{ backgroundColor: st.ring }}
                >
                  {label}
                </span>
              )
            }
            return (
              <span
                key={s.code}
                title={`${label} · ${st.label}`}
                className="h-1.5 w-1.5 rounded-full"
                style={{ backgroundColor: st.ring }}
              />
            )
          })}
        </span>
      </button>
    )
  }

  return (
    <button
      type="button"
      title={tip}
      onClick={() => openKp(point.id)}
      className="h-8 min-w-8 px-1 rounded-lg text-[11px] font-medium leading-none
                 transition-transform hover:scale-110 hover:ring-2 focus:outline-none focus:ring-2"
      style={{ backgroundColor: style.bg, color: style.text, boxShadow: `inset 0 0 0 1px ${style.ring}22` }}
    >
      {point.title.length <= 4 ? point.title : point.title.slice(0, 3) + '…'}
    </button>
  )
}

function ensureSkills(skills: MatrixSkill[] | undefined, codes: string[]): MatrixSkill[] {
  const byCode = new Map((skills ?? []).map((s) => [s.code, s]))
  return codes.map((code) => byCode.get(code) ?? {
    code,
    status: 'not_started' as MasteryStatus,
    accuracy: 0,
    attempts: 0,
  })
}
