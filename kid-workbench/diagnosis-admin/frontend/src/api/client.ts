import { appPath } from '../appPath'
import type {
  ErrorPatternsResponse,
  KpArchive,
  Overview,
  SubjectDiagnosis,
} from './types'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof (body as { error?: string }).error === 'string'
      ? (body as { error: string }).error
      : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export function fetchOverview(childId: number): Promise<Overview> {
  return fetch(appPath(`/api/v1/children/${childId}/diagnosis/overview`)).then((res) => parseJSON(res))
}

export function fetchSubject(childId: number, code: string): Promise<SubjectDiagnosis> {
  return fetch(appPath(`/api/v1/children/${childId}/diagnosis/subjects/${code}`)).then((res) => parseJSON(res))
}

export function fetchKpArchive(childId: number, kpId: number): Promise<KpArchive> {
  return fetch(appPath(`/api/v1/children/${childId}/knowledge-points/${kpId}`)).then((res) => parseJSON(res))
}

export function fetchErrorPatterns(childId: number): Promise<ErrorPatternsResponse> {
  return fetch(appPath(`/api/v1/children/${childId}/error-patterns`)).then((res) => parseJSON(res))
}
