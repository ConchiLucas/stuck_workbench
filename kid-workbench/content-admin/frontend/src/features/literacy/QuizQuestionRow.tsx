import type {LiteracyQuizQuestion} from './quizPreview'
import {MaterialPracticePreview} from './MaterialPracticePreview'
import {speechAudioURL} from '../../api/literacy'
export function QuizQuestionRow({question,mode='list',onOpen}:{question:LiteracyQuizQuestion;mode?:'list'|'detail';onOpen?:()=>void}){return mode==='detail'?<MaterialPracticePreview kpId={question.target.kpId}/>:<button className="quiz-row-hit" onClick={onOpen}>{question.target.charText} · {question.title} · 冻结素材后试做</button>}
export async function playCharSpeech(args:{kpId:number;charText:string;speechUrl?:string;speechAudioUrl?:string}){await new Audio(args.speechUrl||speechAudioURL(args.kpId,args.speechAudioUrl)).play()}
export async function playQuestionSpeech(question:LiteracyQuizQuestion){return playCharSpeech({kpId:question.target.kpId,charText:question.target.charText,speechUrl:question.speechUrl,speechAudioUrl:question.target.speechAudioUrl})}
