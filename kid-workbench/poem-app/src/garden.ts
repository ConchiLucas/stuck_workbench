export type PavilionCode = 'moon' | 'dawn' | 'goose' | 'farm' | 'tower' | 'snow' | 'scroll'

export type PoemOption = { id: string; label: string; visual: string }

export type ClueKind = 'choice' | 'title' | 'author' | 'fill' | 'couplet' | 'recite'

export type Clue = {
  id: string
  kind: ClueKind
  prompt: string
  line: string
  speech: string
  answerId: string
  options: PoemOption[]
  sequence?: string[]
  workId?: string
  lineId?: string
  sourceLine?: string
  gapIndexes?: number[]
  upperLineId?: string
  nextLineId?: string
  speechUrl?: string
  audioMissingReason?: string
}

export type Pavilion = {
  code: PavilionCode
  title: string
  kidTitle: string
  kind: string
  orderNo: number
  line: string
  clues: Clue[]
}

const opt = (id: string, label: string, visual: string): PoemOption => ({ id, label, visual })

function choice(id: string, prompt: string, line: string, answer: string, options: PoemOption[]): Clue {
  return { id, kind: 'choice', prompt, line, speech: line, answerId: answer, options }
}

function title(id: string, prompt: string, line: string, answer: string, options: PoemOption[]): Clue {
  return { id, kind: 'title', prompt, line, speech: line, answerId: answer, options }
}

function author(id: string, prompt: string, line: string, answer: string, options: PoemOption[]): Clue {
  return { id, kind: 'author', prompt, line, speech: line, answerId: answer, options }
}

function fill(id: string, prompt: string, line: string, answer: string, options: PoemOption[]): Clue {
  return { id, kind: 'fill', prompt, line, speech: line, answerId: answer, options }
}

function couplet(id: string, prompt: string, line: string, answer: string, options: PoemOption[]): Clue {
  return { id, kind: 'couplet', prompt, line, speech: line, answerId: answer, options }
}

export function taskLabel(kind: ClueKind) {
  switch (kind) {
    case 'choice': return '选图'
    case 'title': return '选诗名'
    case 'author': return '选作者'
    case 'fill': return '补字'
    case 'couplet': return '选下一句'
    case 'recite': return '排顺序'
  }
}

