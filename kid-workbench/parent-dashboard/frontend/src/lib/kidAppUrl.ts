import { gatewayLink } from '../appPath'
/** 识字孩子端地址（Docker / 本地默认 19152）。 */
export const kidAppUrl = gatewayLink(19152, import.meta.env.VITE_KID_APP_URL ?? 'http://localhost:19152')
