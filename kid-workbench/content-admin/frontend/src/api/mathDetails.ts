import type { MathCatalog, MathDetail } from '@kid-workbench/math-player'
import { appPath } from '../appPath'
export type AdminMathDetail = MathDetail & {publishedRevision?:number}
export type AdminMathCatalog = Omit<MathCatalog,'items'> & {items:AdminMathDetail[]}
async function request<T>(path:string,method='GET',body?:unknown):Promise<T>{
 const response=await fetch(appPath('/api/v1/math/details'+path),{method,headers:{'Content-Type':'application/json'},...(body===undefined?{}:{body:JSON.stringify(body)})})
 const value=await response.json().catch(()=>null)
 if(!response.ok)throw new Error(typeof value?.error==='string'?value.error:value?.error?.message??`请求失败 ${response.status}`)
 return value as T
}
export async function listMathDetails(){
 const catalog=await request<AdminMathCatalog>('')
 if(catalog?.schemaVersion!==1||!Array.isArray(catalog.items))throw new Error('素材目录格式或版本不受支持')
 return catalog
}
export const saveMathDetail=(detail:MathDetail)=>request<AdminMathDetail>('/'+encodeURIComponent(detail.id),'PUT',detail)
export const publishMathDetail=(detail:MathDetail)=>request<AdminMathDetail>('/'+encodeURIComponent(detail.id)+'/publish','POST',{revision:detail.revision})
export const generateMathDetailAudio=(detail:MathDetail)=>request<AdminMathDetail>('/'+encodeURIComponent(detail.id)+'/audio','POST',{revision:detail.revision})