export const pavilions: Pavilion[] = [
  {
    code: 'moon', title: '月下亭', kidTitle: '望月', kind: 'choice', orderNo: 1, line: '床前明月光',
    clues: [
      choice('moon-light', '诗句里写了什么？', '床前明月光', 'moon', [
        opt('sun', '太阳', 'sun'), opt('lamp', '灯', 'lamp'), opt('moon', '月亮', 'moon'), opt('star', '星星', 'star'),
      ]),
      choice('moon-frost', '地上像什么？', '疑是地上霜', 'frost', [
        opt('frost', '霜', 'frost'), opt('snow', '雪', 'snow'), opt('flower', '花', 'flower'), opt('rain', '雨', 'rain'),
      ]),
      title('moon-title', '这首诗叫什么？', '床前明月光', 'jingyesi', [
        opt('chunxiao', '春晓', 'char'), opt('yonge', '咏鹅', 'char'), opt('jingyesi', '静夜思', 'char'), opt('minnong', '悯农', 'char'),
      ]),
      author('moon-author', '这首诗是谁写的？', '床前明月光', 'libai', [
        opt('dufu', '杜甫', 'char'), opt('libai', '李白', 'char'), opt('wangwei', '王维', 'char'), opt('menghaoran', '孟浩然', 'char'),
      ]),
    ],
  },
  {
    code: 'dawn', title: '春晓径', kidTitle: '闻鸟', kind: 'choice', orderNo: 2, line: '处处闻啼鸟',
    clues: [
      choice('dawn-bird', '诗句里听见了什么？', '处处闻啼鸟', 'bird', [
        opt('bell', '钟', 'bell'), opt('horse', '马', 'horse'), opt('bird', '鸟', 'bird'), opt('drum', '鼓', 'drum'),
      ]),
      choice('dawn-rain', '夜里是什么声音？', '夜来风雨声', 'rain', [
        opt('rain', '风雨', 'rain'), opt('song', '踏歌', 'song'), opt('drum', '战鼓', 'drum'), opt('bell', '钟', 'bell'),
      ]),
      title('dawn-title', '这首诗叫什么？', '春眠不觉晓', 'chunxiao', [
        opt('jingyesi', '静夜思', 'char'), opt('chunxiao', '春晓', 'char'), opt('jiangxue', '江雪', 'char'), opt('yonge', '咏鹅', 'char'),
      ]),
      author('dawn-author', '这首诗是谁写的？', '春眠不觉晓', 'menghaoran', [
        opt('libai', '李白', 'char'), opt('wangzhihuan', '王之涣', 'char'), opt('menghaoran', '孟浩然', 'char'), opt('liuzongyuan', '柳宗元', 'char'),
      ]),
    ],
  },
  {
    code: 'goose', title: '白鹅池', kidTitle: '诵句', kind: 'recite', orderNo: 3, line: '鹅鹅鹅',
    clues: [
      {
        id: 'goose-recite', kind: 'recite', prompt: '按顺序点出这四句', line: '鹅鹅鹅', speech: '鹅鹅鹅，曲项向天歌，白毛浮绿水，红掌拨清波',
        answerId: 'done', sequence: ['鹅鹅鹅', '曲项向天歌', '白毛浮绿水', '红掌拨清波'],
        options: [
          opt('鹅鹅鹅', '鹅鹅鹅', 'line'), opt('曲项向天歌', '曲项向天歌', 'line'),
          opt('红掌拨清波', '红掌拨清波', 'line'), opt('白毛浮绿水', '白毛浮绿水', 'line'),
        ],
      },
      couplet('goose-next', '下一句是哪一句？', '鹅鹅鹅', '曲项向天歌', [
        opt('曲项向天歌', '曲项向天歌', 'line'), opt('白毛浮绿水', '白毛浮绿水', 'line'),
        opt('床前明月光', '床前明月光', 'line'), opt('红掌拨清波', '红掌拨清波', 'line'),
      ]),
      fill('goose-water', '缺的字是哪个？', '白毛浮绿□', '水', [
        opt('山', '山', 'char'), opt('水', '水', 'char'), opt('天', '天', 'char'), opt('月', '月', 'char'),
      ]),
      choice('goose-palm', '红掌是什么颜色？', '红掌拨清波', 'red', [
        opt('green', '绿', 'green'), opt('red', '红', 'red'), opt('blue', '蓝', 'blue'), opt('white', '白', 'white'),
      ]),
    ],
  },
  {
    code: 'farm', title: '田园井', kidTitle: '补字', kind: 'fill', orderNo: 4, line: '锄禾日当午',
    clues: [
      fill('farm-noon', '缺的字是哪个？', '锄禾日当□', '午', [
        opt('天', '天', 'char'), opt('午', '午', 'char'), opt('月', '月', 'char'), opt('鹅', '鹅', 'char'),
      ]),
      fill('farm-soil', '缺的字是哪个？', '汗滴禾下□', '土', [
        opt('土', '土', 'char'), opt('水', '水', 'char'), opt('火', '火', 'char'), opt('石', '石', 'char'),
      ]),
      fill('farm-meal', '缺的字是哪个？', '谁知盘中□', '餐', [
        opt('饭', '饭', 'char'), opt('菜', '菜', 'char'), opt('餐', '餐', 'char'), opt('茶', '茶', 'char'),
      ]),
      title('farm-title', '这首诗叫什么？', '粒粒皆辛苦', 'minnong', [
        opt('chunxiao', '春晓', 'char'), opt('yonge', '咏鹅', 'char'), opt('minnong', '悯农', 'char'), opt('jiangxue', '江雪', 'char'),
      ]),
    ],
  },
  {
    code: 'tower', title: '鹳楼台', kidTitle: '对句', kind: 'couplet', orderNo: 5, line: '欲穷千里目',
    clues: [
      couplet('tower-sun', '下一句是哪一句？', '白日依山尽', '黄河入海流', [
        opt('黄河入海流', '黄河入海流', 'line'), opt('处处闻啼鸟', '处处闻啼鸟', 'line'),
        opt('疑是地上霜', '疑是地上霜', 'line'), opt('红掌拨清波', '红掌拨清波', 'line'),
      ]),
      couplet('tower-climb', '下一句是哪一句？', '欲穷千里目', '更上一层楼', [
        opt('低头思故乡', '低头思故乡', 'line'), opt('更上一层楼', '更上一层楼', 'line'),
        opt('花落知多少', '花落知多少', 'line'), opt('春风吹又生', '春风吹又生', 'line'),
      ]),
      choice('tower-river', '入海的是哪条河？', '黄河入海流', 'yellow-river', [
        opt('yangtze', '长江', 'yangtze'), opt('yellow-river', '黄河', 'yellow-river'), opt('pond', '小池', 'pond'), opt('well', '水井', 'well'),
      ]),
      title('tower-title', '这首诗叫什么？', '欲穷千里目', 'dengguanquelou', [
        opt('jingyesi', '静夜思', 'char'), opt('chunxiao', '春晓', 'char'), opt('dengguanquelou', '登鹳雀楼', 'char'), opt('hua', '画', 'char'),
      ]),
    ],
  },
  {
    code: 'snow', title: '寒江矶', kidTitle: '寻影', kind: 'choice', orderNo: 6, line: '独钓寒江雪',
    clues: [
      choice('snow-bird', '这句里的鸟怎么样了？', '千山鸟飞绝', 'gone', [
        opt('many', '很多鸟', 'bird'), opt('goose', '有一只鹅', 'goose'), opt('gone', '飞光了', 'gone'), opt('cicada', '有蝉', 'cicada'),
      ]),
      choice('snow-boat', '江上有什么？', '孤舟蓑笠翁', 'boat', [
        opt('boat', '孤舟', 'boat'), opt('ship', '大船', 'ship'), opt('cart', '马车', 'cart'), opt('kite', '纸鸢', 'kite'),
      ]),
      choice('snow-weather', '下着什么？', '独钓寒江雪', 'snow', [
        opt('rain', '雨', 'rain'), opt('sun', '日', 'sun'), opt('snow', '雪', 'snow'), opt('flower', '花', 'flower'),
      ]),
      title('snow-title', '这首诗叫什么？', '千山鸟飞绝', 'jiangxue', [
        opt('yonge', '咏鹅', 'char'), opt('chunxiao', '春晓', 'char'), opt('minnong', '悯农', 'char'), opt('jiangxue', '江雪', 'char'),
      ]),
    ],
  },
  {
    code: 'scroll', title: '画中卷', kidTitle: '读画', kind: 'choice', orderNo: 7, line: '近听水无声',
    clues: [
      choice('scroll-mountain', '远看的山怎么样？', '远看山有色', 'color', [
        opt('silent', '没有声音', 'silent'), opt('color', '有颜色', 'color'), opt('gone', '看不见', 'gone'), opt('snow', '全是雪', 'snow'),
      ]),
      choice('scroll-water', '近处听水，水怎么样？', '近听水无声', 'silent', [
        opt('loud', '很响', 'loud'), opt('sing', '在唱歌', 'sing'), opt('silent', '没有声音', 'silent'), opt('hot', '是热水', 'hot'),
      ]),
      title('scroll-title', '这首诗叫什么？', '远看山有色', 'hua', [
        opt('jiangxue', '江雪', 'char'), opt('hua', '画', 'char'), opt('chunxiao', '春晓', 'char'), opt('yonge', '咏鹅', 'char'),
      ]),
      choice('scroll-bird', '人走过来，画里的鸟会怎样？', '人来鸟不惊', 'still', [
        opt('fly', '飞走', 'fly'), opt('sing', '唱歌', 'sing'), opt('swim', '游泳', 'swim'), opt('still', '还停着', 'still'),
      ]),
    ],
  },
]

