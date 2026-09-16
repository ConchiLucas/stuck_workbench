import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { pinyinApi } from '../api/pinyin'
import { FourLineGrid } from '../components/FourLineGrid'
import { usePinyinAudio } from '../hooks/usePinyinAudio'

export function LearnPage() {
  const kpId = Number(useParams().kpId)
  const [glyphFailed, setGlyphFailed] = useState(false)
  const item = useQuery({ queryKey: ['item', kpId], queryFn: () => pinyinApi.item(kpId) })
  const audio = usePinyinAudio()
  if (item.isLoading) return <div className="skeleton-block">字母正在飞过来……</div>
  if (!item.data) return <button className="retry-card" onClick={() => item.refetch()}>这个拼音还没准备好，再试一次</button>
  const value = item.data
  return <section className="learn-page page-enter">
    <div className="learn-letter"><FourLineGrid>{value.hasGlyph && !glyphFailed ? <img src={pinyinApi.glyphUrl(kpId)} alt={value.letter} onError={() => setGlyphFailed(true)} /> : <strong>{value.letter}</strong>}</FourLineGrid></div>
    <div className="learn-copy"><p className="eyebrow">{value.moduleName}</p><h1>{value.letter}</h1><p>先听单读，再听它藏在词语里的声音。</p>
      <div className="sound-actions">
        {value.soloText && <button className="primary-button" onClick={() => audio.play(pinyinApi.speechUrl(kpId, 'solo'))}>🔊 听单读 · {value.soloText}</button>}
        {value.wordText && <button className="secondary-button" onClick={() => audio.play(pinyinApi.speechUrl(kpId, 'word'))}>🔊 听词语 · {value.wordText}</button>}
      </div>{audio.state === 'error' && <p className="inline-note">声音暂时没准备好，可以先看字母。</p>}
    </div>
  </section>
}
