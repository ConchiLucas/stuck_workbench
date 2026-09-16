import { gatewayLink } from '../appPath'
export const KID_APP_PORTS = {
  literacy: 19152,
  pinyin: 19112,
  math: 19142,
  english: 19132,
  science: 19122,
  poem: 19162,
  logic: 19192,
  chengyu: 19182,
  phrase: 19172,
} as const

export function kidAppHref(port: number) {
  const host = globalThis.location?.hostname || 'localhost'
  return gatewayLink(port, `http://${host}:${port}`)
}
