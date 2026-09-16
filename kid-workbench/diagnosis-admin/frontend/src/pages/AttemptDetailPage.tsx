import { BackToList, useListReturn } from '../lib/listNavigation'
import { useQuery } from '@tanstack/react-query'
import { Link,useParams } from 'react-router-dom'
import { knowledge,timeLabel,followLabel } from '../api/knowledge'
import type { Evidence } from '../api/knowledgeTypes'
import { getChildId } from '../store/childStore'
import { AnswerEvidence } from '../components/knowledge/AnswerEvidence'
import { LoadState } from '../components/knowledge/Shared'
export function AttemptDetailPage(){const returnState=useListReturn();const{attemptId}=useParams();const child=getChildId();const q=useQuery({queryKey:['knowledge','attempt',child,attemptId],queryFn:()=>knowledge<Evidence>(`/attempts/${attemptId}`)});const e=q.data;return <article><p className="crumb"><BackToList fallback="/wrongs" label="返回列表"/> / 作答现场</p><LoadState loading={q.isPending} error={q.error}/>{e&&<><header className="page-head"><div><p className="eyebrow">{e.subjectName} · {e.moduleName}</p><h1>{e.title} <span className={`result-pill ${e.isCorrect?'ok':'wrong'}`}>{e.isCorrect?'这次答对':'这次答错'}</span></h1><p className="lede">{e.skillLabel} · {timeLabel(e.occurredAt)} · 用时 {(e.costMs/1000).toFixed(1)} 秒</p></div><Link className="text-link" state={returnState()} to={`/knowledge-points/${e.kpId}`}>查看知识点档案 ↗</Link></header><section className="panel"><AnswerEvidence evidence={e}/></section><section className="panel result-context"><h2>这次之后</h2><p>{e.isCorrect?'本次结果已保存在学习记录中。':followLabel(e.followUpState)}</p><Link state={returnState()} to={`/knowledge-points/${e.kpId}`}>查看这个知识点的全部记录 →</Link>{!e.isCorrect&&<Link className="primary-link" to={`/reviews?subject=${e.subjectCode}&kpId=${e.kpId}`}>查看相关复习建议</Link>}</section></>}</article>}
