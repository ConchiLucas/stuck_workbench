import { knowledge } from './knowledge'
import type { Suggestion } from './knowledgeTypes'
export function requestKey(){return crypto.randomUUID()}
export function saveSuggestion(input:unknown,key:string){return knowledge<Suggestion>('/review-suggestions',{method:'POST',headers:{'Content-Type':'application/json','Idempotency-Key':key},body:JSON.stringify(input)})}
export function suggestionCommand(id:number,op:string,version:number,key:string,partitionKeys?:string[]){return knowledge<Suggestion>(`/review-suggestions/${id}/${op}`,{method:'POST',headers:{'Content-Type':'application/json','Idempotency-Key':key},body:JSON.stringify({expectedRowVersion:version,...(partitionKeys?{partitionKeys}:{})})})}
