import clsx from 'clsx'

export interface SubjectMetaItem {
  code: string
  name: string
  pct: number
}

export const ORDERED_SUBJECTS: SubjectMetaItem[] = [
  { code: 'literacy', name: '识字', pct: 78 },
  { code: 'pinyin', name: '拼音', pct: 62 },
  { code: 'math', name: '算数', pct: 40 },
  { code: 'english', name: '英语', pct: 56 },
  { code: 'science', name: '科普', pct: 48 },
  { code: 'poem', name: '古诗', pct: 72 },
  { code: 'logic', name: '逻辑', pct: 36 },
  { code: 'chengyu', name: '成语', pct: 68 },
  { code: 'phrase', name: '英语短句', pct: 54 },
]

export function SubjectBadge({
  code,
  className,
  size = 'md',
}: {
  code: string
  className?: string
  size?: 'sm' | 'md' | 'lg'
}) {
  const sizeClasses = {
    sm: 'h-5 w-5 rounded text-[10px]',
    md: 'h-6 w-6 rounded-md text-xs',
    lg: 'h-7 w-7 rounded-lg text-sm',
  }[size]

  switch (code) {
    case 'literacy':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#ff4e88]/15 border border-[#ff4e88]/30 text-[#ff659c]',
            sizeClasses,
            className
          )}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" />
            <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" />
          </svg>
        </span>
      )
    case 'pinyin':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#3b82f6]/15 border border-[#3b82f6]/30 text-[#60a5fa] font-bold tracking-tighter leading-none',
            sizeClasses,
            className
          )}
        >
          <span className="text-[10px] font-black -translate-y-px">abc</span>
        </span>
      )
    case 'math':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#8b5cf6]/15 border border-[#8b5cf6]/30 text-[#a78bfa] font-bold tracking-tighter leading-none',
            sizeClasses,
            className
          )}
        >
          <span className="text-[10px] font-black -translate-y-px">123</span>
        </span>
      )
    case 'english':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#06b6d4]/15 border border-[#06b6d4]/30 text-[#38bdf8]',
            sizeClasses,
            className
          )}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <circle cx="12" cy="12" r="9" />
            <path d="M3 12h18" />
            <path d="M12 3a14 14 0 0 1 3.5 9 14 14 0 0 1-3.5 9 14 14 0 0 1-3.5-9 14 14 0 0 1 3.5-9z" />
          </svg>
        </span>
      )
    case 'science':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#10b981]/15 border border-[#10b981]/30 text-[#34d399]',
            sizeClasses,
            className
          )}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <path d="M12 21V10" />
            <path d="M12 10a5 5 0 0 1 5-5c0 4-2 7-5 7z" />
            <path d="M12 14a5 5 0 0 0-5-5c0 4 2 7 5 7z" />
          </svg>
        </span>
      )
    case 'poem':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#f43f5e]/15 border border-[#f43f5e]/30 text-[#fb7185]',
            sizeClasses,
            className
          )}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <circle cx="12" cy="12" r="3" fill="currentColor" fillOpacity="0.3" />
            <path d="M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M5.6 18.4l2.1-2.1M16.3 7.7l2.1-2.1" />
          </svg>
        </span>
      )
    case 'logic':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#14b8a6]/15 border border-[#14b8a6]/30 text-[#2dd4bf]',
            sizeClasses,
            className
          )}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <path d="M4 11h2a2 2 0 0 0 2-2 2 2 0 0 1 4 0 2 2 0 0 0 2 2h2v4h-2a2 2 0 0 0-2 2 2 2 0 0 1-4 0 2 2 0 0 0-2-2H4z" />
            <rect x="3" y="3" width="18" height="18" rx="2" />
          </svg>
        </span>
      )
    case 'chengyu':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#f59e0b]/15 border border-[#f59e0b]/30 text-[#fbbf24]',
            sizeClasses,
            className
          )}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <rect x="4" y="3" width="16" height="18" rx="2" />
            <path d="M8 7h8M8 11h8M8 15h5" />
          </svg>
        </span>
      )
    case 'phrase':
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-[#c084fc]/15 border border-[#c084fc]/30 text-[#e879f9]',
            sizeClasses,
            className
          )}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
          </svg>
        </span>
      )
    default:
      return (
        <span
          className={clsx(
            'grid shrink-0 place-items-center bg-white/10 text-white rounded-md',
            sizeClasses,
            className
          )}
        >
          •
        </span>
      )
  }
}

export function SubjectOutlineIcon({
  code,
  className = 'h-4 w-4 text-[#c084fc]',
}: {
  code: string
  className?: string
}) {
  switch (code) {
    case 'literacy':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" />
          <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" />
        </svg>
      )
    case 'pinyin':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <path d="M3 18v-6a9 9 0 0 1 18 0v6" />
          <path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z" />
        </svg>
      )
    case 'math':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <rect x="4" y="3" width="16" height="18" rx="2" />
          <line x1="8" y1="7" x2="16" y2="7" />
          <path d="M8 11h.01M12 11h.01M16 11h.01M8 15h.01M12 15h.01M16 15h.01" />
        </svg>
      )
    case 'english':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <circle cx="12" cy="12" r="9" />
          <path d="M3 12h18" />
          <path d="M12 3a14 14 0 0 1 3.5 9 14 14 0 0 1-3.5 9 14 14 0 0 1-3.5-9 14 14 0 0 1 3.5-9z" />
        </svg>
      )
    case 'science':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <path d="M10 2v7.527a2 2 0 0 1-.211.896L4.72 20.55a1 1 0 0 0 .9 1.45h12.76a1 1 0 0 0 .9-1.45l-5.069-10.127A2 2 0 0 1 14 9.527V2" />
          <path d="M8.5 2h7" />
          <path d="M7 16h10" />
        </svg>
      )
    case 'poem':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <path d="M19 17h2c.6 0 1-.4 1-1v-3c0-.9-.7-1.7-1.5-1.9C18.7 10.6 16 10 16 10s-1.3-1.4-2.2-2.3c-.5-.4-1.1-.7-1.8-.7H5c-.6 0-1 .4-1 1v7c0 .6.4 1 1 1h14" />
          <path d="M19 17v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-4" />
        </svg>
      )
    case 'logic':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <path d="M4 11h2a2 2 0 0 0 2-2 2 2 0 0 1 4 0 2 2 0 0 0 2 2h2v4h-2a2 2 0 0 0-2 2 2 2 0 0 1-4 0 2 2 0 0 0-2-2H4z" />
          <rect x="3" y="3" width="18" height="18" rx="2" />
        </svg>
      )
    case 'chengyu':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <rect x="4" y="3" width="16" height="18" rx="2" />
          <path d="M8 7h8M8 11h8M8 15h5" />
        </svg>
      )
    case 'phrase':
      return (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
          <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
          <circle cx="9" cy="11.5" r="1" fill="currentColor" />
          <circle cx="12" cy="11.5" r="1" fill="currentColor" />
          <circle cx="15" cy="11.5" r="1" fill="currentColor" />
        </svg>
      )
    default:
      return (
        <span className="text-xs text-[#c084fc]">•</span>
      )
  }
}