export function pavilionByCode(code?: string) {
  return pavilions.find((row) => row.code === code)
}

export function clueHref(code: PavilionCode, n = 1) {
  return n <= 1 ? `/pavilions/${code}` : `/pavilions/${code}/${n}`
}

export function nextHref(pavilion: Pavilion, n: number) {
  return n >= pavilion.clues.length ? `/pavilions/${pavilion.code}/result` : clueHref(pavilion.code, n + 1)
}

export function prevHref(pavilion: Pavilion, n: number) {
  if (n <= 1) return undefined
  return clueHref(pavilion.code, n - 1)
}

export function pickKey(code: string, n: number) {
  return `${code}:${n}`
}

export const practiceTypes = ['title', 'fill', 'couplet', 'recite'] as const
export type PracticeType = (typeof practiceTypes)[number]

export const typeTitles: Record<PracticeType, string> = {
  title: '选诗名',
  fill: '补字',
  couplet: '选下一句',
  recite: '排顺序',
}

function clueOf(code: PavilionCode, id: string) {
  const clue = pavilionByCode(code)?.clues.find((row) => row.id === id)
  if (!clue) throw new Error(`missing clue ${code}/${id}`)
  return clue
}

function recitePoem(id: string, workId: string, line: string, lines: string[]): Clue {
  const ids = lines.map((_, i) => `${workId}:L${i + 1}`)
  const display = [...ids]
  if (display.length > 1) {
    const last = display.length - 1
    ;[display[0], display[last]] = [display[last], display[0]]
  }
  const byID = Object.fromEntries(ids.map((lineId, i) => [lineId, lines[i]]))
  return {
    id,
    kind: 'recite',
    prompt: `按顺序点出这${lines.length}句`,
    line,
    speech: '',
    answerId: ids.join(','),
    sequence: ids,
    workId,
    sourceLine: lines.join(''),
    audioMissingReason: '排顺序不按正确句序朗读，以免泄露答案。',
    options: display.map((lineId) => opt(lineId, byID[lineId], 'line')),
  }
}

