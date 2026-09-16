import {render,screen,fireEvent,waitFor,cleanup} from '@testing-library/react'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {MemoryRouter} from 'react-router-dom'
import {afterEach,expect,it,vi} from 'vitest'
import {ScienceTasksPage} from './ScienceTasksPage'
afterEach(()=>{cleanup();vi.unstubAllGlobals()})
const task={id:1,title:'科普测试',count:1,types:['choice'],createdAt:'2026-09-14',items:[{id:'q1',kind:'choice',skillCode:'choice',targetId:1,sourceId:1,sourceTable:'knowledge_points',moduleCode:'observe',sourceContentHash:'abc123',example:{kind:'choice',prompt:'哪种动物的脚掌最适合在水里游泳？',options:[{id:'cat',label:'猫'},{id:'duck',label:'鸭子'},{id:'rabbit',label:'兔子'}],answerId:'duck'},mediaSHA256:{}}]}
function page(){render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><ScienceTasksPage/></MemoryRouter></QueryClientProvider>)}
it('opens saved question, previews locally and restores focus on Escape',async()=>{
 const fetcher=vi.fn(async(url:string,_init?:RequestInit)=>({ok:true,status:200,json:async()=>url.endsWith('/1')?task:{items:[task]}} as Response));vi.stubGlobal('fetch',fetcher);page();const trigger=await screen.findByRole('button',{name:/科普测试/});trigger.focus();fireEvent.click(trigger);await screen.findByRole('button',{name:'选择：鸭子'});fireEvent.click(screen.getByRole('button',{name:'选择：鸭子'}));expect(screen.getByText('答对了')).toBeInTheDocument();expect(fetcher.mock.calls.every(c=>c.length===1||!(c[1] as RequestInit)?.method||(c[1] as RequestInit)?.method==='GET')).toBe(true);fireEvent(await screen.findByRole('dialog'),new Event('cancel',{bubbles:true,cancelable:true}));await waitFor(()=>expect(screen.queryByRole('dialog')).not.toBeInTheDocument());expect(trigger).toHaveFocus();expect(screen.queryByText('发布')).not.toBeInTheDocument()
})
it('generates selected types then loads persisted task detail',async()=>{
 const calls:{url:string;init?:RequestInit}[]=[];vi.stubGlobal('fetch',vi.fn(async(url:string,init?:RequestInit)=>{calls.push({url,init});return {ok:true,status:200,json:async()=>init?.method==='POST'||url.endsWith('/1')?task:{items:[]}} as Response}));page();await screen.findByText('暂无科普出题任务。');fireEvent.click(screen.getByRole('button',{name:'生成科普题目'}));fireEvent.click(screen.getByRole('button',{name:'生成并保存'}));await screen.findByRole('button',{name:'选择：鸭子'});const post=calls.find(c=>c.init?.method==='POST');expect(JSON.parse(post!.init!.body as string)).toMatchObject({count:12,types:['choice','match','sequence','label']});expect(calls.some(c=>c.url.endsWith('/1')&&c.init?.method==='GET')).toBe(true)
})
it('shows load failure without claiming an empty list',async()=>{vi.stubGlobal('fetch',vi.fn(async()=>({ok:false,status:503,json:async()=>({error:'科普任务服务未配置'})})));page();expect(await screen.findByRole('alert')).toHaveTextContent('科普任务服务未配置');expect(screen.queryByText('暂无科普出题任务。')).not.toBeInTheDocument()}
)
