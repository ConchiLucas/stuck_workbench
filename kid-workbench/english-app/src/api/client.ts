export class APIError extends Error {
  constructor(public code: string, message: string, public status: number) {
    super(message)
  }
}

type Envelope<T> = { data: T | null; error: { code: string; message: string } | null }

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  let envelope: Envelope<T>
  try {
    envelope = await response.json() as Envelope<T>
  } catch {
    throw new APIError('invalid_response', '服务返回了无法识别的内容', response.status)
  }
  if (!response.ok || envelope.error) {
    throw new APIError(envelope.error?.code ?? 'request_failed', envelope.error?.message ?? '请求失败', response.status)
  }
  return envelope.data as T
}
