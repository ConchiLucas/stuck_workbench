import type {EnglishExample,EnglishKind} from '@kid-workbench/english-player'

export type DemoChoice={id:string;label:string;picture?:string}

export type ChoiceDemo={
  mode:'choice'
  speech?:string
  speechUrl?:string
  prompt?:string
  passage?:string
  options:DemoChoice[]
  answerId:string
}

export type TileDemo={
  mode:'tiles'
  prompt:string
  speech:string
  speechUrl?:string
  cue?:string
  bank:string[]
  answer:string
  wide?:boolean
}

export type TypeDemo={
  mode:'type'
  prompt:string
  speech:string
  speechUrl?:string
  cue?:string
  answer:string
}

export type DemoItem=ChoiceDemo|TileDemo|TypeDemo

const word = {
  apple: 784, banana: 785, dog: 755, bird: 756, cat: 754, cake: 790,
  ball: 834, bag: 847, book: 844, fish: 757, sun: 929, milk: 787, hat: 857,
} as const

function sense(id: number) {
  return `/api/v1/english/words/${id}/sense.png`
}
function speech(id: number) {
  return `/api/v1/english/words/${id}/speech.mp3`
}
function sentenceSpeech(code: string) {
  return `/api/v1/english/sentences/${code}/speech.mp3`
}

export const questionDemos:Record<string,DemoItem[]>={
  'audio-choice':[
    {mode:'choice',speech:'apple',speechUrl:speech(word.apple),answerId:'apple',options:[
      {id:'apple',label:'苹果',picture:sense(word.apple)},{id:'banana',label:'香蕉',picture:sense(word.banana)},{id:'dog',label:'小狗',picture:sense(word.dog)},{id:'bird',label:'小鸟',picture:sense(word.bird)},
    ]},
    {mode:'choice',speech:'cat',speechUrl:speech(word.cat),answerId:'cat',options:[
      {id:'cat',label:'小猫',picture:sense(word.cat)},{id:'fish',label:'小鱼',picture:sense(word.fish)},{id:'sun',label:'太阳',picture:sense(word.sun)},{id:'book',label:'书本',picture:sense(word.book)},
    ]},
    {mode:'choice',speech:'cake',speechUrl:speech(word.cake),answerId:'cake',options:[
      {id:'milk',label:'牛奶',picture:sense(word.milk)},{id:'cake',label:'蛋糕',picture:sense(word.cake)},{id:'hat',label:'帽子',picture:sense(word.hat)},{id:'ball',label:'皮球',picture:sense(word.ball)},
    ]},
    {mode:'choice',speech:'ball',speechUrl:speech(word.ball),answerId:'ball',options:[
      {id:'book',label:'书本',picture:sense(word.book)},{id:'bag',label:'书包',picture:sense(word.bag)},{id:'ball',label:'皮球',picture:sense(word.ball)},{id:'hat',label:'帽子',picture:sense(word.hat)},
    ]},
  ],
  'image-text':[
    {mode:'choice',prompt:'哪一张图是 apple？',answerId:'apple',options:[
      {id:'apple',label:'apple',picture:sense(word.apple)},{id:'banana',label:'banana',picture:sense(word.banana)},{id:'dog',label:'dog',picture:sense(word.dog)},{id:'bird',label:'bird',picture:sense(word.bird)},
    ]},
    {mode:'choice',prompt:'哪一张图是 banana？',answerId:'banana',options:[
      {id:'cat',label:'cat',picture:sense(word.cat)},{id:'banana',label:'banana',picture:sense(word.banana)},{id:'sun',label:'sun',picture:sense(word.sun)},{id:'book',label:'book',picture:sense(word.book)},
    ]},
    {mode:'choice',prompt:'哪一张图是 dog？',answerId:'dog',options:[
      {id:'fish',label:'fish',picture:sense(word.fish)},{id:'cake',label:'cake',picture:sense(word.cake)},{id:'dog',label:'dog',picture:sense(word.dog)},{id:'hat',label:'hat',picture:sense(word.hat)},
    ]},
    {mode:'choice',prompt:'哪一张图是 bird？',answerId:'bird',options:[
      {id:'bird',label:'bird',picture:sense(word.bird)},{id:'ball',label:'ball',picture:sense(word.ball)},{id:'milk',label:'milk',picture:sense(word.milk)},{id:'bag',label:'bag',picture:sense(word.bag)},
    ]},
  ],
  'card-builder':[
    {mode:'tiles',prompt:'把单词排成一句话',speech:'This is an apple',speechUrl:sentenceSpeech('this-is-an-apple'),answer:'This is an apple',bank:['apple','This','an','is']},
    {mode:'tiles',prompt:'把单词排成一句话',speech:'I like the dog',speechUrl:sentenceSpeech('i-like-the-dog'),answer:'I like the dog',bank:['dog','like','I','the']},
    {mode:'tiles',prompt:'把单词排成一句话',speech:'She has a cat',speechUrl:sentenceSpeech('she-has-a-cat'),answer:'She has a cat',bank:['a','has','cat','She']},
    {mode:'tiles',prompt:'把单词排成一句话',speech:'We see a bird',speechUrl:sentenceSpeech('we-see-a-bird'),answer:'We see a bird',bank:['see','We','bird','a']},
  ],
  'input-gap':[
    {mode:'type',prompt:'写出这个单词',speech:'apple',speechUrl:speech(word.apple),cue:sense(word.apple),answer:'apple'},
    {mode:'type',prompt:'写出这个单词',speech:'cat',speechUrl:speech(word.cat),cue:sense(word.cat),answer:'cat'},
    {mode:'type',prompt:'写出这个单词',speech:'dog',speechUrl:speech(word.dog),cue:sense(word.dog),answer:'dog'},
    {mode:'type',prompt:'写出这个单词',speech:'sun',speechUrl:speech(word.sun),cue:sense(word.sun),answer:'sun'},
  ],
  'reading-qa':[
    {mode:'choice',prompt:'Lucy 把什么放在桌上？',passage:'Lucy has a red apple. She puts it on the table.',answerId:'apple',options:[
      {id:'apple',label:'苹果',picture:sense(word.apple)},{id:'banana',label:'香蕉',picture:sense(word.banana)},{id:'dog',label:'小狗',picture:sense(word.dog)},{id:'bag',label:'书包',picture:sense(word.bag)},
    ]},
    {mode:'choice',prompt:'Tom 有什么？',passage:'Tom has a small dog. He plays with it in the park.',answerId:'dog',options:[
      {id:'cat',label:'小猫',picture:sense(word.cat)},{id:'dog',label:'小狗',picture:sense(word.dog)},{id:'ball',label:'皮球',picture:sense(word.ball)},{id:'book',label:'书本',picture:sense(word.book)},
    ]},
    {mode:'choice',prompt:'Mia 喜欢什么？',passage:'Mia likes the yellow banana. She eats it after lunch.',answerId:'banana',options:[
      {id:'cake',label:'蛋糕',picture:sense(word.cake)},{id:'milk',label:'牛奶',picture:sense(word.milk)},{id:'banana',label:'香蕉',picture:sense(word.banana)},{id:'sun',label:'太阳',picture:sense(word.sun)},
    ]},
    {mode:'choice',prompt:'Sam 看见了什么？',passage:'Sam sees a blue bird. It sits on the tree.',answerId:'bird',options:[
      {id:'fish',label:'小鱼',picture:sense(word.fish)},{id:'hat',label:'帽子',picture:sense(word.hat)},{id:'bird',label:'小鸟',picture:sense(word.bird)},{id:'bag',label:'书包',picture:sense(word.bag)},
    ]},
  ],
}

