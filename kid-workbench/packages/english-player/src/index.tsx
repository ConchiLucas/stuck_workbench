import {useEffect,useRef,useState,type ReactNode} from 'react'

export type EnglishKind='audio-choice'|'image-text'|'card-builder'|'input-gap'|'reading-qa'
export type EnglishChoice={id:string;label:string;picture?:string}
export type EnglishExample={
  kind:EnglishKind
  prompt?:string
  passage?:string
  speech?:string
  speechUrl?:string
  cue?:string
  options?:EnglishChoice[]
  answerId?:string
  bank?:string[]
  answer?:string
  wide?:boolean
}
export type EnglishAnswer={kind:'choice'|'order'|'input';value:string;correct:boolean|null}

export const englishKinds:EnglishKind[]=['audio-choice','image-text','card-builder','input-gap','reading-qa']
export const englishTitles:Record<EnglishKind,string>={
  'audio-choice':'听音选词','image-text':'看图选词','card-builder':'组句子','input-gap':'写单词','reading-qa':'读一读',
}
export const englishSkills:Record<EnglishKind,string>={
  'audio-choice':'listen','image-text':'picture','card-builder':'build','input-gap':'type','reading-qa':'read',
}

export function isImageSrc(picture?:string){
  return Boolean(picture&&(picture.startsWith('/')||picture.startsWith('http')))
}
export function hideEnglishOnPicture(label:string,picture?:string){
  return Boolean(picture&&/^[A-Za-z]+$/.test(label))
}

function IconPlay(){
  return <svg width={56} height={56} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M4.5 5.653c0-1.427 1.529-2.33 2.779-1.643l11.54 6.347c1.295.712 1.295 2.573 0 3.286L7.28 19.99c-1.25.687-2.779-.217-2.779-1.643V5.653Z"/></svg>
}
function IconCheck(){return <svg width={28} height={28} viewBox="0 0 256 256" fill="currentColor" aria-hidden="true"><path d="M229.66 77.66l-128 128a8 8 0 0 1-11.32 0l-56-56a8 8 0 0 1 11.32-11.32L96 188.69 218.34 66.34a8 8 0 0 1 11.32 11.32Z"/></svg>}
function IconWrong(){return <svg width={28} height={28} viewBox="0 0 256 256" fill="currentColor" aria-hidden="true"><path d="M205.66 194.34a8 8 0 0 1-11.32 11.32L128 139.31 61.66 205.66a8 8 0 0 1-11.32-11.32L116.69 128 50.34 61.66A8 8 0 0 1 61.66 50.34L128 116.69l66.34-66.35a8 8 0 0 1 11.32 11.32L139.31 128Z"/></svg>}

function speak(text:string){
  if(!('speechSynthesis' in window)) return
  window.speechSynthesis.cancel()
  const utterance=new SpeechSynthesisUtterance(text)
  utterance.lang='en-US'
  utterance.rate=0.86
  window.speechSynthesis.speak(utterance)
}

export function ListenButton({onClick,disabled}:{onClick:()=>void;disabled?:boolean}){
  const [playing,setPlaying]=useState(false)
  const timer=useRef(0)
  useEffect(()=>()=>window.clearTimeout(timer.current),[])
  function play(){
    if(disabled) return
    onClick()
    setPlaying(true)
    window.clearTimeout(timer.current)
    timer.current=window.setTimeout(()=>setPlaying(false),1600)
  }
  return <button type="button" className={`kid-play${playing?' is-playing':''}`} aria-label="播放" aria-pressed={playing} disabled={disabled} onClick={play}>
    <span className="kid-play-rings" aria-hidden="true"><i/><i/><i/></span>
    <span className="kid-play-face" data-icon="play" aria-hidden="true"><IconPlay/></span>
  </button>
}

