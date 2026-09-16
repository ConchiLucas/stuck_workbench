import {render,screen,fireEvent,waitFor,cleanup} from '@testing-library/react'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {MemoryRouter} from 'react-router-dom'
import {afterEach,expect,it,vi} from 'vitest'
import {LogicTasksPage} from './LogicTasksPage'
afterEach(()=>{cleanup();vi.unstubAllGlobals()})
const example={kind:'classify',prompt:'哪个不属于这一类？',rule:{type:'odd-one-out',dimension:'kingdom',inGroup:'animal',explain:'动物'},objects:[{id:'cat',caption:'猫',glyph:'cat',fill:'#f59e0b'},{id:'car',caption:'汽车',glyph:'car',fill:'#e11d48'}],options:['cat','car'],answerId:'car'}
const task={id:1,title:'逻辑测试',count:1,types:['classify'],createdAt:'2026-09-15',items:[{id:'q1',kind:'classify',skillCode:'classify',targetId:1,sourceId:1,sourceTable:'knowledge_points',moduleCode:'playground',sourceContentHash:'abc123',example,mediaSHA256:{}}]}
function page(){render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><LogicTasksPage/></MemoryRouter></QueryClientProvider>)}
it('opens saved question, previews locally and restores focus on Escape',async()=>{
 const fetcher=vi.fn(async(url:string,_init?:RequestInit)=>({ok:true,status:200,json:async()=>url.endsWith('/1')?task:{items:[task]}} as Response));vi.stubGlobal('fetch',fetcher);page();const trigger=await screen.findByRole('button',{name:/逻辑测试/});trigger.focus();fireEvent.click(trigger);await screen.findByRole('button',{name:/汽车/});fireEvent.click(screen.getByRole('button',{name:/汽车/}));expect(screen.getByRole('status')).toHaveTextContent('答对了');expect(fetcher.mock.calls.every(c=>c.length===1||!(c[1] as RequestInit)?.method||(c[1] as RequestInit)?.method==='GET')).toBe(true);fireEvent(await screen.findByRole('dialog'),new Event('cancel',{bubbles:true,cancelable:true}));await waitFor(()=>expect(screen.queryByRole('dialog')).not.toBeInTheDocument());expect(trigger).toHaveFocus();expect(screen.queryByText('发布')).not.toBeInTheDocument()
})
it('generates selected types then loads persisted task detail',async()=>{
 const calls:{url:string;init?:RequestInit}[]=[];vi.stubGlobal('fetch',vi.fn(async(url:string,init?:RequestInit)=>{calls.push({url,init});return {ok:true,status:200,json:async()=>init?.method==='POST'||url.endsWith('/1')?task:{items:[]}} as Response}));page();await screen.findByText('暂无逻辑出题任务。');fireEvent.click(screen.getByRole('button',{name:'生成逻辑题目'}));fireEvent.click(screen.getByRole('button',{name:'生成并保存'}));await screen.findByRole('button',{name:/汽车/});const post=calls.find(c=>c.init?.method==='POST');expect(JSON.parse(post!.init!.body as string)).toMatchObject({count:12,types:['pattern','classify','order','shape_reason','diff','compare']});expect(calls.some(c=>c.url.endsWith('/1')&&c.init?.method==='GET')).toBe(true)
})
it('shows load failure without claiming an empty list',async()=>{vi.stubGlobal('fetch',vi.fn(async()=>({ok:false,status:503,json:async()=>({error:'逻辑任务服务未配置'})})));page();expect(await screen.findByRole('alert')).toHaveTextContent('逻辑任务服务未配置');expect(screen.queryByText('暂无逻辑出题任务。')).not.toBeInTheDocument()}
)