function withWork(clue: Clue, extra: Partial<Clue>): Clue {
  return { ...clue, ...extra }
}

export const typeBanks: Record<PracticeType, Clue[]> = {
  title: [
    withWork(clueOf('moon', 'moon-title'), {
      workId: 'pm001', answerId: 'pm001', lineId: 'pm001:L1', sourceLine: '床前明月光',
      options: [opt('pm002', '春晓', 'char'), opt('pm003', '咏鹅', 'char'), opt('pm001', '静夜思', 'char'), opt('pm004', '悯农', 'char')],
    }),
    withWork(clueOf('dawn', 'dawn-title'), {
      workId: 'pm002', answerId: 'pm002', lineId: 'pm002:L1', sourceLine: '春眠不觉晓',
      options: [opt('pm001', '静夜思', 'char'), opt('pm002', '春晓', 'char'), opt('pm006', '江雪', 'char'), opt('pm003', '咏鹅', 'char')],
    }),
    withWork(clueOf('farm', 'farm-title'), {
      workId: 'pm004', answerId: 'pm004', lineId: 'pm004:L4', sourceLine: '粒粒皆辛苦',
      options: [opt('pm002', '春晓', 'char'), opt('pm003', '咏鹅', 'char'), opt('pm004', '悯农', 'char'), opt('pm006', '江雪', 'char')],
    }),
    withWork(clueOf('tower', 'tower-title'), {
      workId: 'pm005', answerId: 'pm005', lineId: 'pm005:L3', sourceLine: '欲穷千里目',
      options: [opt('pm001', '静夜思', 'char'), opt('pm002', '春晓', 'char'), opt('pm005', '登鹳雀楼', 'char'), opt('pm028', '画', 'char')],
    }),
  ],
  fill: [
    withWork(clueOf('farm', 'farm-noon'), {
      workId: 'pm004', lineId: 'pm004:L1', sourceLine: '锄禾日当午', gapIndexes: [4], answerId: 'char:午#4',
      audioMissingReason: '补字题不朗读缺字，以免直接说出答案。', speech: '',
      options: [opt('char:天', '天', 'char'), opt('char:午#4', '午', 'char'), opt('char:月', '月', 'char'), opt('char:鹅', '鹅', 'char')],
    }),
    withWork(clueOf('farm', 'farm-soil'), {
      workId: 'pm004', lineId: 'pm004:L2', sourceLine: '汗滴禾下土', gapIndexes: [4], answerId: 'char:土#4',
      audioMissingReason: '补字题不朗读缺字，以免直接说出答案。', speech: '',
      options: [opt('char:土#4', '土', 'char'), opt('char:水', '水', 'char'), opt('char:火', '火', 'char'), opt('char:石', '石', 'char')],
    }),
    withWork(clueOf('farm', 'farm-meal'), {
      workId: 'pm004', lineId: 'pm004:L3', sourceLine: '谁知盘中餐', gapIndexes: [4], answerId: 'char:餐#4',
      audioMissingReason: '补字题不朗读缺字，以免直接说出答案。', speech: '',
      options: [opt('char:饭', '饭', 'char'), opt('char:菜', '菜', 'char'), opt('char:餐#4', '餐', 'char'), opt('char:茶', '茶', 'char')],
    }),
    withWork(clueOf('goose', 'goose-water'), {
      workId: 'pm003', lineId: 'pm003:L3', sourceLine: '白毛浮绿水', gapIndexes: [4], answerId: 'char:水#4',
      audioMissingReason: '补字题不朗读缺字，以免直接说出答案。', speech: '',
      options: [opt('char:山', '山', 'char'), opt('char:水#4', '水', 'char'), opt('char:天', '天', 'char'), opt('char:月', '月', 'char')],
    }),
  ],
  couplet: [
    withWork(clueOf('tower', 'tower-sun'), {
      workId: 'pm005', lineId: 'pm005:L1', sourceLine: '白日依山尽', answerId: 'pm005:L2',
      upperLineId: 'pm005:L1', nextLineId: 'pm005:L2',
      options: [opt('pm005:L2', '黄河入海流', 'line'), opt('pm002:L2', '处处闻啼鸟', 'line'), opt('pm001:L2', '疑是地上霜', 'line'), opt('pm003:L4', '红掌拨清波', 'line')],
    }),
    withWork(clueOf('tower', 'tower-climb'), {
      workId: 'pm005', lineId: 'pm005:L3', sourceLine: '欲穷千里目', answerId: 'pm005:L4',
      upperLineId: 'pm005:L3', nextLineId: 'pm005:L4',
      options: [opt('pm001:L4', '低头思故乡', 'line'), opt('pm005:L4', '更上一层楼', 'line'), opt('pm002:L4', '花落知多少', 'line'), opt('pm012:L4', '春风吹又生', 'line')],
    }),
    withWork(clueOf('goose', 'goose-next'), {
      workId: 'pm003', lineId: 'pm003:L1', sourceLine: '鹅鹅鹅', answerId: 'pm003:L2',
      upperLineId: 'pm003:L1', nextLineId: 'pm003:L2',
      options: [opt('pm003:L2', '曲项向天歌', 'line'), opt('pm003:L3', '白毛浮绿水', 'line'), opt('pm001:L1', '床前明月光', 'line'), opt('pm003:L4', '红掌拨清波', 'line')],
    }),
    withWork(couplet('jingye-next', '下一句是哪一句？', '举头望明月', 'pm001:L4', [
      opt('pm001:L4', '低头思故乡', 'line'),
      opt('pm001:L2', '疑是地上霜', 'line'),
      opt('pm002:L2', '处处闻啼鸟', 'line'),
      opt('pm005:L2', '黄河入海流', 'line'),
    ]), { workId: 'pm001', lineId: 'pm001:L3', sourceLine: '举头望明月', upperLineId: 'pm001:L3', nextLineId: 'pm001:L4' }),
  ],
  recite: [
    recitePoem('jingye-recite', 'pm001', '床前明月光', ['床前明月光', '疑是地上霜', '举头望明月', '低头思故乡']),
    recitePoem('yonge-recite', 'pm003', '鹅鹅鹅', ['鹅鹅鹅', '曲项向天歌', '白毛浮绿水', '红掌拨清波']),
    recitePoem('chunxiao-recite', 'pm002', '春眠不觉晓', ['春眠不觉晓', '处处闻啼鸟', '夜来风雨声', '花落知多少']),
    recitePoem('tower-recite', 'pm005', '白日依山尽', ['白日依山尽', '黄河入海流', '欲穷千里目', '更上一层楼']),
  ],
}

