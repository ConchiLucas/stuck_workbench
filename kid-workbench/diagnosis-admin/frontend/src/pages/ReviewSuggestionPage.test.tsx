import { cleanup,render,screen,fireEvent,waitFor } from '@testing-library/react'
import { QueryClient,QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter,Routes,Route } from 'react-router-dom'
import { afterEach,expect,it,vi } from 'vitest'
import { ReviewSuggestionPage } from './ReviewSuggestionPage'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('retries failed partitions and refreshes a rejected row version',async()=>{
 const posts:{body:string;headers:unknown}[]=[];let version=2
 vi.spyOn(globalThis,'fetch').mockImplementation(async(input,init)=>{if(init?.method==='POST'){posts.push({body:String(init.body),headers:init.headers});if(posts.length===1){version=3;return new Response(JSON.stringify({error:{code:'version_conflict',message:'内容已变化'}}),{status:409})}return new Response(JSON.stringify({id:7}),{status:202})};return new Response(JSON.stringify(String(input).endsWith('/review-suggestions/7')?{id:7,title:'复习山',rowVersion:version,lifecycle:'open',generationStatus:'partial',targets:[],partitions:[{id:1,key:'g1',status:'succeeded',generatedCount:3},{id:2,key:'g2',status:'failed',generatedCount:0}],requestedCount:6,generatedCount:3}:{items:[],hasMore:false}),{status:200})})
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={['/reviews/7']}><Routes><Route path="/reviews/:id" element={<ReviewSuggestionPage/>}/></Routes></MemoryRouter></QueryClientProvider>)
 fireEvent.click(await screen.findByRole('button',{name:'重试未完成部分'}));await screen.findByText('内容已变化');await waitFor(()=>expect(screen.getByRole('button',{name:'重试未完成部分'})).not.toBeDisabled());fireEvent.click(screen.getByRole('button',{name:'重试未完成部分'}));await waitFor(()=>expect(posts).toHaveLength(2));expect(JSON.parse(posts[0].body)).toEqual({expectedRowVersion:2,partitionKeys:['g2']});expect(JSON.parse(posts[1].body).expectedRowVersion).toBe(3);expect(posts[0].headers).not.toEqual(posts[1].headers)
})

it('keeps saved pinyin analysis visible without promising unsupported generation',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).endsWith('/review-suggestions/8')?{id:8,title:'拼音复习',subjectCode:'pinyin',lifecycle:'open',generationStatus:null,targets:[],partitions:[],requestedCount:3,generatedCount:0}:{items:[],hasMore:false})))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={['/reviews/8']}><Routes><Route path="/reviews/:id" element={<ReviewSuggestionPage/>}/></Routes></MemoryRouter></QueryClientProvider>)
 expect(await screen.findByText(/拼音分析已保存/)).toBeInTheDocument();expect(screen.queryByRole('button',{name:'生成复习草稿'})).not.toBeInTheDocument();
})
