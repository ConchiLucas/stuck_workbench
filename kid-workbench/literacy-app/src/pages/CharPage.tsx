import { Navigate, useParams } from 'react-router-dom'
import { CharStudio } from '../components/CharStudio'

function firstHanzi(raw: string) {
  return raw.match(/\p{Script=Han}/u)?.[0] ?? ''
}

export function CharPage() {
  const character = firstHanzi(useParams().character ?? '')
  if (!character) return <Navigate to="/char/山" replace />
  return <CharStudio character={character} />
}
