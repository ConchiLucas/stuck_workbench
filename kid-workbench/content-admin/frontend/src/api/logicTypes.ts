export interface LogicItem {
  kpId: number
  title: string
  kind: string
  seq: string[]
  answer: string
  wrong: string[]
  prompt: string
  speech: string
  rule?: string
  example?: import('@kid-workbench/logic-player').LogicExample
  glyphUrls?: Record<string, string>
  difficulty: number
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
}

export interface LogicGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: LogicItem[]
}

export interface LogicListResult {
  view: string
  total: number
  groups?: LogicGroup[]
  items?: LogicItem[]
}
