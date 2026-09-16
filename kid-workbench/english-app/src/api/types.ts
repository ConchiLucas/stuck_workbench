export type Status='not_started'|'learning'|'shaky'|'review_due'|'mastered';export interface Module{code:string;name:string;wordCount:number};export interface Word{kpId:number;word:string;meaningZh:string;phonetic?:string;partOfSpeech?:string;example?:string;exampleMeaningZh?:string;moduleCode:string;moduleName:string;hasSense:boolean;hasSpeech:boolean};export interface Home{child:{id:number;name:string;flowers:number};currentPlan?:{id:number;status:string};dueCount:number;modules:{code:string;name:string;mastered:number;total:number}[]};export interface Option{kpId:number;label?:string;assetUrl?:string};export interface PlanItem{id:number;kpId:number;word:string;meaningZh:string;status:string;tries:number;question:{code:'listen'|'picture';stem:string;options:Option[];speech:{text:string}}};export interface Plan{plan:{id:number;doneCount:number;targetCount:number;status:string};items:PlanItem[]}
export type EnglishQuizType='listen'|'look'
export type EnglishGeneratedQuiz={
  instanceId:string
  type:EnglishQuizType
  stem:string
  targetId:number
  speechText?:string
  speechUrl?:string
  visual:{kind:string;text?:string;imageUrl?:string}
  options:{id:number;label?:string;imageUrl?:string}[]
  answerIndex:number
}

