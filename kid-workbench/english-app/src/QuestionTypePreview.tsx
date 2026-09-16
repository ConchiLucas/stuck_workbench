import {useEffect} from 'react'
import {useNavigate} from 'react-router-dom'
import {EnglishPlayer, ListenButton, type EnglishAnswer, type EnglishKind} from '@kid-workbench/english-player'
import type {QuestionType} from './questionTypes'
import {demoToExample, type DemoItem} from './questionDemos'
import '@kid-workbench/english-player/player.css'

export {ListenButton}

export function practiceKind(item:QuestionType){
  if(item.id==='audio-choice') return '听音选词'
  if(item.id==='image-text') return '看图选词'
  if(item.id==='reading-qa') return '读一读'
  if(item.family==='order') return '组句子'
  return '写单词'
}

function answerOf(demo:DemoItem,value?:string):EnglishAnswer|undefined{
  if(!value) return undefined
  if(demo.mode==='choice') return {kind:'choice',value,correct:null}
  if(demo.mode==='type') return {kind:'input',value,correct:null}
  return {kind:'order',value,correct:null}
}

export function QuestionTypePreview({item,demo,value,onChange,nextTo}:{item:QuestionType;demo:DemoItem;value?:string;onChange:(value:string)=>void;nextTo?:string}){
  const navigate=useNavigate()
  useEffect(()=>{window.scrollTo({top:0,behavior:'auto'})},[item.id,demo])
  const example=demoToExample(item.id as EnglishKind,demo)
  return <EnglishPlayer
    key={`${item.id}-${demo.mode}-${demo.mode==='choice'?demo.answerId:demo.answer}`}
    example={example}
    allowSyntheticSpeech={!example.speechUrl}
    initialAnswer={answerOf(demo,value)}
    onAnswer={answer=>onChange(answer.value)}
    completeTo={nextTo}
    onSettled={nextTo?()=>navigate(nextTo):undefined}
  />
}
