import { useEffect, useRef, useState } from 'react'
import { ShapeGlyph, shapeKey, shapeNames, type MathExample } from './index'

export interface KidAnswer { selected: string; placements: Record<string, string>; correct: boolean | null }
export interface KidPlayerProps {
  example: MathExample
  resolveAssetUrl: (url: string) => string
  initialAnswer?: KidAnswer
  readOnly?: boolean
  onAnswer?: (answer: KidAnswer) => void
}
const objectNames: Record<string, string> = { apple: '苹果', star: '星星', strawberry: '草莓', 苹果: '苹果', 星星: '星星', 草莓: '草莓' }
const objectGlyphs: Record<string, string> = { 苹果: '🍎', 星星: '★', 草莓: '🍓' }
export function KidPlayer({ example: e, resolveAssetUrl, initialAnswer, readOnly = false, onAnswer }: KidPlayerProps) {
  const [state, setState] = useState<KidAnswer>(initialAnswer ?? { selected: '', placements: {}, correct: null })
  const [active, setActive] = useState('')
  const [audioState, setAudioState] = useState<'idle' | 'playing' | 'error'>('idle')
  const [imageError, setImageError] = useState(false)
  const audio = useRef<HTMLAudioElement | null>(null)
  useEffect(() => () => { if (audio.current) { audio.current.onended = null; audio.current.onerror = null; audio.current.pause() } }, [])
  const candidates = (e.buckets ?? []).flatMap((bucket, group) => bucket.items.map((item, index) => ({ item, group, id: `${group}-${index}` }))).sort((a,b) => a.item.localeCompare(b.item))
  const shape = (key: string) => e.shapeImageUrls?.[shapeKey(key)]
  const missingAudio = e.kind === 'audio-shape' && (!e.audioUrl || audioState === 'error')
  const locked = readOnly || state.correct === true || imageError || missingAudio
  const options = e.options?.length ? e.options : e.kind === 'judgement' ? e.statements?.length === 1 ? ['正确','错误'] : e.statements ?? [] : []
  const prompt = e.kind === 'choice' ? e.prompt.replace(/\s*[=＝]\s*[?？]\s*$/, '').trim() : e.kind === 'audio-shape' ? '听一听，选出图形' : e.kind === 'shape-name' ? '看一看，这是什么图形？' : e.kind === 'missing' && state.selected ? e.prompt.replace(/□|＿+|_+|\?/, state.selected) : e.prompt
  function update(next: KidAnswer) { setState(next); onAnswer?.(next) }
  function play() {
    if (!e.audioUrl) return
    audio.current?.pause()
    const player = new Audio(resolveAssetUrl(e.audioUrl))
    audio.current = player
    player.onended = () => setAudioState('idle')
    player.onerror = () => setAudioState('error')
    setAudioState('playing')
    void player.play().catch(() => { if (audio.current === player) setAudioState('error') })
  }
  function picture(key: string) {
    const href = shape(key)
    if (href) return <img className="math-kid-shape" src={resolveAssetUrl(href)} alt={shapeNames[shapeKey(key)] ?? '图形'} onError={() => setImageError(true)} />
    return <ShapeGlyph shape={key} className="math-kid-shape" />
  }
  const objectName = objectNames[e.object ?? ''] ?? e.object ?? '物品'
  const objectMark = objectGlyphs[objectName] ?? '●'
  function objects(count: number, removed = 0) {
    return Array.from({length:Math.min(100,Math.max(0,count))}, (_,i) => {
      const taken = i >= count-removed
      return <span key={i} className={`math-kid-object ${taken ? 'is-removed' : ''}`} aria-label={`${taken ? '拿走' : '剩下'}的${objectName}`}>
        {e.objectImageUrl ? <img src={resolveAssetUrl(e.objectImageUrl)} alt={objectName} onError={() => setImageError(true)} /> : <b aria-hidden="true">{objectMark}</b>}
      </span>
    })
  }
  const keys = e.kind === 'shape-name' ? e.shapeKeys ?? [e.prompt] : e.shapeKeys ?? e.options ?? []
  const feedback = state.correct === null ? null : readOnly ? (state.correct ? '当时答对' : '当时答错') : (state.correct ? '✓ 答对了！' : '再想一想，再试一次。')
  return <div className={`math-kid-player kind-${e.kind}${readOnly ? ' is-readonly' : ''}`} data-readonly={readOnly || undefined}>
    <section className="math-kid-stem" aria-label="题目">
      <p className="math-kid-instruction">{e.kind === 'missing' ? '补一补' : e.kind === 'choice' ? '算一算' : e.kind === 'shape-sort' ? '分一分' : '看一看'}</p>
      <div className="math-kid-equation" data-length={prompt.replace(/\s/g, "").length} aria-label={prompt}>{prompt}</div>
      {e.kind === 'audio-shape' && e.audioUrl && <button className="math-kid-listen" type="button" onClick={play} aria-label="播放题目读音">{audioState === 'playing' ? '◼ 正在播放 · 重播' : '▶ 听一听'}</button>}
      {(imageError || missingAudio || audioState === 'error') && <p className="math-kid-unavailable" role="alert">{missingAudio ? '题目读音暂不可用，请稍后重试。' : imageError ? '这道题的图片素材还未准备好，请先练其他题。' : '读音播放失败，请点听一听重试。'}</p>}
      {e.kind === 'shape-name' && picture(keys[0])}
      {e.kind === 'objects' && !imageError && <div className="math-kid-counts">{e.operation === 'sub' ? <div className="math-kid-object-group">{objects(e.counts?.[0] ?? 0, e.counts?.[1] ?? 0)}<small>拿走 {e.counts?.[1] ?? 0} 个，还剩多少？</small></div> : e.counts?.map((count,i) => <div className="math-kid-count-part" key={i}>{i > 0 && <b>＋</b>}<div className="math-kid-object-group">{objects(count)}</div></div>)}</div>}
      {e.kind === 'judgement' && e.statements?.length === 1 && <div className="math-kid-equation">{e.statements[0]}</div>}
    </section>
    <section className="math-kid-work" aria-label="作答">
      {e.kind === 'shape-sort' ? <>
        <div className="math-kid-cards">{candidates.map(c => <button type="button" key={c.id} aria-label={`选择${shapeNames[shapeKey(c.item)] ?? c.item}`} aria-pressed={active === c.id} disabled={locked} onClick={() => setActive(c.id)}>{picture(c.item)}<small>{state.placements[c.id] !== undefined ? e.buckets?.[Number(state.placements[c.id])]?.label : '待分类'}</small></button>)}</div>
        <div className="math-kid-buckets">{e.buckets?.map((bucket,i) => <button type="button" key={i} aria-label={`放入${bucket.label}`} disabled={!active || locked} onClick={() => { update({...state, correct:null, placements:{...state.placements,[active]:String(i)}}); setActive('') }}><strong>{bucket.label}</strong><small>{candidates.filter(c => state.placements[c.id] === String(i)).length} 个图形</small></button>)}</div>
        <button className="math-kid-check" type="button" disabled={locked || !candidates.length || candidates.some(c => state.placements[c.id] === undefined)} onClick={() => update({...state,correct:candidates.every(c => state.placements[c.id] === String(c.group))})}>检查分类</button>
      </> : <div className={`math-player-options math-kid-options options-${options.length}`}>{options.map((option,i) => {
        const key = keys[i]
        const label = e.kind === 'audio-shape' || e.kind === 'shape-feature' ? (shapeNames[shapeKey(key)] ?? option) : option
        return <button type="button" key={`${i}-${option}`} disabled={locked} aria-label={label} aria-pressed={state.selected === option} className={`${state.selected === option ? state.correct ? 'is-correct' : 'is-wrong' : ''} ${/^[0-9]+$/.test(option) ? 'is-number' : 'is-text'}`} onClick={() => update({...state,selected:option,correct:option === e.answer || (e.kind === 'judgement' && Boolean(e.answer?.startsWith(option+'，')))})}>{e.kind === 'audio-shape' || e.kind === 'shape-feature' ? picture(key) : option}</button>
      })}</div>}
      <div className="math-kid-feedback" hidden={feedback === null}>{feedback !== null && <p role="status" className={state.correct ? 'is-correct' : 'is-wrong'}>{feedback}</p>}</div>
      {!readOnly && state.correct !== null && <button className="math-kid-again" type="button" onClick={() => { update({selected:'',placements:{},correct:null}); setActive('') }}>再做一次</button>}
    </section>
  </div>
}
