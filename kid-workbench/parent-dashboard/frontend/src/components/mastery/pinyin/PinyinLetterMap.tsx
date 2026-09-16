import { useChildStore } from '../../../store/childStore'
import { LETTER_TYPES, formatPinyinDate, pinyinState, type PinyinGroup, type PinyinPoint, type PinyinView } from '../../../lib/pinyinMastery'

export function PinyinGroupProgress({ group, view = 'all' }: { group: PinyinGroup; view?: PinyinView }) {
  const mastered = group.points.filter(p => {
    const state = pinyinState(p)
    return view === 'all' ? state.complete : state.skills.find(s => s.code === view)?.lit
  }).length
  const tip = `${group.name} · ${view === 'all' ? '完全掌握' : LETTER_TYPES.find(t => t.code === view)?.name} ${mastered} / ${group.points.length}`
  return <div className="pinyin-group-progress" role="img" title={tip} aria-label={tip}><span style={{ width: `${group.points.length ? mastered / group.points.length * 100 : 0}%` }} /></div>
}

export function PinyinCard({ point, view = 'all' }: { point: PinyinPoint; view?: PinyinView }) {
  const open = useChildStore(s => s.openKp)
  const selected = useChildStore(s => s.kpDrawerId === point.id)
  const state = pinyinState(point)
  const current = state.skills.find(s => s.code === view)
  const lit = current ? current.lit : state.complete
  const partial = view === 'all' && !state.complete && state.mastered > 0
  const title = point.syllable || point.title
  const composition = point.kind === 'syllable' && point.final ? `${point.initial ? `${point.initial} + ` : ''}${point.final}` : ''
  const skills = state.skills.map(s => `${s.name}：${s.detail}`).join('；')
  const last = point.last_at ? `最近作答 ${formatPinyinDate(point.last_at, true)}` : ''
  const tip = [title, composition, state.label, skills, last].filter(Boolean).join(' · ')
  return <button className={`pinyin-card ${lit ? 'is-complete' : partial ? 'is-partial' : ''}`} data-selected={selected} onClick={event => { event.currentTarget.focus(); open(point.id) }} title={tip} aria-label={`${tip}；查看详情`}>
    <span className="pinyin-glyph">{title}</span>
    {state.complete && <span className="pinyin-check" aria-label="完全掌握">✓</span>}
    {point.kind === 'letter' && <span className="pinyin-marks">{state.skills.map(s => <span key={s.code} className={s.lit ? 'lit' : 'unlit'} data-active={view === s.code} aria-label={`${s.name}：${s.lit ? '已掌握' : '未掌握'}`}>{s.short}</span>)}</span>}
  </button>
}

export function PinyinLetterMap({ groups, view, setView }: { groups: PinyinGroup[]; view: PinyinView; setView: (view: PinyinView) => void }) {
  const visible = groups.filter(g => g.visiblePoints.length)
  return <section id="pinyin-letter-map" className="pinyin-map" tabIndex={-1} aria-labelledby="pinyin-letter-title">
    <header className="pinyin-map-heading"><h2 id="pinyin-letter-title">字母掌握地图</h2><p>听｜找｜认：亮起已掌握，暗色未掌握 · 三项全过显示 ✓</p></header>
    <div className="pinyin-tabs" role="group" aria-label="选择字母掌握视图">{[{ code: 'all', name: '整体掌握' }, ...LETTER_TYPES].map(type => <button key={type.code} aria-pressed={view === type.code} onClick={() => setView(type.code as PinyinView)}>{type.name}</button>)}</div>
    {!groups.length ? <div className="dash-card pinyin-empty">暂无字母内容。</div> : !visible.length ? <p className="pinyin-empty">没有匹配的字母。</p> : visible.map(group => <section key={group.code} className="dash-card pinyin-group" aria-label={group.name}>
      <header><h3>{group.name}</h3><PinyinGroupProgress group={group} view={view} /></header>
      <div className="pinyin-grid">{group.visiblePoints.map(point => <PinyinCard key={point.id} point={point} view={view} />)}</div>
    </section>)}
  </section>
}
