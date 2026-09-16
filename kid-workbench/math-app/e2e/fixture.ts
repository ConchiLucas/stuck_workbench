import { test as base, expect } from '@playwright/test'
import { readFile } from 'node:fs/promises'
import { resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { MathCatalog } from '@kid-workbench/math-player'

const origin = 'https://math.test'
const dist = fileURLToPath(new URL('../dist/', import.meta.url))
const defaults = fileURLToPath(new URL('../../shared-go/mathcontent/defaults.json', import.meta.url))
const mime: Record<string, string> = {
  js: 'text/javascript', css: 'text/css', html: 'text/html', json: 'application/json',
  png: 'image/png', svg: 'image/svg+xml', woff2: 'font/woff2', ico: 'image/x-icon',
  webmanifest: 'application/manifest+json',
}

export type MockMathAPI = {
  catalog: MathCatalog
  unavailable: boolean
  reads: string[]
  writes: string[]
  unexpected: string[]
}

// Every request is fulfilled from the built app or this in-memory catalog.
// Unknown hosts, API routes and all writes fail the test instead of reaching a server.
export const test = base.extend<{ mockAPI: MockMathAPI }>({
  mockAPI: [async ({ page }, use) => {
    await readFile(resolve(dist, 'index.html'))
    const mock: MockMathAPI = {
      catalog: JSON.parse(await readFile(defaults, 'utf8')),
      unavailable: false, reads: [], writes: [], unexpected: [],
    }
    const pageErrors: string[] = []
    page.on('pageerror', error => pageErrors.push(error.message))
    await page.addInitScript(() => {
      if ('speechSynthesis' in window) {
        window.speechSynthesis.speak = () => { throw new Error('Prepared audio must not fall back to browser TTS') }
      }
    })
    await page.route('**/*', async route => {
      const request = route.request()
      const url = new URL(request.url())
      if (request.method() !== 'GET') {
        mock.writes.push(`${request.method()} ${request.url()}`)
        return route.abort('blockedbyclient')
      }
      // The production HTML references Google Fonts; provide an empty local stylesheet.
      if (url.origin === 'https://fonts.googleapis.com' && url.pathname === '/css2') {
        return route.fulfill({ contentType: 'text/css', body: '/* Isolated tests use system fonts. */' })
      }
      if (url.origin !== origin) {
        mock.unexpected.push(`external request ${request.url()}`)
        return route.abort('blockedbyclient')
      }
      if (url.pathname === '/api/v1/math/details') {
        mock.reads.push(url.pathname)
        return mock.unavailable
          ? route.fulfill({ status: 503, json: { data: null, error: { code: 'unavailable', message: '素材服务暂不可用' } } })
          : route.fulfill({ json: { data: mock.catalog, error: null } })
      }
      if (url.pathname.startsWith('/api/')) {
        mock.unexpected.push(`unhandled API ${url.pathname}`)
        return route.abort('blockedbyclient')
      }
      const relative = request.isNavigationRequest() ? 'index.html' : decodeURIComponent(url.pathname).replace(/^\/+/, '')
      const asset = resolve(dist, relative)
      if (!asset.startsWith(resolve(dist) + sep)) {
        mock.unexpected.push(`invalid asset ${url.pathname}`)
        return route.abort('blockedbyclient')
      }
      try {
        return await route.fulfill({ body: await readFile(asset), contentType: mime[asset.split('.').at(-1)!] ?? 'application/octet-stream' })
      } catch {
        mock.unexpected.push(`missing built asset ${url.pathname}`)
        return route.fulfill({ status: 404, body: 'Missing local test asset' })
      }
    })
    await use(mock)
    expect(mock.writes, 'Sample interactions must never submit an attempt or any POST').toEqual([])
    expect(mock.unexpected, 'All network requests must stay inside the isolated fixture').toEqual([])
    expect(pageErrors, 'The built app must not raise runtime errors').toEqual([])
  }, { auto: true }],
})
export { expect }
