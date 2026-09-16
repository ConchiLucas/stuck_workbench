import { render,screen,fireEvent,waitFor,cleanup } from '@testing-library/react'
import { QueryClient,QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { afterEach,expect,it,vi } from 'vitest'
import { ReviewSuggestionsPage } from './ReviewSuggestionsPage'
afterEach(()=>{cleanup();vi.restoreAllMocks();localStorage.clear();sessionStorage.clear()})
it('saves explicit requirements without generating and reuses the request key after a lost response',async()=>{
 const posts:RequestInit[]=[]
 vi.spyOn(globalThis,'fetch').mockImplementation(async (input,init)=>{if(init?.method==='POST'){posts.push(init);throw new Error('网络中断')};return new Response(JSON.stringify(String(input).includes('review-candidates')?{items:[{key:'10:glyph_sense',kpId:10,title:'山',subjectCode:'literacy',questionType:'glyph_sense',reasonCode:'observed_wrong',reasonText:'记录过一次错误',evidence:[{attemptId:4,role:'target_error',source:{kind:'version_receipt'}}],reviewEligible:true}],evidenceAsOf:'2026-09-12T00:00:00Z',hasMore:false}:{items:[],hasMore:false}),{status:200})})
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><ReviewSuggestionsPage/></MemoryRouter></QueryClientProvider>)
 fireEvent.click(await screen.findByRole('checkbox'));fireEvent.click(screen.getByRole('button',{name:'保存复习建议'}));await screen.findByText('网络中断');fireEvent.click(screen.getByRole('button',{name:'重试保存建议'}));await waitFor(()=>expect(posts).toHaveLength(2));expect(posts[0].headers).toEqual(posts[1].headers);expect(posts[0].body).toBe(posts[1].body);expect(JSON.parse(String(posts[0].body)).targets[0].requestedCount).toBe(3)
})

it('restores a draft after leaving and prevents mixing subjects',async()=>{
 const items=[{key:'10:glyph_sense',kpId:10,title:'山',subjectCode:'literacy',questionType:'glyph_sense',reasonText:'需要再练',evidence:[],reviewEligible:true},{key:'20:count',kpId:20,title:'数数',subjectCode:'math',questionType:'count',reasonText:'需要再练',evidence:[],reviewEligible:true}]
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).includes('review-candidates')?{items,evidenceAsOf:'2026-09-13T00:00:00Z',hasMore:false}:{items:[],hasMore:false}),{status:200}))
 const mount=()=>render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><ReviewSuggestionsPage/></MemoryRouter></QueryClientProvider>)
 const first=mount();const checks=await screen.findAllByRole('checkbox');fireEvent.click(checks[0]);expect(checks[1]).toBeDisabled()
 fireEvent.change(screen.getByLabelText('山题数'),{target:{value:'7'}})
 await waitFor(()=>expect(localStorage.getItem('knowledge-review-draft:1')).toContain('"count":7'))
 first.unmount();mount();await screen.findByText('已恢复上次未保存的复习选择。');expect(screen.getByLabelText('山题数')).toHaveValue(7)
 fireEvent.click(screen.getByRole('button',{name:'清空本次选择'}));await waitFor(()=>expect(localStorage.getItem('knowledge-review-draft:1')).toBeNull());expect(screen.getAllByRole('checkbox')[1]).toBeEnabled()
})
