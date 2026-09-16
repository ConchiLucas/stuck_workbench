const stemByCode: Record<string, string> = {
  glyph_sense: '看这个字，选出它的意思',
  sense_char: '这幅意思对应哪个字？',
  write_char: '先听读音，再写到格子里',
}

export function practiceStem(code: string, fallback: string) {
  if (stemByCode[code]) return stemByCode[code]
  return fallback.replace(/（暂用字卡）/g, '').trim()
}
