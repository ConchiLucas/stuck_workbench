import { appPath } from '../appPath'
import { Link } from 'react-router-dom'
import { phraseTypes } from '../content/phraseTypes'

export function HomePage() {
  return (
    <section className="type-gallery-page" aria-label="题型">
      <div className="type-gallery">
        {phraseTypes.map((type) => (
          <Link
            key={type.code}
            className={`type-card type-card-${type.code}`}
            to={`/practice/type/${type.code}`}
            aria-label={type.title}
          >
            <img src={appPath(`/cards/${type.code}.png`)} alt="" />
            <strong>{type.title}</strong>
          </Link>
        ))}
      </div>
    </section>
  )
}
