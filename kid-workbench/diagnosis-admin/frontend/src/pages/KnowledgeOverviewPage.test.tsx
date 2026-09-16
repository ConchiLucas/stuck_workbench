import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, expect, it, vi } from 'vitest'
import { App } from '../App'

const counts = { complete: 1, partial: 1, weak: 0, learning: 0, unpracticed: 2, unknown: 0, reviewDue: 1 }
const summary = {child:{id:1,name:'测试孩子'},totalCount:4,practicedCount:2,wrongCount:1,pointCounts:counts,subjects:[{code:'literacy',name:'识字',total:4,practicedCount:2,attemptsCount:3,wrongCount:1,pointCounts:counts}]}
const day = {date:'2026-09-13',mastered:1,total:3,wrong:1,subjects:[{code:'literacy',name:'识字',mastered:1,total:3,wrong:1}]}
const calendar = {today:'2026-09-13',timezone:'Asia/Shanghai',month:'2026-09',days:[day],trend:[day],masteryDateUnknownCount:0}
afterEach(()=>{cleanup();vi.restoreAllMocks();localStorage.clear();sessionStorage.clear()})
function mount(path='/'){render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={[path]}><App/></MemoryRouter></QueryClientProvider>)}
it('uses real zero-capable summaries and links to exact mastery/date filters',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).includes('/calendar')?calendar:summary)))
 mount('/?month=2026-09&date=2026-09-13')
 const complete=await screen.findByRole('link',{name:'已掌握 1 个，查看完整掌握'})
 expect(complete).toHaveAttribute('href','/library?state=complete')
 expect(screen.queryByText('82')).not.toBeInTheDocument()
 expect(screen.queryByText('让知识点亮未来')).not.toBeInTheDocument()
 expect(screen.getByRole('link',{name:'今日掌握 1 个'})).toHaveAttribute('href','/library?masteredOn=2026-09-13')
 expect(screen.getByRole('link',{name:'当天错题 1 次'})).toHaveAttribute('href','/wrongs?from=2026-09-13&to=2026-09-13')
})
it('fetches another month and distinguishes no practice from zero errors',async()=>{
 const calls:string[]=[]
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>{calls.push(String(input));return new Response(JSON.stringify(String(input).includes('/calendar')?calendar:summary))})
 mount('/?month=2026-09&date=2026-09-13')
 await screen.findByRole('link',{name:'当天错题 1 次'})
 fireEvent.click(screen.getByRole('button',{name:'上一月'}))
 await waitFor(()=>expect(calls.some(u=>u.includes('month=2026-08'))).toBe(true))
 fireEvent.click(screen.getByRole('button',{name:'今天'}))
 fireEvent.click(await screen.findByRole('button',{name:'2026年9月12日'}))
 expect(screen.getByText('这一天没有作答记录')).toBeInTheDocument()
 expect(screen.getByLabelText('当天正确率')).toHaveTextContent('—')
})
it('keeps summary visible when calendar fails and never invents daily counts',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>String(input).includes('/calendar')?new Response(JSON.stringify({error:{message:'日历暂不可读取'}}),{status:503}):new Response(JSON.stringify(summary)))
 mount('/?month=2026-09')
 await screen.findByText('日历暂不可读取')
 expect(screen.getByRole('link',{name:'已掌握 1 个，查看完整掌握'})).toBeInTheDocument()
 expect(screen.queryByRole('link',{name:'今日掌握 6 个'})).not.toBeInTheDocument()
})
it('preserves old bookmarked library filters',async()=>{
 const calls:string[]=[]
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>{calls.push(String(input));return new Response(JSON.stringify(String(input).endsWith('/summary')?summary:{items:[],hasMore:false}))})
 mount('/?subject=literacy&state=weak')
 await screen.findByRole('heading',{name:'能力档案'})
 expect(calls.some(url=>url.includes('/points?subject=literacy&state=weak'))).toBe(true)
})
