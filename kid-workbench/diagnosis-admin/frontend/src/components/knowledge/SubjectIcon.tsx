import { AudioLines, BookOpen, Calculator, Feather, Languages, MessagesSquare, Puzzle, ScrollText, Sprout } from 'lucide-react'

const icons = { literacy: BookOpen, pinyin: AudioLines, math: Calculator, english: Languages, science: Sprout, poem: Feather, logic: Puzzle, chengyu: ScrollText, phrase: MessagesSquare }
export function SubjectIcon({ code }: { code: string }) {
  const Icon = icons[code as keyof typeof icons] ?? BookOpen
  return <Icon size={18} strokeWidth={1.7} aria-hidden="true" />
}
