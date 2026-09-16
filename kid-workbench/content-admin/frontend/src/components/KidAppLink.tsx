import { ExternalLink } from 'lucide-react'
import { kidAppHref } from '../content/kidApps'

export function KidAppLink({ port }: { port: number }) {
  return (
    <a
      className="kid-app-link"
      href={kidAppHref(port)}
      target="_blank"
      rel="noopener noreferrer"
    >
      <ExternalLink size={14} aria-hidden="true" />
      孩子端
    </a>
  )
}
