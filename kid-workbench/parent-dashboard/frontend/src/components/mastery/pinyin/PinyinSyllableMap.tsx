import type { PinyinGroup } from '../../../lib/pinyinMastery'
import { PinyinCard, PinyinGroupProgress } from './PinyinLetterMap'

export function PinyinSyllableMap({ groups, available }: { groups: PinyinGroup[]; available: boolean }) {
  const visible = groups.filter(g => g.visiblePoints.length)
  return <section id="pinyin-syllable-map" className="pinyin-map" tabIndex={-1} aria-labelledby="pinyin-syllable-title">
    <header className="pinyin-map-heading"><h2 id="pinyin-syllable-title">音节拼读地图</h2><p>每个带调音节单独积累，声韵拼读掌握后显示 ✓</p></header>
    {!available ? <div className="dash-card pinyin-empty" role="status">音节内容暂不可用。</div> : !groups.length ? <div className="dash-card pinyin-empty">当前还没有启用的音节内容。</div> : !visible.length ? <p className="pinyin-empty">没有匹配的音节。</p> : visible.map(group => <section key={group.code} className="dash-card pinyin-group" aria-label={group.name}>
      <header><h3>{group.name}</h3><PinyinGroupProgress group={group} /></header>
      <div className="pinyin-grid pinyin-syllable-grid">{group.visiblePoints.map(point => <PinyinCard key={point.id} point={point} />)}</div>
    </section>)}
  </section>
}