function Face({label,picture,state,resolveAssetUrl}:{label:string;picture?:string;state?:string;resolveAssetUrl:(url:string)=>string}){
  return <>
    {picture&&(isImageSrc(picture)
      ?<img className="kid-pic" src={resolveAssetUrl(picture)} alt=""/>
      :<span className="kid-pic" aria-hidden="true">{picture}</span>)}
    {!hideEnglishOnPicture(label,picture)&&<strong>{label}</strong>}
    {state?.includes('right')&&<span className="kid-mark" aria-hidden="true"><IconCheck/></span>}
    {state?.includes('wrong')&&<span className="kid-mark" aria-hidden="true"><IconWrong/></span>}
  </>
}

function Stage({prompt,action,lead,children,feedback}:{prompt?:string;action?:ReactNode;lead?:ReactNode;children:ReactNode;feedback?:string}){
  return <>
    <div className="kid-board">
      <div className="kid-prompt">{lead}{prompt&&<h1>{prompt}</h1>}{action}</div>
      <div className="kid-work">{children}</div>
    </div>
    {feedback&&<p className="kid-feedback" role="status">{feedback}</p>}
  </>
}

function usedFromValue(bank:string[],answer:string,value:string){
  if(!value) return [] as number[]
  const tokens=answer.includes(' ')?value.split(' ').filter(Boolean):[...value]
  const used:number[]=[]
  const taken=new Set<number>()
  for(const token of tokens){
    const index=bank.findIndex((item,i)=>item===token&&!taken.has(i))
    if(index<0) break
    used.push(index)
    taken.add(index)
  }
  return used
}

export function EnglishPlayer({
  example:e,readOnly=false,allowSyntheticSpeech=false,initialAnswer,onAnswer,resolveAssetUrl=(url)=>url,completeTo,onSettled,
}:{
  example:EnglishExample
  readOnly?:boolean
  allowSyntheticSpeech?:boolean
  initialAnswer?:EnglishAnswer
  onAnswer?:(answer:EnglishAnswer)=>void
  resolveAssetUrl?:(url:string)=>string
  completeTo?:string
  onSettled?:()=>void
}){
  const speechUrl=e.speechUrl?resolveAssetUrl(e.speechUrl):''
  const [audioError,setAudioError]=useState(false)
  const needsSpeech=e.kind==='audio-choice'||e.kind==='input-gap'||e.kind==='card-builder'
  const missingSpeech=needsSpeech&&!speechUrl&&!allowSyntheticSpeech
  function playClip(){
    if(speechUrl){
      const audio=new Audio(speechUrl)
      audio.onerror=()=>setAudioError(true)
      const playing=audio.play()
      if(playing&&typeof playing.then==='function'){
        void playing.then(()=>setAudioError(false)).catch(()=>{
          setAudioError(true)
          if(allowSyntheticSpeech&&e.speech) speak(e.speech)
        })
      }
      return
    }
    if(allowSyntheticSpeech&&e.speech) speak(e.speech)
  }
  const audioAction=(!missingSpeech&&(speechUrl||(allowSyntheticSpeech&&e.speech)))
    ?<ListenButton onClick={playClip}/>:undefined
  const audioNotice=(missingSpeech||audioError)&&<p className="kid-audio-missing" role="alert">{missingSpeech?'读音素材暂不可用':'读音播放失败，可以再点一次。'}</p>
  const lead=<>
    {e.passage&&<p className="kid-read">{e.passage}</p>}
    {e.cue&&(isImageSrc(e.cue)?<div className="kid-cue"><img src={resolveAssetUrl(e.cue)} alt=""/></div>:<div className="kid-cue" aria-hidden="true">{e.cue}</div>)}
    {audioNotice}
  </>

  if(e.kind==='card-builder'){
    return <PlayShell readOnly={readOnly}><TilePlay example={e} readOnly={readOnly} initial={initialAnswer} onAnswer={onAnswer} action={audioAction} lead={lead}/></PlayShell>
  }
  if(e.kind==='input-gap'){
    return <PlayShell readOnly={readOnly}><TypePlay example={e} readOnly={readOnly} initial={initialAnswer} onAnswer={onAnswer} completeTo={completeTo} onSettled={onSettled} action={audioAction} lead={lead}/></PlayShell>
  }
  return <PlayShell readOnly={readOnly}><ChoicePlay example={e} readOnly={readOnly} initial={initialAnswer} onAnswer={onAnswer} onSettled={onSettled} resolveAssetUrl={resolveAssetUrl} action={audioAction} lead={lead}/></PlayShell>
}

