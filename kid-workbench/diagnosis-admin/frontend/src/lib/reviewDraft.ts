import type { Candidate, Page } from '../api/knowledgeTypes'
import { knowledge, queryString } from '../api/knowledge'
import { getChildId } from '../store/childStore'
export type DraftItem={key:string;kpId:number;subjectCode:string;count:number;mode:string}
export type ReviewChoice={candidate:Candidate;count:number;mode:string}
export const draftKey=(child:number)=>`knowledge-review-draft:${child}`
export function readReviewDraft(child:number):DraftItem[] {
 try {
  const items=JSON.parse(localStorage.getItem(draftKey(child))||'[]')
  if(!Array.isArray(items)||items.length>20||items.some(i=>!i||typeof i.key!=='string'||!Number.isSafeInteger(i.kpId)||i.kpId<1||typeof i.subjectCode!=='string'||!Number.isInteger(i.count)||i.count<1||i.count>10||!['mixed','original_only'].includes(i.mode)))return []
  if(new Set(items.map(i=>i.subjectCode)).size>1||items.reduce((n,i)=>n+i.count,0)>20)return []
  return items
 }catch{return []}
}
export function writeReviewDraft(child:number,items:DraftItem[]) {
 try{if(items.length)localStorage.setItem(draftKey(child),JSON.stringify(items));else localStorage.removeItem(draftKey(child))}catch{/* Storage can be disabled; in-memory selection continues to work. */}
}
export const compactChoices=(choices:Record<string,ReviewChoice>):DraftItem[]=>Object.values(choices).map(({candidate:c,count,mode})=>({key:c.key,kpId:c.kpId,subjectCode:c.subjectCode,count,mode}))
export async function refreshReviewChoices(child:number,items:DraftItem[]) {
 const choices:Record<string,ReviewChoice>={};let asOf='',missing=0
 const groups=new Map<string,DraftItem[]>()
 for(const item of items){const k=`${item.subjectCode}:${item.kpId}`;groups.set(k,[...(groups.get(k)||[]),item])}
 await Promise.all([...groups.values()].map(async group=>{
  let cursor='';const found=new Map<string,Candidate>();const seen=new Set<string>()
  do {
   if(getChildId()!==child)throw new Error('孩子已切换，请重新选择复习内容')
   const page=await knowledge<Page<Candidate>>('/review-candidates'+queryString({subject:group[0].subjectCode,kpId:group[0].kpId,cursor}))
   for(const c of page.items)if(c.kpId===group[0].kpId&&c.subjectCode===group[0].subjectCode)found.set(c.key,c)
   if(page.evidenceAsOf>asOf)asOf=page.evidenceAsOf
   cursor=page.hasMore?page.nextCursor||'':''
   if(cursor&&seen.has(cursor))throw new Error('复习候选分页已变化，请重试')
   seen.add(cursor)
  }while(cursor)
  for(const item of group){const candidate=found.get(item.key);if(candidate)choices[item.key]={candidate,count:item.count,mode:item.mode};else missing++}
 }))
 if(getChildId()!==child)throw new Error('孩子已切换，请重新选择复习内容')
 return {choices,asOf,missing}
}
