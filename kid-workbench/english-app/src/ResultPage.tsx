import {Link,Navigate,useParams} from 'react-router-dom'
import {IconClose} from './icons'
import {demoCorrect,demoHref,demoPickedLabel,demoPromptLabel,isDemoCorrect,questionDemos,quizToChoiceDemo} from './questionDemos'
import {questionTypes} from './questionTypes'
import {practiceKind} from './QuestionTypePreview'
import {useDemoAnswerStore} from './store/demoAnswerStore'
import {useLiveQuizStore,type LiveQuizCard} from './store/liveQuizStore'

export function ResultPage(){
  const{id}=useParams()
  const item=questionTypes.find(x=>x.id===id)
  const live=id==='audio-choice'||id==='image-text'
  const liveQuestions=useLiveQuizStore(s=>live?s.questionsByType[id as LiveQuizCard]:undefined)
  const fallback=useLiveQuizStore(s=>live&&Boolean(s.fallback[id as LiveQuizCard]))
  const invalidate=useLiveQuizStore(s=>s.invalidate)
  const bank=live&&liveQuestions?.length&&!fallback?liveQuestions.map(quizToChoiceDemo):id?questionDemos[id]:undefined
  const picks=useDemoAnswerStore(s=>s.picks)
  const clearType=useDemoAnswerStore(s=>s.clearType)
  if(!item||!bank||!id)return <Navigate to="/" replace/>
  const rows=bank.map((demo,index)=>{
    const n=index+1
    const picked=picks[`${id}:${n}`]
    const skipped=!picked
    return {
      n,
      prompt:demoPromptLabel(demo),
      pickedLabel:skipped?'未选':demoPickedLabel(demo,picked),
      answerLabel:demoPickedLabel(demo,demoCorrect(demo)),
      skipped,
      correct:!skipped&&isDemoCorrect(demo,picked),
    }
  })
  const correctCount=rows.filter(row=>row.correct).length
  return <section className="kid-result" aria-label="答题结果">
    <Link className="kid-circle kid-result-close" to="/" aria-label="返回"><IconClose/></Link>
    <p className="kid-result-kind">{practiceKind(item)}</p>
    <h1>答题结果</h1>
    <p className="kid-result-count">答对 {correctCount} / {bank.length} 题</p>
    <ol className="kid-result-list">
      {rows.map(row=><li key={row.n} className={row.skipped?'is-skipped':row.correct?'is-correct':'is-wrong'}>
        <span className="kid-result-seq">第 {row.n} 题</span>
        <strong>{row.prompt}</strong>
        <span>你的答案 {row.pickedLabel}</span>
        <span>正确答案 {row.answerLabel}</span>
        <b>{row.skipped?'未作答':row.correct?'答对':'答错'}</b>
      </li>)}
    </ol>
    <div className="kid-result-actions">
      <Link className="kid-result-retry" to={demoHref(item.id,1)} onClick={()=>{
        clearType(item.id)
        if(live) invalidate(id as LiveQuizCard)
      }}>再练一次</Link>
      <Link className="kid-result-home" to="/">回到首页</Link>
    </div>
  </section>
}
