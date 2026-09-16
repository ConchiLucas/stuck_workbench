import { useParams } from 'react-router-dom'
import { useSubjects } from '../api/dashboard'
import { useChildStore } from '../store/childStore'
import { STATUS_ORDER, STATUS_STYLE } from '../theme'
import { ProgressRing } from '../components/stats/ProgressRing'
import { MasteryMatrix } from '../components/mastery/MasteryMatrix'
import { LiteracyMastery } from '../components/mastery/LiteracyMastery'
import { MathMastery } from '../components/mastery/math/MathMastery'
import { PinyinMastery } from '../components/mastery/pinyin/PinyinMastery'
import { EnglishMastery } from '../components/mastery/english/EnglishMastery'
import { PhraseMastery } from '../components/mastery/phrase/PhraseMastery'
import { ChengyuMastery } from '../components/mastery/chengyu/ChengyuMastery'
import { ScienceMastery } from '../components/mastery/science/ScienceMastery'
import { PoemMastery } from '../components/mastery/poem/PoemMastery'
import { LogicMastery } from '../components/mastery/logic/LogicMastery'

export function SubjectDetail() {
  const { code = '' } = useParams()
  const childId = useChildStore((s) => s.childId)
  const { data: subjects } = useSubjects(childId)
  if (code === 'literacy') return <LiteracyMastery />
  if (code === 'math') return <MathMastery />
  if (code === 'pinyin') return <PinyinMastery />
  if (code === 'english') return <EnglishMastery />
  if (code === 'phrase') return <PhraseMastery />
  if (code === 'chengyu') return <ChengyuMastery />
  if (code === 'science') return <ScienceMastery />
  if (code === 'poem') return <PoemMastery />
  if (code === 'logic') return <LogicMastery />
  const s = subjects?.find((x) => x.code === code)
  if (!s) return <div className="text-sm text-night-mute">加载中…</div>

  const done = s.counts.mastered + s.counts.review_due

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-6 rounded-2xl border border-white/10 bg-white/[0.055] p-5">
        <ProgressRing value={done} total={s.total} size={110} />
        <div>
          <h2 className="text-xl font-semibold text-white">{s.icon} {s.name}</h2>
          <div className="mt-1 text-sm text-night-mute">已掌握 {done} / {s.total} 个知识点</div>
          {s.week_new > 0 && (
            <div className="text-xs text-emerald-300">本周新掌握 {s.week_new} 个</div>
          )}
        </div>
        <div className="ml-auto grid grid-cols-5 gap-3 text-center">
          {STATUS_ORDER.map((k) => (
            <div key={k}>
              <div className="text-lg font-semibold" style={{ color: STATUS_STYLE[k].ring }}>
                {s.counts[k]}
              </div>
              <div className="text-xs text-night-mute">{STATUS_STYLE[k].label}</div>
            </div>
          ))}
        </div>
      </div>

      <MasteryMatrix subject={code} />
    </div>
  )
}
