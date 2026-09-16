import { appPath } from '../appPath'
import { getChildId } from '../store/childStore'
export class KnowledgeError extends Error {code:string;constructor(message:string,code='request_failed',public status=0){super(message);this.code=code}}
export const base=()=>appPath(`/api/v1/children/${getChildId()}/knowledge`)
export async function knowledge<T>(path:string,init?:RequestInit):Promise<T>{const res=await fetch(base()+path,init);const b=await res.json().catch(()=>({error:{message:'暂时无法读取，请稍后重试'}}));if(!res.ok)throw new KnowledgeError(typeof b.error==='string'?b.error:b.error?.message||'请求失败',b.error?.code,res.status);return b as T}
export function queryString(values:Record<string,string|number|undefined>){const q=new URLSearchParams();Object.entries(values).forEach(([k,v])=>{if(v!==undefined&&v!=='')q.set(k,String(v))});return q.size?'?'+q.toString():''}
export const mediaURL=(attemptId:number,id:string)=>base()+`/attempts/${attemptId}/media/${encodeURIComponent(id)}`
export const statusLabel=(s:string)=>({mastered:'已掌握',review_due:'已掌握 · 到期复习',shaky:'待巩固',learning:'练习中',not_started:'尚未练习'}[s]||'尚无记录')
export const followLabel=(s:string)=>({no_later_practice:'之后还没练过',still_wrong:'后续仍有错误',answered_correctly_later:'后续已答对',mastered_later:'后续已掌握',insufficient_evidence:'后续证据不足'}[s]||'查看后续记录')
export const timeLabel=(s:string)=>new Date(s).toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit',hour12:false})
export const reasonLabel=(s:string)=>({evidence_incomplete:'历史记录不足以生成原题',source_plan_incomplete:'来源练习尚未完成',unsupported_subject:'这个学科的复习出题尚未接入',unsupported_source:'这类历史记录暂不支持出题',missing_material:'缺少所需素材',insufficient_variants:'可用变式不足'}[s]||'暂不满足生成条件')

export const skillLabel=(s:string)=>({glyph_sense:'看字选义',sense_char:'看义选字',write_char:'听音写字',listen:'听音选词',inword:'听例字选音',shape:'辨认字形',blend:'音节拼读',picture:'看图选词',build:'组句子',type:'写单词',read:'读一读',calc:'算式计算',story:'情境应用',find:'听音找图形',name:'看图认名称',recognize:'科普辨认',choice:'选择题',match:'连线题',sequence:'排序题',label:'结构标注题',listen_zh:'听一听',listen_en:'选句子',scene:'什么时候说',reply:'问与答',meaning:'听释义',pick:'选成语',pinyin:'看拼音',example:'看句子'}[s]||'历史题型')
