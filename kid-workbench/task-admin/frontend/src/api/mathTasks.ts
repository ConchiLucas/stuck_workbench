import type {MathDetail} from '@kid-workbench/math-player'
import {genAPI} from './generation'
export type MathTaskItem={sequence:number;sourceId:string;sourceRevision:number;source:MathDetail;detail:MathDetail;mediaSHA256?:Record<string,string>}
export type MathTask={id:number;title:string;status?:'draft'|'published';rangeMax:number;count:number;titles?:string[];groups?:string[];items?:MathTaskItem[];createdAt:string;publishedAt?:string}
export type MathTaskInput={title:string;detailIds:string[];rangeMax:number;count:number}
const base='math/question-tasks'
export const mathTasksAPI={
 materials:()=>genAPI<{schemaVersion:number;items:MathDetail[]}>(`${base}/materials`),
 list:()=>genAPI<{items:MathTask[]}>(base),
 get:(id:number)=>genAPI<MathTask>(`${base}/${id}`),
 create:(input:MathTaskInput)=>genAPI<MathTask>(base,'POST',input),
}
