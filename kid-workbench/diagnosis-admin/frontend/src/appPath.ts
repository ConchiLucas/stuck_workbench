// Support both direct local access and the shared public /apps gateway.
export function appBase(): string {
  const prefix = '/apps/diagnosis-admin'
  const pathname = window.location.pathname
  return pathname === prefix || pathname.startsWith(prefix + '/') ? prefix : ''
}
export function appPath(value: string): string {
  const base = appBase()
  return base && value.startsWith('/') && !value.startsWith('//') && value !== base && !value.startsWith(base + '/') ? base + value : value
}

export function isGateway(): boolean {
  const { hostname, port } = window.location
  return !['localhost', '127.0.0.1', '[::1]'].includes(hostname) && ['', '80', '443'].includes(port)
}
export function gatewayLink(port: number, fallback: string): string {
  const routes: Record<number, string> = {"19081":"progress","19083":"progress","19091":"content-admin","19112":"pinyin","19122":"science","19132":"english","19142":"math","19152":"literacy","19162":"poem","19172":"phrase","19182":"chengyu","19192":"logic","19201":"task-admin","19211":"diagnosis-admin","19212":"diagnosis-admin"}
  return isGateway() && routes[port] ? '/apps/' + routes[port] + '/' : fallback
}