function PlayShell({children,readOnly}:{children:ReactNode;readOnly:boolean}){
  return <div className={`english-player${readOnly?' is-readonly':''}`} data-readonly={readOnly||undefined}><section className="kid-stage" aria-label="当前题目">{children}</section></div>
}

function ChoicePlay({example:e,readOnly,initial,onAnswer,onSettled,resolveAssetUrl,action,lead}:{example:EnglishExample;readOnly:boolean;initial?:EnglishAnswer;onAnswer?:(answer:EnglishAnswer)=>void;onSettled?:()=>void;resolveAssetUrl:(url:string)=>string;action?:ReactNode;lead?:ReactNode}){
  const options=e.options??[]
  const answerId=e.answerId??''
  const [picked,setPicked]=useState(initial?.value)
  const [tries,setTries]=useState(initial?.value?2:0)
  const locked=readOnly||Boolean(initial?.value)||tries>=2||picked===answerId
  const correct=picked===answerId
  useEffect(()=>{
    if(!locked||readOnly||!onSettled||!picked) return
    const timer=window.setTimeout(onSettled,450)
    return ()=>window.clearTimeout(timer)
  },[locked,readOnly,onSettled,picked])
  const feedback=readOnly
    ?(picked?(correct?'当时答对':'当时答错'):undefined)
    :(picked?(correct?'答对了':tries>=2?'看正确答案':'再试一次'):undefined)
  return <Stage prompt={e.prompt} lead={lead} action={action} feedback={feedback}>
    <div className="kid-options">{options.map(option=>{
      const isPicked=picked===option.id
      const showRight=locked&&option.id===answerId
      const showWrong=locked&&isPicked&&option.id!==answerId
      return <button type="button" key={option.id} className={`kid-option${isPicked?' is-picked':''}${showRight?' is-right':''}${showWrong?' is-wrong':''}${hideEnglishOnPicture(option.label,option.picture)?' is-pic':''}`} aria-label={option.label} aria-pressed={isPicked} disabled={locked} onClick={()=>{
        if(locked) return
        setPicked(option.id)
        const nextTries=tries+1
        setTries(nextTries)
        if(option.id===answerId||nextTries>=2) onAnswer?.({kind:'choice',value:option.id,correct:option.id===answerId})
      }}>
        <Face label={option.label} picture={option.picture} state={showRight?'right':showWrong?'wrong':undefined} resolveAssetUrl={resolveAssetUrl}/>
      </button>
    })}</div>
  </Stage>
}

