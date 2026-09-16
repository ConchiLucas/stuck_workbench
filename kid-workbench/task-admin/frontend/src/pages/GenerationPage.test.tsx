import {render,screen,fireEvent,waitFor,cleanup} from '@testing-library/react'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {MemoryRouter,Routes,Route,Link} from 'react-router-dom'
import {afterEach,expect,it,vi} from 'vitest'
import {GenerationPage} from './GenerationPage'
afterEach(()=>{cleanup();vi.unstubAllGlobals()})
it('shows task cards without the removed heading, creation controls, filters or status',async()=>{
 vi.stubGlobal('fetch',vi.fn(async()=>({ok:true,json:async()=>[taskFixture()]} as Response)))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><GenerationPage/></MemoryRouter></QueryClientProvider>)
 expect(await screen.findByRole('link',{name:/任务1/})).toHaveAttribute('href','/tasks/1')
 expect(screen.queryByRole('button',{name:'新建出题任务'})).not.toBeInTheDocument()
 expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
 expect(screen.queryByText('草稿')).not.toBeInTheDocument()
 expect(screen.queryByRole('heading',{name:'出题任务'})).not.toBeInTheDocument()
})

const taskFixture=(id=1,status='draft')=>({id,title:'任务'+id,subjectCode:'literacy',moduleName:'第一组',targetCount:1,status,kind:'practice',sourceMode:'material_template',rowVersion:2,activeRevisionId:1,publishedRevisionId:status==='published'?1:undefined,updatedAt:'2026-09-12',spec:{subjectCode:'literacy',kind:'practice',scope:{moduleCodes:['g1'],kpIds:[1]},targetCount:1,typeCounts:{glyph_sense:1},distractorScope:'module'},items:[{id:10,seq:1,fingerprint:'fp',snapshot:{schemaVersion:1,kpId:1,targetText:'春',questionType:'glyph_sense',prompt:'选图片',stem:{text:'春'},options:[{id:'kp:1',kpId:1,text:'春'}],answerOptionId:'kp:1',explanation:'春',materialRevisionIds:[]}}],revisions:[{id:1,revisionNo:1,createdAt:'2026-09-12'}]})
function detailPage(){render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={['/tasks/1']}><Link to="/tasks/2">切换任务</Link><Routes><Route path="/tasks/:id" element={<GenerationPage/>}/></Routes></MemoryRouter></QueryClientProvider>)}
it('opens task questions in a dialog and closes back to the list',async()=>{
 vi.stubGlobal('fetch',vi.fn(async(url:string)=>({ok:true,status:200,json:async()=>url.includes('?subject=')?[taskFixture()]:taskFixture()} as Response)))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><Routes><Route path="/" element={<GenerationPage/>}/><Route path="/tasks/:id" element={<GenerationPage/>}/></Routes></MemoryRouter></QueryClientProvider>)
 fireEvent.click(await screen.findByRole('link',{name:/任务1/}))
 expect(await screen.findByRole('dialog',{name:'题目查看'})).toBeInTheDocument()
 await screen.findByText('第 1 题 · 春')
 for(const name of ['刷新','修改要求','整包重新生成','删除任务'])expect(screen.queryByRole('button',{name})).not.toBeInTheDocument()
 expect(screen.queryByText('普通出题')).not.toBeInTheDocument()
 expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
 fireEvent.click(screen.getByRole('button',{name:'关闭弹窗'}))
 await waitFor(()=>expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
 expect(screen.getByRole('link',{name:/任务1/})).toBeInTheDocument()
 fireEvent.click(screen.getByRole('link',{name:/任务1/}))
 fireEvent(await screen.findByRole('dialog'),new Event('cancel',{bubbles:true,cancelable:true}))
 await waitFor(()=>expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
})
it.each(['draft','published'])('shows %s task questions without publication controls',async(status)=>{
 vi.stubGlobal('fetch',vi.fn(async(url:string)=>({ok:true,status:200,json:async()=>url.includes('?subject=')?[]:url.endsWith('/review-jobs')?[]:taskFixture(1,status)} as Response)))
 detailPage();await screen.findByText('第 1 题 · 春')
 expect(screen.queryByRole('button',{name:'发布题包'})).not.toBeInTheDocument()
 expect(screen.queryByRole('button',{name:'撤回发布'})).not.toBeInTheDocument()
 expect(screen.queryByText(/普通出题 ·/)).not.toBeInTheDocument()
 expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
})
it('shows the historical selected option as review evidence',async()=>{
 vi.stubGlobal('fetch',vi.fn(async(url:string)=>({ok:true,status:200,json:async()=>url.includes('?subject=')?[]:url.endsWith('/review-evidence')?[{receiptId:17,planId:9,sourceQuestionVersionId:10,createdAt:'2026-09-12',targetText:'春',questionType:'glyph_sense',selectedOptionId:'kp:2',selectedOption:{id:'kp:2',kpId:2,text:'雨'},correctOption:{id:'kp:1',kpId:1,text:'春'}}]:{...taskFixture(),kind:'review',targetChildId:1}} as Response)))
 detailPage();await screen.findByText('当时选了：雨');expect(screen.getByText('正确选项：春')).toBeInTheDocument();expect(screen.getByText('作答回执 #17 · 原题版本 #10')).toBeInTheDocument()
})
it('shows original handwriting and hinted evaluation without a missing choice message',async()=>{
 vi.stubGlobal('fetch',vi.fn(async(url:string)=>({ok:true,status:200,json:async()=>url.includes('?subject=')?[]:url.endsWith('/review-evidence')?[{receiptId:18,planId:9,sourceQuestionVersionId:10,createdAt:'2026-09-12',targetText:'山',questionType:'write_char',responseKind:'handwriting',answerPayload:{kind:'handwriting',strokes:[[{x:.1,y:.2,t:0},{x:.8,y:.9,t:10}]],hintsUsed:1},evaluation:{outcome:'passed',assistance:'hinted'}}]:{...taskFixture(),kind:'review',targetChildId:1}} as Response)))
 detailPage();expect(await screen.findByLabelText('当时的书写笔迹')).toBeInTheDocument();expect(screen.getByText('通过 · 使用过提示')).toBeInTheDocument();expect(screen.getByText('标准字：山')).toBeInTheDocument();expect(screen.queryByText('当时选了：未保存具体选项')).not.toBeInTheDocument()
})
it('matches App meaning-question audio without changing the frozen snapshot',async()=>{
 const task=taskFixture();task.items[0].snapshot={...task.items[0].snapshot,questionType:'sense_char',stem:{image:{revisionId:'r',kind:'sense'},audio:{revisionId:'r',kind:'speech'}},options:[{id:'kp:1',kpId:1,text:'春',audio:{revisionId:'r',kind:'speech'}}]} as any
 vi.stubGlobal('fetch',vi.fn(async(url:string)=>({ok:true,status:200,json:async()=>url.includes('?subject=')?[]:task} as Response)))
 detailPage();await screen.findByRole('button',{name:'选项 1'})
 expect(screen.getByRole('button',{name:'播放读音'})).toBeInTheDocument()
 expect(screen.queryByRole('button',{name:'播放选项 1 读音'})).not.toBeInTheDocument()
 expect((task.items[0].snapshot.options[0] as any).audio).toBeDefined()
})
