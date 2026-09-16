import { gatewayLink } from '../appPath'
/** 进度后台：本地 Vite 19083，Docker 内嵌 19081。 */
export const progressUrl = gatewayLink(19081, import.meta.env.VITE_PROGRESS_URL ?? 'http://localhost:19083')
