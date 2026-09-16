import { Link } from 'react-router-dom'
import type { CalendarDay, CalendarSubject, PointCounts } from '../../api/knowledgeTypes'

export function ChartLegend() {return <div className="chart-legend"><span className="legend-item"><i className="bar-legend-sq sq-total"/>答题总数</span><span className="legend-item"><i className="bar-legend-sq sq-wrong"/>错题数</span></div>}
export function SubjectBars({items,date,daily=false}:{items:CalendarSubject[];date?:string;daily?:boolean}) {
 const max=Math.max(2,Math.ceil(Math.max(0,...items.map(i=>i.total))*1.2))
 const step=Math.max(1,Math.ceil(max/4)); const ceiling=Math.ceil(max/step)*step
 const ticks=Array.from({length:ceiling/step+1},(_,i)=>ceiling-i*step)
 return <div className={daily?'daily-bar-container':'subject-bar-chart-area'}>
  <div className={daily?'daily-y-axis':'bar-y-axis'}>{ticks.map(t=><span key={t}>{t}</span>)}</div>
  <div className={daily?'daily-bars-row':'bar-columns-flex'}>{items.map(item=>{
   const height=item.total/ceiling*100, ratio=item.total?item.wrong/item.total*100:0
   const href=`/wrongs?subject=${encodeURIComponent(item.code)}${date?`&from=${date}&to=${date}`:''}`
   return <Link key={item.code} to={href} className={daily?'daily-bar-item':'bar-col-item'} aria-label={`${item.name}，错题 ${item.wrong} 次，总作答 ${item.total} 次`} title={`${item.name}：错题 ${item.wrong} / 总数 ${item.total}`}>
    {!daily&&<div className="bar-num-top"><span className="num-wrong">{item.wrong}</span><span className="num-sep">/</span><span className="num-total">{item.total}</span></div>}
    <div className={daily?'daily-bar-track':'bar-track'}>
     {daily&&<div className="daily-bar-num" style={{bottom:`calc(${height}% + 8px)`}}><span className="num-wrong">{item.wrong}</span><span className="num-sep">/</span><span className="num-total">{item.total}</span></div>}
     <div className={daily?'daily-bar-stem':'bar-total-stem'} style={{height:`${height}%`}}><div className={daily?'daily-bar-wrong':'bar-wrong-fill'} style={{height:`${ratio}%`}}/></div>
    </div><span className={daily?'daily-bar-name':'bar-label-bottom'}>{item.name}</span>
   </Link>
  })}</div>
 </div>
}
export const categoryLabels:Record<string,string>={complete:'完整掌握',partial:'部分掌握',weak:'待巩固',learning:'练习中',unpracticed:'未练习',unknown:'状态待确认',review_due:'到期复习',mastered:'有已掌握能力',all:'全部目录'}
const categories=[['complete','#00e5a3'],['partial','#38bdf8'],['weak','#ffb703'],['learning','#8b5cf6'],['unpracticed','#536078'],['unknown','#c4b5fd']] as const
export function MasteryDonut({counts,total}:{counts:PointCounts;total:number}) {
 let offset=0; const circumference=2*Math.PI*58
 return <div className="donut-chart-layout"><div className="donut-graphic-wrapper"><svg viewBox="0 0 160 160" className="donut-svg" role="img" aria-label={`知识掌握结构，共 ${total} 个知识点`}><circle cx="80" cy="80" r="58" fill="none" stroke="#ffffff12" strokeWidth="20"/>{categories.map(([key,color])=>{const length=total?counts[key]/total*circumference:0;const start=offset;offset+=length;return <circle key={key} cx="80" cy="80" r="58" fill="none" stroke={color} strokeWidth="20" strokeDasharray={`${length} ${circumference-length}`} strokeDashoffset={-start}/>})}</svg><div className="donut-center-info"><strong className="donut-center-num">{total}</strong><span className="donut-center-sub">总知识项</span></div></div><div className="donut-legend-col mastery-category-legend">{categories.map(([key,color])=><Link key={key} to={`/library?state=${key}`} className="donut-legend-row"><i className="legend-indicator" style={{background:color}}/><div className="legend-row-text"><span className="legend-name">{categoryLabels[key]}</span><strong className="legend-val" style={{color}}>{counts[key]} <small>({total?Math.round(counts[key]/total*100):0}%)</small></strong></div></Link>)}</div></div>
}
// Midpoint cubic segments stay within adjacent observations, so smoothing cannot invent a peak.
function curve(points:{x:number;y:number}[]) {return points.map((p,i)=>i?`C ${(points[i-1].x+p.x)/2} ${points[i-1].y}, ${(points[i-1].x+p.x)/2} ${p.y}, ${p.x} ${p.y}`:`M ${p.x} ${p.y}`).join(' ')}
export function TrendChart({days}:{days:CalendarDay[]}) {
 const max=Math.max(4,Math.ceil(Math.max(0,...days.flatMap(d=>[d.mastered,d.wrong]))/4)*4)
 const points=(key:'mastered'|'wrong')=>days.map((d,i)=>({x:40+i*300/Math.max(1,days.length-1),y:150-d[key]/max*125}))
 return <div className="trend-svg-container"><svg viewBox="0 0 380 180" className="trend-svg" role="img" aria-label="近7日新增掌握与历史错误" preserveAspectRatio="none">{Array.from({length:5},(_,i)=><g key={i}><line x1="32" y1={25+i*31.25} x2="350" y2={25+i*31.25} stroke="#ffffff14" strokeDasharray="3 3"/><text x="25" y={29+i*31.25} textAnchor="end" className="axis-label">{max-i*max/4}</text></g>)}{(['mastered','wrong'] as const).map(key=><g key={key}><path d={curve(points(key))} fill="none" stroke={key==='mastered'?'#00e5a3':'#ff5c8a'} strokeWidth="2.5"/>{points(key).map((p,i)=><Link key={days[i].date} to={key==='mastered'?`/library?masteredOn=${days[i].date}`:`/wrongs?from=${days[i].date}&to=${days[i].date}`} aria-label={`${days[i].date} ${key==='mastered'?'新增掌握':'错题'} ${days[i][key]}`}><circle cx={p.x} cy={p.y} r="5" fill={key==='mastered'?'#00e5a3':'#ff5c8a'}><title>{days[i].date}：{days[i][key]}</title></circle></Link>)}</g>)}{days.map((d,i)=><text key={d.date} x={40+i*300/Math.max(1,days.length-1)} y="172" textAnchor="middle" className="axis-label">{Number(d.date.slice(5,7))}/{Number(d.date.slice(8))}</text>)}</svg></div>
}
