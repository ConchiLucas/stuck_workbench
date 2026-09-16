import { gatewayLink } from '../appPath'
/** 诊断台：本地 Vite 19212，Docker 内嵌 19211。 */
export const diagnosisUrl = gatewayLink(19211, import.meta.env.VITE_DIAGNOSIS_URL ?? 'http://localhost:19212')
