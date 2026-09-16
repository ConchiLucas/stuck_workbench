import type { ReactNode } from 'react'

export function FourLineGrid({ children }: { children: ReactNode }) {
  return <div className="four-lines"><span>{children}</span></div>
}
