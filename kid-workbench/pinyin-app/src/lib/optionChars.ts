export function optionCharCount(label: string | undefined | null) {
  return Math.max(1, Array.from(label ?? '').length)
}
