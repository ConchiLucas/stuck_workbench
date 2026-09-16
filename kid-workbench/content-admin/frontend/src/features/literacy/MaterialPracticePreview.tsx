import {WritingTemplateEditor} from './WritingTemplateEditor'
import {useState} from 'react'
import {useQuery} from '@tanstack/react-query'
import {LiteracyPlayer,type PlayerFeedback,type PlayerQuestion} from '@kid-workbench/literacy-player'
import {questionTypes,type QuestionType} from '@kid-workbench/literacy-contract'
import {appPath} from '../../appPath'
type Snapshot={schemaVersion:number;questionType:QuestionType;interaction?:'choice'|'handwriting';prompt?:string;stem:PlayerQuestion['stem'];options?:PlayerQuestion['options'];materialRevisionIds?:string[]}
async function previewAPI<T>(path:string,body:unknown):Promise<T>{const res=await fetch(appPath('/api/v1/generation-preview/literacy'+path),{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});const data=await res.json();if(!res.ok)throw new Error(typeof data.error==='string'?data.error:data.error?.message??'素材尚未就绪');return data}
export function MaterialPracticePreview({kpId,onClose}:{kpId:number;onClose?:()=>void}){
 const [type,setType]=useState<QuestionType>('glyph_sense')
 const preview=useQuery({queryKey:['material-practice-preview',kpId,type],queryFn:()=>previewAPI<Snapshot>('',{kpId,questionType:type}),retry:false,staleTime:0})
 return <section className="material-practice-preview"><h2>素材试做</h2><WritingTemplateEditor kpId={kpId}/><p>冻结当前素材版本后试做，不记录孩子学习情况。</p><div className="heading-actions">{questionTypes.map(t=><button className="mini-btn" key={t.code} aria-pressed={type===t.code} onClick={()=>setType(t.code)}>{t.label}</button>)}<button className="mini-btn" onClick={()=>void preview.refetch()}>使用最新素材</button>{onClose&&<button className="mini-btn" onClick={onClose}>关闭试做</button>}</div>{preview.isFetching?<p>正在校验并冻结素材…</p>:preview.error?<p role="alert">{preview.error.message}</p>:preview.data&&<><LiteracyPlayer key={preview.dataUpdatedAt} mode="preview" question={{id:preview.dataUpdatedAt,questionType:preview.data.questionType,interaction:preview.data.interaction??(type==='write_char'?'handwriting':'choice'),prompt:preview.data.prompt,stem:preview.data.stem,options:preview.data.options?.map(option=>({...option,audio:preview.data.questionType==='glyph_sense'?option.audio:undefined}))}} mediaResolver={ref=>{const r=ref as {revisionId:string;kind:string};return appPath(`/api/v1/material-revisions/${encodeURIComponent(r.revisionId)}/media/${r.kind}`)}} onSubmit={response=>previewAPI<PlayerFeedback>('/answer',{snapshot:preview.data,response})}/><details><summary>冻结素材版本</summary><p>{preview.data.materialRevisionIds?.join(' · ')}</p></details></>}</section>
}
