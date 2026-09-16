import { useQuery } from '@tanstack/react-query'
import { BookOpen, Check, CheckCircle2, ChevronLeft, ChevronRight, Lightbulb, Target, Trophy, X, XCircle } from 'lucide-react'
import { Link, Navigate, useSearchParams } from 'react-router-dom'
import { knowledge } from '../api/knowledge'
import type { CalendarDay, KnowledgeCalendar, Summary } from '../api/knowledgeTypes'
import { ChartLegend, MasteryDonut, SubjectBars, TrendChart } from '../components/knowledge/OverviewCharts'
import { LoadState } from '../components/knowledge/Shared'
import { dateCaption, monthDates, shanghaiToday, shiftDay, shiftMonth } from '../lib/knowledgeDates'
import { getChildId } from '../store/childStore'

const emptyDay=(date:string):CalendarDay=>({date,mastered:0,total:0,wrong:0,subjects:[]})
export function KnowledgeOverviewPage() {
 const child=getChildId(), [params,setParams]=useSearchParams()
 const redirect=['subject','module','q','state'].some(key=>params.has(key))
 const summary=useQuery({queryKey:['knowledge','summary',child],queryFn:()=>knowledge<Summary>('/summary')})
 const clientToday=shanghaiToday()
 const requestedMonth=params.get('month')||''
 const month=/^\d{4}-(0[1-9]|1[0-2])$/.test(requestedMonth)?requestedMonth:clientToday.slice(0,7)
 const calendar=useQuery({queryKey:['knowledge','calendar',child,month],queryFn:()=>knowledge<KnowledgeCalendar>(`/calendar?month=${month}`),enabled:!redirect})
 const today=calendar.data?.today||clientToday
 const dates=monthDates(month), requestedDate=params.get('date')||''
 const selectedDate=dates.includes(requestedDate)?requestedDate:month===today.slice(0,7)?today:dates[0]
 const selected=calendar.data?.days.find(d=>d.date===selectedDate)||emptyDay(selectedDate)
 const todayMastered=calendar.data?.trend.find(d=>d.date===today)?.mastered??0
 const subjects=(summary.data?.subjects||[]).filter(s=>s.code!=='game')
 const currentSubjects=subjects.map(s=>selected.subjects.find(d=>d.code===s.code)||{code:s.code,name:s.name,mastered:0,total:0,wrong:0})
 const changeDate=(date:string)=>setParams({month:date.slice(0,7),date})
 const changeMonth=(offset:number)=>{const next=shiftMonth(month,offset);changeDate(next===today.slice(0,7)?today:`${next}-01`)}
 const dayLink=`/wrongs?from=${selectedDate}&to=${selectedDate}`
 const knownSummary=summary.data?.pointCounts&&summary.data.totalCount!==undefined
 if(redirect)return <Navigate to={`/library?${params}`} replace/>
 return <article className="knowledge-overview-v3" aria-label="知识总览">
  <LoadState loading={summary.isPending} error={summary.error}/>
  {summary.data&&!knownSummary&&<p role="alert">掌握统计暂不可读取，请刷新后重试。</p>}
  {knownSummary&&<section className="overview-metric-grid" aria-label="核心指标汇总">
   <Link className="overview-metric-card" to="/library?state=all" aria-label={`总知识项 ${summary.data!.totalCount} 个`}><div className="metric-icon-bubble bubble-purple"><BookOpen size={24}/></div><div className="metric-text-group"><span className="metric-label">总知识项</span><strong className="metric-val val-white">{summary.data!.totalCount}</strong></div></Link>
   <Link className="overview-metric-card" to="/library?state=complete" aria-label={`已掌握 ${summary.data!.pointCounts!.complete} 个，查看完整掌握`}><div className="metric-icon-bubble bubble-green"><Check size={26}/></div><div className="metric-text-group"><span className="metric-label">已掌握</span><strong className="metric-val val-green">{summary.data!.pointCounts!.complete}</strong></div></Link>
   <Link className="overview-metric-card" to={`/library?masteredOn=${today}`} aria-label={calendar.data?`今日掌握 ${todayMastered} 个`:'今日掌握暂不可读取'}><div className="metric-icon-bubble bubble-blue"><Lightbulb size={24}/></div><div className="metric-text-group"><span className="metric-label">今日掌握</span><strong className="metric-val val-blue">{calendar.data?todayMastered:'—'}</strong></div></Link>
   <Link className="overview-metric-card" to="/wrongs" aria-label={`累计错题 ${summary.data!.wrongCount} 次`}><div className="metric-icon-bubble bubble-pink"><X size={24}/></div><div className="metric-text-group"><span className="metric-label">累计错题</span><strong className="metric-val val-pink">{summary.data!.wrongCount}</strong></div></Link>
  </section>}
  <section className="overview-chart-row" aria-label="掌握情况与学科结构">
   <div className="chart-panel panel-trend"><div className="panel-header"><h2 className="panel-title">近7日掌握情况</h2><div className="chart-legend"><span className="legend-item"><i className="dot dot-green"/>新增掌握</span><span className="legend-item"><i className="dot dot-pink"/>错题</span></div></div><LoadState loading={calendar.isPending} error={calendar.error}/>{calendar.data&&<TrendChart days={calendar.data.trend}/>}</div>
   <div className="chart-panel panel-structure"><div className="panel-header"><h2 className="panel-title">知识掌握结构</h2></div>{knownSummary&&<MasteryDonut counts={summary.data!.pointCounts!} total={summary.data!.totalCount!}/>}</div>
   <div className="chart-panel panel-subject-bars"><div className="panel-header"><h2 className="panel-title">错题学科分布</h2><ChartLegend/></div>{knownSummary&&<SubjectBars items={subjects.map(s=>({code:s.code,name:s.name,mastered:s.pointCounts?.complete||0,total:s.attemptsCount||0,wrong:s.wrongCount}))}/>}</div>
  </section>
  <section className="overview-bottom-split">
   <div className="calendar-panel"><div className="calendar-header-row"><h2 className="panel-title">掌握与错题日历</h2><div className="calendar-month-controls"><div className="month-pill-switch"><button type="button" className="cal-nav-btn" aria-label="上一月" disabled={month<='2000-01'} onClick={()=>changeMonth(-1)}><ChevronLeft size={16}/></button><span className="current-month-text">{Number(month.slice(0,4))}年{Number(month.slice(5))}月</span><button type="button" className="cal-nav-btn" aria-label="下一月" disabled={month>=today.slice(0,7)} onClick={()=>changeMonth(1)}><ChevronRight size={16}/></button></div><button type="button" className="cal-today-btn" onClick={()=>changeDate(today)}>今天</button></div><div className="calendar-legend-box"><span className="cal-leg-item"><span className="leg-badge-mastered">+N掌握</span>新增能力</span><span>错题比例</span><span>做题总数</span></div></div>
    {calendar.isError?<p className="calendar-unavailable">日历暂不可用，请在上方重试。</p>:calendar.isPending?<p role="status">正在读取日历…</p>:<div className="cal-table"><div className="cal-week-head">{['周一','周二','周三','周四','周五','周六','周日'].map(d=><span key={d}>{d}</span>)}</div><div className="cal-days-grid">
     {Array.from({length:(new Date(`${month}-01T00:00:00Z`).getUTCDay()+6)%7},(_,i)=><div key={`blank-${i}`} className="cal-day-cell cell-empty"/>)}
     {dates.map(date=>{const day=calendar.data?.days.find(d=>d.date===date)||emptyDay(date);const practiced=day.total>0;const ratio=practiced?day.wrong/day.total*100:0;const future=date>today;return <button key={date} type="button" className={`cal-day-cell${date===selectedDate?' cell-selected':''}${day.wrong?' cell-many-wrongs':''}`} aria-label={`${Number(date.slice(0,4))}年${dateCaption(date)}`} aria-pressed={date===selectedDate} disabled={future} onClick={()=>changeDate(date)} title={future?'尚未到这一天':`${date} · 新增掌握 ${day.mastered} · ${day.wrong} 次错误 / ${day.total} 次作答`}><div className="cal-cell-top"><span className="day-number">{Number(date.slice(8))}</span>{day.mastered>0&&<span className="day-mastered-badge">+{day.mastered}掌握</span>}</div>{practiced?<div className="cal-practice-wrap"><div className="cal-practice-labels"><span className={day.wrong?'cal-lbl-wrong':'cal-lbl-clean'}>{day.wrong?`${day.wrong}错`:'全对'}</span><span className="cal-lbl-total">{day.total}次</span></div><div className="cal-progress-track"><div className="cal-progress-wrong" style={{width:`${ratio}%`}}/><div className="cal-progress-correct" style={{width:`${100-ratio}%`}}/></div></div>:<span className="cal-no-practice">{future?'':'未练习'}</span>}</button>})}
    </div></div>}
    {!!calendar.data?.masteryDateUnknownCount&&<p className="coverage-note">{calendar.data.masteryDateUnknownCount} 个已掌握知识点缺少可靠的首次完整掌握日期，未计入每日新增。</p>}
   </div>
   <div className="daily-overview-panel"><div className="daily-header-row"><h2 className="panel-title">{dateCaption(selectedDate)} · 当天概况</h2><div className="daily-arrows"><button className="arrow-btn" aria-label="前一天" onClick={()=>changeDate(shiftDay(selectedDate,-1))}><ChevronLeft size={16}/></button><button className="arrow-btn" aria-label="后一天" disabled={selectedDate>=today} onClick={()=>changeDate(shiftDay(selectedDate,1))}><ChevronRight size={16}/></button></div></div>
    <div className="daily-stats-row"><Link to={`/library?masteredOn=${selectedDate}`} className="daily-stat-box box-green" aria-label={calendar.data?`当天新增掌握 ${selected.mastered} 个`:'当天新增掌握暂不可读取'}><CheckCircle2 size={20} className="icon-green"/><div className="stat-value-wrap"><strong className="stat-num">{calendar.data?selected.mastered:'—'}</strong><span className="stat-lbl">新增掌握</span></div></Link><Link to={dayLink} className="daily-stat-box box-pink" aria-label={calendar.data?`当天错题 ${selected.wrong} 次`:'当天错题暂不可读取'}><XCircle size={20} className="icon-pink"/><div className="stat-value-wrap"><strong className="stat-num">{calendar.data?selected.wrong:'—'}</strong><span className="stat-lbl">错题</span></div></Link><div className="daily-stat-box box-blue" aria-label="当天正确率"><Target size={20} className="icon-blue"/><div className="stat-value-wrap"><strong className="stat-num">{calendar.data&&selected.total?`${Math.round((selected.total-selected.wrong)/selected.total*100)}%`:'—'}</strong><span className="stat-lbl">正确率</span></div></div></div>
    <div className="daily-subject-section"><div className="daily-subject-header"><h3 className="section-subtitle">当日学科分布</h3><ChartLegend/></div>{calendar.data&&<SubjectBars daily items={currentSubjects} date={selectedDate}/>}</div>
    <Link to={selected.total?dayLink:`/library?masteredOn=${selectedDate}`} className="cheer-banner-card"><div className="cheer-left"><Trophy size={18}/><span className="cheer-text">{!calendar.data?'当天记录暂不可用':selected.total?selected.wrong?`查看当天错题 · ${selected.wrong} 次错误`:'当天没有错题':'这一天没有作答记录'}</span></div><ChevronRight size={18}/></Link>
   </div>
  </section>
 </article>
}
