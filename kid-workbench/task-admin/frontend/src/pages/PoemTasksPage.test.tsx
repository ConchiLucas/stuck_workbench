import {render,screen,fireEvent,waitFor,cleanup} from '@testing-library/react'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {MemoryRouter} from 'react-router-dom'
import {afterEach,expect,it,vi} from 'vitest'
import {PoemTasksPage} from './PoemTasksPage'
afterEach(()=>{cleanup();vi.unstubAllGlobals()})
const task={id:1,title:'古诗测试',count:1,types:['title'],createdAt:'2026-09-15',items:[{id:'q1',kind:'title',skillCode:'title',targetId:1,sourceId:1,sourceTable:'knowledge_points',moduleCode:'poem50',sourceContentHash:'abc123',example:{kind:'title',prompt:'这首诗叫什么？',line:'床前明月光',workId:'pm001',options:[{id:'pm002',label:'春晓'},{id:'pm001',label:'静夜思'},{id:'pm003',label:'咏鹅'},{id:'pm004',label:'悯农'}],answerId:'pm001'},mediaSHA256:{}}]}
function page(){render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><PoemTasksPage/></MemoryRouter></QueryClientProvider>)}
it('opens saved question, previews locally and restores focus on Escape',async()=>{
 const fetcher=vi.fn(async(url:string,_init?:RequestInit)=>({ok:true,status:200,json:async()=>url.endsWith('/1')?task:{items:[task]}} as Response));vi.stubGlobal('fetch',fetcher);page();const trigger=await screen.findByRole('button',{name:/古诗测试/});trigger.focus();fireEvent.click(trigger);await screen.findByRole('button',{name:'静夜思'});fireEvent.click(screen.getByRole('button',{name:'静夜思'}));expect(screen.getByText('答对了')).toBeInTheDocument();expect(fetcher.mock.calls.every(c=>c.length===1||!(c[1] as RequestInit)?.method||(c[1] as RequestInit)?.method==='GET')).toBe(true);fireEvent(await screen.findByRole('dialog'),new Event('cancel',{bubbles:true,cancelable:true}));await waitFor(()=>expect(screen.queryByRole('dialog')).not.toBeInTheDocument());expect(trigger).toHaveFocus();expect(screen.queryByText('发布')).not.toBeInTheDocument()
})
it('generates selected types then loads persisted task detail',async()=>{
 const calls:{url:string;init?:RequestInit}[]=[];vi.stubGlobal('fetch',vi.fn(async(url:string,init?:RequestInit)=>{calls.push({url,init});return {ok:true,status:200,json:async()=>init?.method==='POST'||url.endsWith('/1')?task:{items:[]}} as Response}));page();await screen.findByText('暂无古诗出题任务。');fireEvent.click(screen.getByRole('button',{name:'生成古诗题目'}));fireEvent.click(screen.getByRole('button',{name:'生成并保存'}));await screen.findByRole('button',{name:'静夜思'});const post=calls.find(c=>c.init?.method==='POST');expect(JSON.parse(post!.init!.body as string)).toMatchObject({count:12,types:['title','fill','couplet','recite']});expect(calls.some(c=>c.url.endsWith('/1')&&c.init?.method==='GET')).toBe(true)
})
it('shows load failure without claiming an empty list',async()=>{vi.stubGlobal('fetch',vi.fn(async()=>({ok:false,status:503,json:async()=>({error:'古诗任务服务未配置'})})));page();expect(await screen.findByRole('alert')).toHaveTextContent('古诗任务服务未配置');expect(screen.queryByText('暂无古诗出题任务。')).not.toBeInTheDocument()}
)
