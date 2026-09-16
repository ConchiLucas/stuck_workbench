import { appPath } from '../appPath'
import { Link } from 'react-router-dom'
import { isPracticeType, practiceHref, typeTitles, practiceTypes } from '../content/typePracticeBanks'
import { useLiveQuizStore } from '../store/liveQuizStore'
import { useTypePracticeStore } from '../store/typePracticeStore'

export function HomePage() {
  return (
    <section className="type-gallery-page" aria-label="题型">
      <div className="question-type-grid">
        {practiceTypes.map((type) => (
          <Link
            key={type}
            className={`question-type-card type-card-${type}`}
            to={practiceHref(type)}
            aria-label={typeTitles[type]}
            onClick={() => {
              if (!isPracticeType(type)) return
              useTypePracticeStore.getState().clearType(type)
              useLiveQuizStore.getState().invalidate(type)
            }}
          >
            <img src={appPath(`/cards/${type}.png`)} alt="" />
            <strong>{typeTitles[type]}</strong>
          </Link>
        ))}
      </div>
    </section>
  )
}
