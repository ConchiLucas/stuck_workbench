export const CHILD_ID_KEY = 'diagnosis-child-id'

export function getChildId(): number {
  const fromQuery = Number(new URLSearchParams(window.location.search).get('child'))
  if (Number.isFinite(fromQuery) && fromQuery > 0) {
    localStorage.setItem(CHILD_ID_KEY, String(fromQuery))
    return fromQuery
  }
  const raw = localStorage.getItem(CHILD_ID_KEY)
  const n = raw ? Number(raw) : 1
  return Number.isFinite(n) && n > 0 ? n : 1
}

export function setChildId(id: number) {
  localStorage.setItem(CHILD_ID_KEY, String(id))
}
