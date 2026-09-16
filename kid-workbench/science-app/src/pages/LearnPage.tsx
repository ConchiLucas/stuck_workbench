import { useMutation, useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { scienceApi } from '../api/science'
import { useScienceAudio } from '../hooks/useScienceAudio'
import { useChildStore } from '../store/childStore'

export function LearnPage() {
  const kpId = Number(useParams().kpId)
	const [imageFailed, setImageFailed] = useState(false)
	const childId = useChildStore((state) => state.childId)
	const navigate = useNavigate()
  const item = useQuery({ queryKey: ['item', kpId], queryFn: () => scienceApi.item(kpId) })
  const audio = useScienceAudio()
	const practice = useMutation({ mutationFn: (moduleCode: string) => scienceApi.createPlan(childId, 'module', moduleCode), onSuccess: (data) => navigate(`/practice/${data.plan.id}`) })
  if (item.isLoading) return <div className="skeleton-block">知识卡正在展开……</div>
  if (!item.data) return <button className="retry-card" onClick={() => item.refetch()}>这个科普还没准备好，再试一次</button>
  const value = item.data
  return <section className="learn-page page-enter">
    <div className="concept-visual">{value.senseImageUrl && !imageFailed ? <img src={value.senseImageUrl} alt={value.title} onError={() => setImageFailed(true)} /> : <span role="img" aria-label={`${value.title}图片暂缺`}>🔭<small>{value.title}</small></span>}</div>
    <div className="learn-copy"><p className="eyebrow">{value.moduleName}</p><h1>{value.title}</h1>{value.summary && <p className="concept-summary">{value.summary}</p>}{value.explanation && <p>{value.explanation}</p>}
      {value.funFact && <aside className="fun-fact"><b>你知道吗？</b>{value.funFact}</aside>}
      <div className="sound-actions">
        {value.speechUrl && <button className="secondary-button" onClick={() => audio.play(value.speechUrl!)}>🔊 听知识卡</button>}
      </div>{audio.state === 'error' && <p className="inline-note">声音暂时没准备好，可以先阅读知识卡。</p>}
      {value.hasPractice && <button className="primary-button" disabled={practice.isPending} onClick={() => practice.mutate(value.moduleCode)}>试一题</button>}
    </div>
  </section>
}