export function demoHref(id:string,n:number){
  return n<=1?`/types/${id}`:`/types/${id}/${n}`
}

export function demoResultHref(id:string){
  return `/types/${id}/result`
}

export function demoCorrect(demo:DemoItem){
  return demo.mode==='choice'?demo.answerId:demo.answer
}

export function demoPromptLabel(demo:DemoItem){
  if(demo.prompt) return demo.prompt
  if(demo.speech) return demo.speech
  return ''
}

export function demoPickedLabel(demo:DemoItem,value?:string){
  if(!value) return '未选'
  if(demo.mode==='choice') return demo.options.find(option=>option.id===value)?.label??'未选'
  return value
}

export function isDemoCorrect(demo:DemoItem,value?:string){
  if(!value) return false
  const answer=demoCorrect(demo)
  if(demo.mode==='type') return value.trim().toLowerCase()===answer.toLowerCase()
  return value===answer
}

export function quizToChoiceDemo(question:import('./api/types').EnglishGeneratedQuiz):ChoiceDemo{
  return {
    mode:'choice',
    prompt:question.visual.kind==='word'
      ?(question.visual.text?`哪一张图是 ${question.visual.text}？`:question.stem)
      :undefined,
    speech:question.speechText,
    speechUrl:question.speechUrl,
    options:question.options.map(option=>({
      id:String(option.id),
      label:option.label??'',
      picture:option.imageUrl,
    })),
    answerId:String(question.options[question.answerIndex]?.id??''),
  }
}

export function demoToExample(kind:EnglishKind,demo:DemoItem):EnglishExample{
  if(demo.mode==='choice'){
    return {kind,prompt:demo.prompt,passage:demo.passage,speech:demo.speech,speechUrl:demo.speechUrl,options:demo.options,answerId:demo.answerId}
  }
  if(demo.mode==='type'){
    return {kind:'input-gap',prompt:demo.prompt,speech:demo.speech,speechUrl:demo.speechUrl,cue:demo.cue,answer:demo.answer}
  }
  return {kind:'card-builder',prompt:demo.prompt,speech:demo.speech,speechUrl:demo.speechUrl,cue:demo.cue,bank:demo.bank,answer:demo.answer,wide:demo.wide}
}