export function isPracticeType(value: string | undefined): value is PracticeType {
  return practiceTypes.includes(value as PracticeType)
}

export function typeHref(type: PracticeType, n = 1) {
  return n <= 1 ? `/practice/type/${type}` : `/practice/type/${type}/${n}`
}

export function typeResultHref(type: PracticeType) {
  return `/practice/type/${type}/result`
}

export function typeNextHref(type: PracticeType, n: number) {
  const total = typeBanks[type].length
  return n >= total ? typeResultHref(type) : typeHref(type, n + 1)
}

export function typePrevHref(type: PracticeType, n: number) {
  if (n <= 1) return undefined
  return typeHref(type, n - 1)
}

export function clueExample(clue: Clue) {
  const gaps = clue.gapIndexes
  const source = clue.sourceLine || clue.line.replaceAll('□', '')
  return {
    kind: clue.kind as PracticeType,
    prompt: clue.prompt,
    line: clue.line,
    speechUrl: clue.speechUrl,
    speechText: clue.speech,
    audioMissingReason: clue.audioMissingReason || (clue.kind === 'title' || clue.kind === 'couplet' ? '读音素材暂不可用' : clue.kind === 'fill' ? '补字题不朗读缺字，以免直接说出答案。' : '排顺序不按正确句序朗读，以免泄露答案。'),
    options: clue.options.map((o) => ({ id: o.id, label: o.label })),
    answerId: clue.kind === 'recite' ? undefined : clue.answerId,
    workId: clue.workId,
    lineId: clue.lineId,
    sourceLine: source,
    gapIndexes: gaps,
    sequenceItems: clue.kind === 'recite' ? clue.options.map((o) => ({ id: o.id, label: o.label })) : undefined,
    sequenceDisplayOrder: clue.kind === 'recite' ? clue.options.map((o) => o.id) : undefined,
    correctSequence: clue.kind === 'recite' ? clue.sequence : undefined,
    upperLineId: clue.upperLineId,
    nextLineId: clue.nextLineId,
  }
}

