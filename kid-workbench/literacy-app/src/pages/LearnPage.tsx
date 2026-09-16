import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { literacyApi } from '../api/literacy'
import { CharStudio } from '../components/CharStudio'

export function LearnPage() {
  const kpId = Number(useParams().kpId)
  const item = useQuery({ queryKey: ['item', kpId], queryFn: () => literacyApi.item(kpId) })
  if (item.isLoading) return <div className="skeleton-block">这个字正在走过来……</div>
  if (!item.data) return <button className="retry-card" onClick={() => item.refetch()}>这个字还没准备好，再试一次</button>
  return <CharStudio character={item.data.character} kpId={kpId} />
}
