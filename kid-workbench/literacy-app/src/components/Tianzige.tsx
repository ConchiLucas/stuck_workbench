import type { ReactNode } from 'react'

export function Tianzige({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <div className={`tianzige ${className}`.trim()}>{children}</div>
}
