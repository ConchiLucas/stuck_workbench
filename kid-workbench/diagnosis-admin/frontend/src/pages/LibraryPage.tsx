import { useInfiniteQuery,useQuery } from '@tanstack/react-query'
import { Link,useParams,useSearchParams } from 'react-router-dom'
import { knowledge,queryString } from '../api/knowledge'
import type { Page,Point,Summary } from '../api/knowledgeTypes'
import { getChildId } from '../store/childStore'
import { LoadState,MoreButton,PointRow } from '../components/knowledge/Shared'
import { categoryLabels } from '../components/knowledge/OverviewCharts'

export function LibraryPage(){
 const child=getChildId(),{code}=useParams(),[params,setParams]=useSearchParams()
 const subject=code||params.get('subject')||'', state=params.get('state')||'',q=params.get('q')||'',module=params.get('module')||'',masteredOn=params.get('masteredOn')||''
 const summary=useQuery({queryKey:['knowledge','summary',child],queryFn:()=>knowledge<Summary>('/summary')})
 const points=useInfiniteQuery({queryKey:['knowledge','points',child,subject,state,q,module,masteredOn],initialPageParam:'',queryFn:({pageParam})=>knowledge<Page<Point>>('/points'+queryString({subject,state,q,module,masteredOn,cursor:pageParam})),getNextPageParam:last=>last.hasMore?last.nextCursor:undefined})
 const update=(k:string,v:string)=>{const next=new URLSearchParams(params);if(v)next.set(k,v);else next.delete(k);setParams(next,{replace:true})}
 const modules=useQuery({queryKey:['knowledge','modules',child,subject],queryFn:()=>knowledge<{items:{code:string;name:string;subjectCode:string}[]}>('/modules'+queryString({subject}))})
 const data=summary.data, scope=subject?data?.subjects.find(s=>s.code===subject):data,rows=points.data?.pages.flatMap(p=>p.items)||[]
 const subjectLink=(next:string)=>{const p=new URLSearchParams(params);p.delete('subject');p.delete('module');return (next?`/subjects/${next}`:'/library')+(p.size?`?${p}`:'')}
 return <article><header className="page-head"><div><p className="eyebrow">孩子的知识档案</p><h1>能力档案</h1><p className="lede">完整掌握与部分掌握分别查看；到期复习不代表已经不会。</p></div><Link className="text-link" to={`/wrongs${queryString({subject})}`}>去看具体错题 ↗</Link></header>
 {scope&&<div className="library-counts"><button onClick={()=>update('state','')} className={!state?'active':''}><strong>{scope.practicedCount}</strong> 个已练知识点</button><button onClick={()=>update('state','complete')} className={state==='complete'?'active':''}><strong>{scope.pointCounts?.complete??'—'}</strong> 个完整掌握</button><Link to={`/wrongs${queryString({subject})}`}><strong>{scope.wrongPointCount}</strong> 个有过错误</Link></div>}
 <div className="library-layout"><aside className="subject-nav" aria-label="学科"><Link to={subjectLink('')} className={!subject?'active':''}>全部学科</Link>{data?.subjects.filter(s=>s.code!=='game').map(s=><Link key={s.code} to={subjectLink(s.code)} className={subject===s.code?'active':''}><span>{s.name}</span><small>{s.practicedCount} 已练</small></Link>)}<p>仅整理已记录的练习。示例试玩不计入。</p></aside><section className="library-content"><div className="filters"><label className="search-field"><span>搜索知识</span><input value={q} maxLength={80} placeholder="搜索字、词或知识点" onChange={e=>update('q',e.target.value)}/></label><label><span>模块</span><select value={module} onChange={e=>update('module',e.target.value)}><option value="">全部模块</option>{modules.data?.items?.map(m=><option value={m.code} key={`${m.subjectCode}:${m.code}`}>{m.name}</option>)}</select></label><label><span>查看范围</span><select value={state} onChange={e=>update('state',e.target.value)}><option value="">全部已练</option>{Object.entries(categoryLabels).map(([key,label])=><option value={key} key={key}>{label}</option>)}</select></label><label><span>首次完整掌握日期</span><input type="date" value={masteredOn} onChange={e=>update('masteredOn',e.target.value)}/></label>{params.size>0&&<button onClick={()=>setParams({})}>清除筛选</button>}</div>
 {masteredOn&&<p className="notice">查看 {masteredOn} 首次完整掌握的知识点；缺少可靠历史日期的记录不在此列表。</p>}
 <LoadState loading={points.isPending} error={points.error}/>{!points.isPending&&!points.error&&rows.length===0&&<div className="empty-panel"><h2>没有找到符合条件的知识点</h2><p>可切换学科或查看范围；没有记录不代表已经掌握。</p></div>}
 <div className="knowledge-list">{rows.map(p=><PointRow key={p.kpId} point={p}/>)}</div><MoreButton more={!!points.hasNextPage} loading={points.isFetchingNextPage} onClick={()=>void points.fetchNextPage()}/></section></div></article>
}
