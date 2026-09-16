import { appPath } from '../appPath'
type Envelope<T> = { data: T | null; error: { code: string; message: string } | null }

export class ApiError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(appPath(`/api/v1${path}`), {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  let envelope: Envelope<T>
  try {
    envelope = await response.json() as Envelope<T>
  } catch {
    throw new ApiError(response.status, 'invalid_response', '服务返回了无法识别的内容')
  }
  if (!response.ok || envelope.error) {
    throw new ApiError(response.status, envelope.error?.code ?? 'request_failed', envelope.error?.message ?? '请求失败')
  }
  return envelope.data as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) => request<T>(path, {
    method: 'POST', body: body === undefined ? undefined : JSON.stringify(body),
  }),
}