function TypePlay({example:e,readOnly,initial,onAnswer,completeTo,onSettled,action,lead}:{example:EnglishExample;readOnly:boolean;initial?:EnglishAnswer;onAnswer?:(answer:EnglishAnswer)=>void;completeTo?:string;onSettled?:()=>void;action?:ReactNode;lead?:ReactNode}){
  const [value,setValue]=useState(initial?.value??'')
  const [submitted,setSubmitted]=useState(Boolean(readOnly&&initial?.value))
  useEffect(()=>{setValue(initial?.value??'');setSubmitted(Boolean(readOnly&&initial?.value))},[initial?.value,e.answer])
  const correct=value.trim().toLowerCase()===(e.answer??'').toLowerCase()
  const canSubmit=Boolean(value.trim())&&!readOnly
  function commit(next:string){
    setValue(next)
    setSubmitted(false)
    onAnswer?.({kind:'input',value:next,correct:next.trim().toLowerCase()===(e.answer??'').toLowerCase()})
  }
  function go(){
    if(!value.trim()||readOnly) return
    setSubmitted(true)
    onAnswer?.({kind:'input',value,correct})
    if(completeTo||onSettled) onSettled?.()
  }
  const feedback=readOnly
    ?(initial?.value?(correct?'当时答对':'当时答错'):undefined)
    :(submitted?(correct?'答对了':'再试一次'):undefined)
  return <Stage prompt={e.prompt} lead={lead} action={action} feedback={feedback}>
    <div className="kid-write">
      <input className="kid-write-field" aria-label="单词" value={value} autoFocus={!readOnly} autoComplete="off" autoCorrect="off" autoCapitalize="none" spellCheck={false} inputMode="text" disabled={readOnly} onChange={ev=>commit(ev.target.value.replace(/[^a-zA-Z]/g,''))} onKeyDown={ev=>{if(ev.key==='Enter'){ev.preventDefault();go()}}}/>
      {!readOnly&&canSubmit&&completeTo&&<a className="kid-write-go" href={completeTo} onClick={ev=>{if(onSettled){ev.preventDefault();onSettled()}}}>写好了</a>}
      {!readOnly&&canSubmit&&!completeTo&&<button type="button" className="kid-write-go" onClick={go}>写好了</button>}
    </div>
  </Stage>
}

function TilePlay({example:e,readOnly,initial,onAnswer,action,lead}:{example:EnglishExample;readOnly:boolean;initial?:EnglishAnswer;onAnswer?:(answer:EnglishAnswer)=>void;action?:ReactNode;lead?:ReactNode}){
  const resolveAssetUrl=(url:string)=>url
  const bank=e.bank??[]
  const answer=e.answer??''
  const spaced=answer.includes(' ')
  const slots=spaced?answer.split(' ').length:answer.length
  const [lineValue,setLineValue]=useState(initial?.value??'')
  useEffect(()=>{setLineValue(initial?.value??'')},[initial?.value,answer])
  const used=usedFromValue(bank,answer,lineValue)
  const usedRef=useRef(used)
  usedRef.current=used
  const line=used.map(index=>bank[index])
  const complete=used.length===slots
  const correct=lineValue===answer
  function commit(next:number[]){
    const tokens=next.map(index=>bank[index])
    const nextValue=spaced?tokens.join(' '):tokens.join('')
    usedRef.current=next
    setLineValue(nextValue)
    onAnswer?.({kind:'order',value:nextValue,correct:nextValue===answer})
  }
  function add(index:number){
    if(readOnly) return
    const current=usedRef.current
    if(current.includes(index)||current.length>=slots) return
    commit([...current,index])
  }
  const feedback=readOnly
    ?(initial?.value?(initial.value===answer?'当时答对':'当时答错'):undefined)
    :(complete?(correct?'答对了':'再试一次'):undefined)
  return <Stage prompt={e.prompt} lead={lead} action={action} feedback={feedback}>
    <div className="kid-slots">
      {Array.from({length:slots},(_,i)=>line[i]
        ?<span key={`${line[i]}-${i}`} className="kid-tag is-kind">{line[i]}</span>
        :<span key={i} className="kid-tag is-slot"/>)}
      {!readOnly&&used.length>0&&<button type="button" className="kid-tag kid-reset" onClick={()=>{usedRef.current=[];setLineValue('');onAnswer?.({kind:'order',value:'',correct:false})}}>重来</button>}
    </div>
    <div className={`kid-options${e.wide?' is-wide':''}`}>
      {bank.map((token,i)=>{
        const picked=used.includes(i)
        return <button type="button" key={`${token}-${i}`} className={`kid-option${picked?' is-used':''}`} aria-label={token} disabled={readOnly||picked} onClick={()=>add(i)}>
          <Face label={token} resolveAssetUrl={resolveAssetUrl}/>
        </button>
      })}
    </div>
  </Stage>
}
