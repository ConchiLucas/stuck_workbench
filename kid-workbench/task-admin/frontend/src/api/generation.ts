import {appPath} from '../appPath'
export type Media={revisionId:string;kind:string;sha256:string}
export type Snapshot={schemaVersion:number;interaction?:'choice'|'handwriting';responseSchemaVersion?:number;kpId:number;targetText:string;questionType:string;prompt:string;stem:{text?:string;image?:Media;audio?:Media};options:{id:string;kpId:number;text:string;image?:Media;audio?:Media}[];answerOptionId:string;explanation:string;materialRevisionIds:string[]}
export type Spec={subjectCode:string;kind:string;scope:{moduleCodes:string[];kpIds:number[]};targetCount:number;typeCounts:Record<string,number>;distractorScope:string}
export type Version={id:number;seq:number;fingerprint:string;snapshot:Snapshot;sourceQuestionVersionId?:number}
export type Task={sourceReviewSuggestionId?:number;id:number;title:string;subjectCode:string;moduleName:string;targetCount:number;status:string;kind:string;sourceMode:string;rowVersion:number;activeRevisionId?:number;publishedRevisionId?:number;targetChildId?:number;parentTaskId?:number;updatedAt:string;spec:Spec;items:Version[];revisions:{id:number;revisionNo:number;createdAt:string}[];lastError?:string}
export type Material={kpId:number;text:string;moduleName:string;capabilities:Record<string,{ready:boolean;reasons:string[]}>}
export const mediaURL=(ref?:Media)=>ref?appPath(`/api/v1/material-revisions/${encodeURIComponent(ref.revisionId)}/media/${ref.kind}`):undefined
export class GenerationAPIError extends Error { constructor(message:string,public code?:string){super(message)} }
export async function genAPI<T>(path:string,method='GET',body?:unknown,key?:string):Promise<T>{
 const res=await fetch(appPath('/api/v1/'+path),{method,headers:{'Content-Type':'application/json',...(key?{'Idempotency-Key':key}:{})},...(body===undefined?{}:{body:JSON.stringify(body)})})
 if(res.status===204)return undefined as T
 const data=await res.json().catch(()=>{throw new Error('服务没有返回有效数据，请使用相同操作重试')})
 if(!res.ok)throw new GenerationAPIError(typeof data.error==='string'?data.error:data.error?.message||'操作失败',data.error?.code)
 return data as T
}
