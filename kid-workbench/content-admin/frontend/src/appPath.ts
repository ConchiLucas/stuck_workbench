// Public gateway deployment and direct localhost access use the same build.
export function appBase(): string {
  return /^\/apps\/content-admin(?:\/|$)/.test(window.location.pathname)
    ? '/apps/content-admin' : ''
}
export function appPath(path: string): string {
  const base = appBase()
  return base && path.startsWith('/') && !path.startsWith('//') &&
    path !== base && !path.startsWith(base + '/') ? base + path : path
}

export function isGateway(): boolean {
  const { hostname, port } = window.location
  return !['localhost', '127.0.0.1', '[::1]'].includes(hostname) && ['', '80', '443'].includes(port)
}
export function gatewayLink(port: number, fallback: string): string {
  const routes: Record<number, string> = {"19081":"progress","19083":"progress","19091":"content-admin","19112":"pinyin","19122":"science","19132":"english","19142":"math","19152":"literacy","19162":"poem","19172":"phrase","19182":"chengyu","19192":"logic","19201":"task-admin","19211":"diagnosis-admin","19212":"diagnosis-admin"}
  return isGateway() && routes[port] ? '/apps/' + routes[port] + '/' : fallback
}
