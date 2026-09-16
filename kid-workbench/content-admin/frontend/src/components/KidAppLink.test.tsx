import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { KidAppLink } from './KidAppLink'
import { KID_APP_PORTS, kidAppHref } from '../content/kidApps'

describe('KidAppLink', () => {
  it('opens the literacy kid app on the screenshot port', () => {
    render(<KidAppLink port={KID_APP_PORTS.literacy} />)
    const link = screen.getByRole('link', { name: '孩子端' })
    expect(link).toHaveAttribute('href', 'http://localhost:19152')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', expect.stringContaining('noopener'))
  })

  it('maps every subject to the kid-app port from the screenshot', () => {
    expect(kidAppHref(KID_APP_PORTS.literacy)).toBe('http://localhost:19152')
    expect(kidAppHref(KID_APP_PORTS.pinyin)).toBe('http://localhost:19112')
    expect(kidAppHref(KID_APP_PORTS.math)).toBe('http://localhost:19142')
    expect(kidAppHref(KID_APP_PORTS.english)).toBe('http://localhost:19132')
    expect(kidAppHref(KID_APP_PORTS.science)).toBe('http://localhost:19122')
    expect(kidAppHref(KID_APP_PORTS.poem)).toBe('http://localhost:19162')
    expect(kidAppHref(KID_APP_PORTS.logic)).toBe('http://localhost:19192')
    expect(kidAppHref(KID_APP_PORTS.chengyu)).toBe('http://localhost:19182')
    expect(kidAppHref(KID_APP_PORTS.phrase)).toBe('http://localhost:19172')
  })
})
