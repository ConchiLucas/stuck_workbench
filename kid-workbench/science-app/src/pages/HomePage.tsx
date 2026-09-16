import { appPath } from '../appPath'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { scienceApi } from '../api/science'
import { questionTypes } from '../data/questionTypes'
import { useChildStore } from '../store/childStore'
import { useDemoAnswerStore } from '../store/demoAnswerStore'
import { useLiveQuizStore } from '../store/liveQuizStore'

export function HomePage() {
  const childId = useChildStore((s) => s.childId)
  const home = useQuery({ queryKey: ['home', childId], queryFn: () => scienceApi.home(childId) })

  return <section className="type-gallery-page" aria-label="题型">
    {home.isError && <button className="retry-card" onClick={() => home.refetch()}>学习记录没加载出来，点这里再试一次</button>}
    <div className="type-gallery">
      {questionTypes.map((type) => <Link
        key={type.slug}
        to={`/question-types/${type.slug}`}
        className={`type-card type-card-${type.slug}`}
        aria-label={`查看题型：${type.title}`}
        onClick={() => {
          useDemoAnswerStore.getState().clearType(type.slug)
          if (type.slug === 'choice') useLiveQuizStore.getState().invalidate()
        }}
      >
        <img src={appPath(`/types/${type.slug}.png`)} alt="" />
        <strong>{type.title}</strong>
      </Link>)}
    </div>
  </section>
}
