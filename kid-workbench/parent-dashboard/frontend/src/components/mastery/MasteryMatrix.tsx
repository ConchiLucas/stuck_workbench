import { useState } from 'react'
import { useMatrix } from '../../api/dashboard'
import type { MatrixModule, MatrixPoint } from '../../api/types'
import { useChildStore } from '../../store/childStore'
import { type MasteryStatus } from '../../theme'
import { MasteryCell } from './MasteryCell'
import { MathQuizSheet } from './MathQuizSheet'
import { StatusLegend } from './StatusLegend'

const LITERACY_SKILL_LABEL: Record<string, string> = {
  glyph_sense: '义',
  sense_char: '字',
  write_char: '写',
}

const PINYIN_SKILL_LABEL: Record<string, string> = {
  inword: '例',
  listen: '音',
}

const ENGLISH_SKILL_LABEL: Record<string, string> = {
  listen: '听',
  picture: '图',
  build: '组',
  type: '写',
  read: '读',
}

export function MasteryMatrix({ subject }: { subject: string }) {
  const childId = useChildStore((s) => s.childId)
  const { data, isLoading } = useMatrix(childId, subject)
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const [quizModule, setQuizModule] = useState<MatrixModule | null>(null)
  const literacy = subject === 'literacy'
  const pinyin = subject === 'pinyin'
  const english = subject === 'english'
  const skillSubject = literacy || pinyin || english
  const skillLabel = literacy ? LITERACY_SKILL_LABEL : pinyin ? PINYIN_SKILL_LABEL : ENGLISH_SKILL_LABEL
  const skillCodes = literacy ? ['glyph_sense', 'sense_char', 'write_char'] : pinyin ? ['inword', 'listen'] : ['listen', 'picture', 'build', 'type', 'read']
  const showQuizButton = (mod: MatrixModule) =>
    (subject === 'math' && (mod.code === 'add10' || mod.code === 'sub10' || mod.code === 'shape'))

  if (isLoading) return <div className="text-sm text-night-mute">加载中…</div>
  if (!data) return null

  const masteredLabel = skillSubject ? '完全掌握' : '已掌握'

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <h3 className="font-semibold text-white">
          {data.subject.icon} {data.subject.name}
          <span className="ml-2 text-sm font-normal text-night-mute">
            {masteredLabel} {data.subject.counts.mastered + data.subject.counts.review_due}/{data.subject.total}
          </span>
        </h3>
        <StatusLegend />
      </div>
      {literacy ? (
        <p className="text-xs text-night-mute -mt-2">
          看字选义、看义选字、手写全部掌握，才算完全掌握这个字
        </p>
      ) : null}
      {pinyin ? (
        <p className="text-xs text-night-mute -mt-2">
          字母变绿 = 例字选音 / 听音选字母 两种题型都过；两点颜色对应各题型状态
        </p>
      ) : null}
      {english ? (
        <p className="text-xs text-night-mute -mt-2">
          听音选词、看图选词、组句子、写单词、读一读五项均掌握后完全点亮；演示账本不能当作真实学习结果
        </p>
      ) : null}

      {data.modules.map((mod) => {
        const started = mod.total - mod.points.filter((p) => p.status === 'not_started').length
        const open = literacy ? true : expanded[mod.code] ?? started > 0
        const skillCounts = skillSubject ? countModuleSkills(mod.points, skillCodes) : null
        return (
          <div key={mod.code} className="dash-card p-4">
            <div className="flex w-full items-start justify-between gap-3">
              <ModuleHeader
                literacy={literacy}
                name={mod.name}
                masteredLabel={skillSubject ? '完全掌握' : ''}
                mastered={mod.mastered}
                total={mod.total}
                skillCounts={skillCounts}
                skillLabel={skillLabel}
                onToggle={() => setExpanded((e) => ({ ...e, [mod.code]: !open }))}
              />
              <div className="flex items-center gap-2 shrink-0">
                {showQuizButton(mod) ? (
                  <button
                    type="button"
                    className="rounded-lg border border-white/15 px-2.5 py-1 text-xs text-night-mute hover:border-brand-300 hover:text-brand-300"
                    onClick={() => setQuizModule(mod)}
                  >
                    题目
                  </button>
                ) : null}
                {literacy ? null : (
                  <button
                    type="button"
                    className="text-xs text-night-mute"
                    onClick={() => setExpanded((e) => ({ ...e, [mod.code]: !open }))}
                  >
                    {open ? '收起' : '展开'}
                  </button>
                )}
              </div>
            </div>
            {open && (
              <div className="mt-3 flex flex-wrap gap-1.5">
                {mod.points.map((p) => (
                  <MasteryCell
                    key={p.id}
                    point={p}
                    skillMode={skillSubject}
                    skillCodes={skillCodes}
                    skillLabel={skillLabel}
                    showSkillLabels={literacy}
                  />
                ))}
              </div>
            )}
          </div>
        )
      })}

      {quizModule &&
      subject === 'math' &&
      (quizModule.code === 'add10' || quizModule.code === 'sub10' || quizModule.code === 'shape') ? (
        <MathQuizSheet module={quizModule} onClose={() => setQuizModule(null)} />
      ) : null}
    </div>
  )
}

function ModuleHeader({
  literacy,
  name,
  masteredLabel,
  mastered,
  total,
  skillCounts,
  skillLabel,
  onToggle,
}: {
  literacy: boolean
  name: string
  masteredLabel: string
  mastered: number
  total: number
  skillCounts: { code: string; done: number }[] | null
  skillLabel: Record<string, string>
  onToggle: () => void
}) {
  const body = (
    <>
      <span className="font-medium text-white">
        {name}
        <span className="ml-2 text-xs text-night-mute">
          {masteredLabel} {mastered}/{total}
        </span>
      </span>
      {skillCounts ? (
        <div className="mt-1 flex flex-wrap gap-2 text-[11px] text-night-mute">
          {skillCounts.map((s) => (
            <span key={s.code}>
              {skillLabel[s.code] ?? s.code} {s.done}/{total}
            </span>
          ))}
        </div>
      ) : null}
    </>
  )
  if (literacy) return <div className="flex-1 text-left">{body}</div>
  return (
    <button type="button" className="flex-1 text-left" onClick={onToggle}>
      {body}
    </button>
  )
}

function countModuleSkills(points: MatrixPoint[], codes: string[]) {
  return codes.map((code) => {
    let done = 0
    for (const p of points) {
      const sk = p.skills?.find((s) => s.code === code)
      const st = (sk?.status ?? 'not_started') as MasteryStatus
      if (st === 'mastered' || st === 'review_due') done++
    }
    return { code, done }
  })
}
