import { useQuery } from '@tanstack/react-query'
import type { MathCatalog } from '@kid-workbench/math-player'
import { api } from './client'

export async function getPublishedMathDetails(): Promise<MathCatalog> {
  const catalog = await api.get<MathCatalog>('/math/details')
  if (!catalog || catalog.schemaVersion !== 1 || !Array.isArray(catalog.items) || catalog.items.some(item => !item.id || !Number.isInteger(item.revision) || !item.example || typeof item.example.prompt !== 'string')) {
    throw new Error('素材目录格式或版本不受支持')
  }
  return catalog
}
export const usePublishedMathDetails=()=>useQuery({queryKey:['math','published-details'],queryFn:getPublishedMathDetails,staleTime:0,retry:false})
