import { appPath } from '../appPath'
import { Link } from 'react-router-dom'

const questionTypes = [
  { key: 'listen', title: '听音选字母' },
  { key: 'inword', title: '字中找拼音' },
  { key: 'shape', title: '看形认读' },
  { key: 'blend', title: '声韵拼读' },
] as const

export function HomePage() {
  return <section className="type-gallery-page" aria-label="题型">
    <div className="type-gallery">
      {questionTypes.map((type) => <Link className={`type-card type-card-${type.key}`} to={`/practice/type/${type.key}`} key={type.key} aria-label={type.title}>
        <img src={appPath(`/cards/${type.key}.png`)} alt="" />
        <strong>{type.title}</strong>
      </Link>)}
    </div>
  </section>
}
