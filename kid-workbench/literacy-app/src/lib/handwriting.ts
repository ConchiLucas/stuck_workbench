import HanziWriter from 'hanzi-writer'

export type InkStroke = Array<{ x: number; y: number }>

const GRID = 64
const MIN_STROKE_COVER = 0.52
const MIN_MEAN_COVER = 0.62
const MIN_PRECISION = 0.22

type Box = { x: number; y: number; w: number; h: number }

function bbox(points: Array<{ x: number; y: number }>): Box {
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const point of points) {
    minX = Math.min(minX, point.x)
    minY = Math.min(minY, point.y)
    maxX = Math.max(maxX, point.x)
    maxY = Math.max(maxY, point.y)
  }
  return { x: minX, y: minY, w: Math.max(1, maxX - minX), h: Math.max(1, maxY - minY) }
}

function normalize(strokes: InkStroke[]): InkStroke[] {
  const points = strokes.flat()
  if (!points.length) return []
  const box = bbox(points)
  const size = Math.max(box.w, box.h)
  const pad = size * 0.1
  const left = box.x - (size - box.w) / 2 - pad
  const top = box.y - (size - box.h) / 2 - pad
  const scale = size + pad * 2
  return strokes.map((stroke) => stroke.map((point) => ({
    x: (point.x - left) / scale,
    y: (point.y - top) / scale,
  })))
}

function stamp(grid: Uint8Array, x: number, y: number, radius: number) {
  const cx = Math.round(x * (GRID - 1))
  const cy = Math.round(y * (GRID - 1))
  const r = Math.max(1, Math.round(radius))
  for (let dy = -r; dy <= r; dy += 1) {
    for (let dx = -r; dx <= r; dx += 1) {
      if (dx * dx + dy * dy > r * r) continue
      const px = cx + dx
      const py = cy + dy
      if (px < 0 || py < 0 || px >= GRID || py >= GRID) continue
      grid[py * GRID + px] = 1
    }
  }
}

function rasterize(strokes: InkStroke[], radius: number) {
  const grid = new Uint8Array(GRID * GRID)
  for (const stroke of strokes) {
    if (!stroke.length) continue
    if (stroke.length === 1) {
      stamp(grid, stroke[0].x, stroke[0].y, radius)
      continue
    }
    for (let i = 1; i < stroke.length; i += 1) {
      const from = stroke[i - 1]
      const to = stroke[i]
      const dist = Math.hypot(to.x - from.x, to.y - from.y)
      const steps = Math.max(1, Math.ceil(dist * GRID * 2))
      for (let step = 0; step <= steps; step += 1) {
        const t = step / steps
        stamp(grid, from.x + (to.x - from.x) * t, from.y + (to.y - from.y) * t, radius)
      }
    }
  }
  return grid
}

function nearby(grid: Uint8Array, x: number, y: number, radius: number) {
  const cx = Math.round(x * (GRID - 1))
  const cy = Math.round(y * (GRID - 1))
  const r = Math.max(1, Math.round(radius))
  for (let dy = -r; dy <= r; dy += 1) {
    for (let dx = -r; dx <= r; dx += 1) {
      const px = cx + dx
      const py = cy + dy
      if (px < 0 || py < 0 || px >= GRID || py >= GRID) continue
      if (grid[py * GRID + px]) return true
    }
  }
  return false
}

function sampleStroke(stroke: InkStroke, count: number) {
  if (stroke.length === 1) return stroke
  const samples: InkStroke = []
  for (let i = 0; i < count; i += 1) {
    const t = i / (count - 1)
    const at = t * (stroke.length - 1)
    const index = Math.min(stroke.length - 2, Math.floor(at))
    const local = at - index
    const from = stroke[index]
    const to = stroke[index + 1]
    samples.push({
      x: from.x + (to.x - from.x) * local,
      y: from.y + (to.y - from.y) * local,
    })
  }
  return samples
}

function strokeCover(stroke: InkStroke, userInk: Uint8Array) {
  const samples = sampleStroke(stroke, 20)
  let hit = 0
  for (const point of samples) {
    if (nearby(userInk, point.x, point.y, 5)) hit += 1
  }
  return samples.length ? hit / samples.length : 0
}

function precision(userInk: Uint8Array, templateInk: Uint8Array) {
  let overlap = 0
  let sum = 0
  for (let i = 0; i < userInk.length; i += 1) {
    if (!userInk[i]) continue
    sum += 1
    overlap += templateInk[i]
  }
  return sum ? overlap / sum : 0
}

function toStrokes(medians: number[][][]): InkStroke[] {
  // Hanzi Writer 中线 y 向上，画板 y 向下；平移由 normalize 统一处理。
  return medians
    .map((stroke) => stroke.map(([x, y]) => ({ x, y: -y })))
    .filter((stroke) => stroke.length > 0)
}

export function inkMatch(user: InkStroke[], template: InkStroke[]) {
  const drawn = normalize(user.filter((stroke) => stroke.length > 0))
  const model = normalize(template.filter((stroke) => stroke.length > 0))
  if (!drawn.length || !model.length) return { ok: false, score: 0 }
  const userInk = rasterize(drawn, 3)
  const templateInk = rasterize(model, 4)
  const covers = model.map((stroke) => strokeCover(stroke, userInk))
  const minStroke = Math.min(...covers)
  const meanStroke = covers.reduce((sum, value) => sum + value, 0) / covers.length
  const neat = precision(userInk, templateInk)
  return {
    ok: minStroke >= MIN_STROKE_COVER && meanStroke >= MIN_MEAN_COVER && neat >= MIN_PRECISION,
    score: minStroke,
  }
}

export async function looksLikeCharacter(user: InkStroke[], character: string) {
  if (!user.some((stroke) => stroke.length > 1)) return { ok: false, score: 0 }
  const data = await HanziWriter.loadCharacterData(character)
  if (!data?.medians?.length) return { ok: false, score: 0 }
  return inkMatch(user, toStrokes(data.medians))
}
