import type { ReactNode } from 'react'

export function FourLineGrid({ children, compact = false }: { children: ReactNode; compact?: boolean }) {
  return <div className={`four-lines${compact ? ' compact' : ''}`}><span>{children}</span></div>
}
