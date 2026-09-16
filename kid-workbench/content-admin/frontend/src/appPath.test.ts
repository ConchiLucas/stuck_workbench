import { afterEach, describe, expect, it } from 'vitest'
import { appBase, appPath } from './appPath'
afterEach(() => window.history.replaceState({}, '', '/'))
describe('deployment paths', () => {
  it('keeps direct localhost routes', () => {
    window.history.replaceState({}, '', '/literacy')
    expect(appBase()).toBe('')
    expect(appPath('/api/v1/literacy/chars?q=a')).toBe('/api/v1/literacy/chars?q=a')
  })
  it('keeps API and media inside the public application prefix', () => {
    window.history.replaceState({}, '', '/apps/content-admin/literacy')
    expect(appBase()).toBe('/apps/content-admin')
    expect(appPath('/api/v1/literacy/chars?q=a')).toBe('/apps/content-admin/api/v1/literacy/chars?q=a')
    expect(appPath('/apps/content-admin/api/v1/test')).toBe('/apps/content-admin/api/v1/test')
  })
})
