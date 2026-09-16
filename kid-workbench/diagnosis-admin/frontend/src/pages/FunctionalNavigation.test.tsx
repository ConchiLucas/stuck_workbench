import {cleanup,fireEvent,render,screen,waitFor} from '@testing-library/react'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {MemoryRouter} from 'react-router-dom'
import {afterEach,it,expect,vi} from 'vitest'
import {App} from '../App'
const counts={complete:2,partial:1,weak:0,learning:0,unpracticed:0,unknown:0,reviewDue:1}
const summary={child:{id:1},totalCount:3,pointCounts:counts,subjects:[{code:'literacy',name:'识字',practicedCount:3,wrongPointCount:1,pointCounts:counts}],practicedCount:9,wrongPointCount:4}
const evidence={attemptId:7,kpId:4,title:'山',subjectCode:'literacy',subjectName:'识字',skillLabel:'看字选义',moduleName:'一组',occurredAt:'2026-09-12T16:30:00Z',costMs:1000,isCorrect:false,question:null,response:{},followUpState:'no_later_practice',evidenceReasonCodes:[]}
afterEach(()=>{cleanup();vi.restoreAllMocks();localStorage.clear();sessionStorage.clear()})
function mount(path:string){render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={[path]}><App/></MemoryRouter></QueryClientProvider>)}
it('sends date/unresolved filters and returns from an attempt to the same filters',async()=>{
 const calls:string[]=[]
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>{const u=String(input);calls.push(u);return new Response(JSON.stringify(u.endsWith('/summary')?summary:u.endsWith('/attempts/7')?evidence:{items:[evidence],hasMore:false}))})
 mount('/wrongs?subject=literacy&from=2026-09-13&to=2026-09-13&followUpState=needs_practice')
 fireEvent.click(await screen.findByRole('link',{name:'完整作答现场 ↗'}))
 const back=await screen.findByRole('link',{name:'返回列表'})
 expect(back).toHaveAttribute('href','/wrongs?subject=literacy&from=2026-09-13&to=2026-09-13&followUpState=needs_practice')
 fireEvent.click(back)
 expect(await screen.findByLabelText('开始日期')).toHaveValue('2026-09-13')
 expect(screen.getByLabelText('后续表现')).toHaveValue('needs_practice')
 expect(calls.some(u=>u.includes('from=2026-09-13')&&u.includes('followUpState=needs_practice'))).toBe(true)
})
it('keeps subject counts scoped and sends complete mastery with date filter',async()=>{
 const calls:string[]=[]
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>{calls.push(String(input));return new Response(JSON.stringify(String(input).endsWith('/summary')?summary:{items:[],hasMore:false}))})
 mount('/subjects/literacy?state=complete&masteredOn=2026-09-13')
 expect(await screen.findByRole('button',{name:'2 个完整掌握'})).toBeInTheDocument()
 expect(screen.getByRole('button',{name:'3 个已练知识点'})).toBeInTheDocument()
 await waitFor(()=>expect(calls.some(u=>u.includes('masteredOn=2026-09-13')&&u.includes('state=complete'))).toBe(true))
})
it('pinyin wrong filters only offer its four actual types and keep all-subject options',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).endsWith('/summary')?{...summary,subjects:[{code:'pinyin',name:'拼音'}]}:{items:[],hasMore:false})))
 mount('/wrongs?subject=pinyin');await screen.findByRole('option',{name:'拼音'});
 expect(screen.queryByRole('option',{name:'看字选义图'})).not.toBeInTheDocument();
 for(const name of ['听音选字母','字中找拼音','看形认读','声韵拼读']) expect(screen.getByRole('option',{name})).toBeInTheDocument();
 fireEvent.change(screen.getByLabelText('学科'),{target:{value:''}});expect(screen.getByRole('option',{name:'看字选义图'})).toBeInTheDocument();
})
it('english wrong filters only offer the five app types',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).endsWith('/summary')?{...summary,subjects:[{code:'english',name:'英语'}]}:{items:[],hasMore:false})))
 mount('/wrongs?subject=english');await screen.findByRole('option',{name:'英语'});
 expect(screen.queryByRole('option',{name:'看字选义图'})).not.toBeInTheDocument();
 for(const name of ['听音选词','看图选词','组句子','写单词','读一读']) expect(screen.getByRole('option',{name})).toBeInTheDocument();
})
it('phrase wrong filters only offer the four app types',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).endsWith('/summary')?{...summary,subjects:[{code:'phrase',name:'英语短句'}]}:{items:[],hasMore:false})))
 mount('/wrongs?subject=phrase');await screen.findByRole('option',{name:'英语短句'});
 expect(screen.queryByRole('option',{name:'看字选义图'})).not.toBeInTheDocument();
 for(const name of ['听一听','选句子','什么时候说','问与答']) expect(screen.getByRole('option',{name})).toBeInTheDocument();
})
it('chengyu wrong filters only offer the four app types',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).endsWith('/summary')?{...summary,subjects:[{code:'chengyu',name:'成语'}]}:{items:[],hasMore:false})))
 mount('/wrongs?subject=chengyu');await screen.findByRole('option',{name:'成语'});
 expect(screen.queryByRole('option',{name:'看字选义图'})).not.toBeInTheDocument();
 for(const name of ['听释义','选成语','看拼音','看句子']) expect(screen.getByRole('option',{name})).toBeInTheDocument();
})
it('logic wrong filters only offer the six app types',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).endsWith('/summary')?{...summary,subjects:[{code:'logic',name:'逻辑'}]}:{items:[],hasMore:false})))
 mount('/wrongs?subject=logic');await screen.findByRole('option',{name:'逻辑'});
 expect(screen.queryByRole('option',{name:'看字选义图'})).not.toBeInTheDocument();
 for(const name of ['找规律','分类','排序','图形推理','找不同','比较']) expect(screen.getByRole('option',{name})).toBeInTheDocument();
})
