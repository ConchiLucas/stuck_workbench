export const demoTypes = ['listen', 'inword', 'shape', 'blend'] as const
export type DemoType = (typeof demoTypes)[number]

export const questionTypes: Array<{ key: DemoType; title: string }> = [
  { key: 'listen', title: '听音选字母' },
  { key: 'inword', title: '字中找拼音' },
  { key: 'shape', title: '看形认读' },
  { key: 'blend', title: '声韵拼读' },
]

export function isDemoType(value: string | undefined): value is DemoType {
  return demoTypes.includes(value as DemoType)
}

export function typeTitle(type: DemoType) {
  return questionTypes.find((item) => item.key === type)?.title ?? type
}

export { practiceStem } from '@kid-workbench/pinyin-player'

export function demoHref(type: DemoType, n: number) {
  return `/practice/type/${type}/${Math.max(1, n)}`
}

export function demoResultHref(type: DemoType) {
  return `/practice/type/${type}/result`
}
