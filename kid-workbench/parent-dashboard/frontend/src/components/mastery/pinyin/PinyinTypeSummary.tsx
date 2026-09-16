import { PINYIN_TYPES, barWidths, type PinyinCounts, type PinyinSkillCode } from '../../../lib/pinyinMastery'

export function PinyinBar({ counts, label }: { counts: PinyinCounts; label: string }) {
  const widths = barWidths(counts)
  const tip = `${label}：已掌握 ${counts.mastered} · 已作答未掌握 ${counts.answered} · 未作答 ${counts.unattempted}（共 ${counts.total}）`
  return <div className="pinyin-bar" role="img" title={tip} aria-label={tip}>
    <span className="is-mastered" style={{ width: `${widths.mastered}%` }} />
    <span className="is-answered" style={{ width: `${widths.answered}%` }} />
  </div>
}

export function PinyinTypeSummary({ summary, select, catalogAvailable }: {
  summary: Record<PinyinSkillCode, PinyinCounts>; select: (code: PinyinSkillCode) => void; catalogAvailable: boolean
}) {
  return <section className="dash-card pinyin-summary" aria-label="拼音四题型掌握进度">
    <div className="pinyin-type-bars">{PINYIN_TYPES.map(type => {
      const unavailable = type.code === 'blend' && !catalogAvailable
      const counts = summary[type.code]
      const tip = unavailable ? '音节内容暂不可用' : `${type.name}：已掌握 ${counts.mastered}，已作答未掌握 ${counts.answered}，未作答 ${counts.unattempted}，共 ${counts.total}`
      return <button key={type.code} onClick={() => select(type.code)} title={tip} aria-label={`${tip}；查看地图`}>
        <span>{type.name}</span><PinyinBar counts={counts} label={type.name} />
        {(unavailable || counts.total === 0) && <small>{unavailable ? '内容暂不可用' : '暂无内容'}</small>}
      </button>
    })}</div>
    <div className="pinyin-legend"><span><i className="is-mastered" />已掌握</span><span><i className="is-answered" />已作答未掌握</span><span><i className="is-unattempted" />未作答</span></div>
  </section>
}
