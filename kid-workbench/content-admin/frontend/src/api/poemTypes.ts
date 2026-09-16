export interface PoemLine {
  id: string
  ord: number
  text: string
}

export interface PoemItem {
  kpId: number
  code?: string
  workId?: string
  title: string
  author: string
  dynasty?: string
  edition?: string
  line1: string
  line2: string
  lines: string[]
  lineItems?: PoemLine[]
  speechOrds?: number[]
  difficulty: number
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
}

export interface PoemGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: PoemItem[]
}

export interface PoemListResult {
  view: string
  total: number
  groups?: PoemGroup[]
  items?: PoemItem[]
}

export interface PoemSyncResult {
  upserted: number
  total: number
}
