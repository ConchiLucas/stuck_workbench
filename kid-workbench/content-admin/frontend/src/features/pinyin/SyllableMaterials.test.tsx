import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {render,screen,cleanup} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {afterEach,it,expect,vi} from 'vitest'
import {SyllableMaterials} from './SyllableMaterials'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('shows missing recordings and saves material eligibility without generating questions',async()=>{
 const item={id:1,initialText:'b',finalText:'ā',tone:1,syllableText:'bā',speechText:'八',speechUrl:'',enabled:true}
 const fetcher=vi.spyOn(globalThis,'fetch').mockImplementation(async(_url,init)=>new Response(JSON.stringify(init?.method==='PATCH'?{...item,...JSON.parse(String(init.body))}:{items:[item],total:1}),{status:200}))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><SyllableMaterials/></QueryClientProvider>)
 expect(await screen.findByRole('button',{name:'试听音节 bā'})).toBeDisabled()
 expect(screen.getByText(/缺少真人录音/)).toBeInTheDocument()
 await userEvent.click(screen.getByRole('button',{name:'停用'}))
 expect(fetcher.mock.calls.some(([url,init])=>String(url).endsWith('/syllables/1')&&init?.method==='PATCH'&&String(init.body).includes('"enabled":false'))).toBe(true)
 expect(fetcher.mock.calls.some(([url])=>String(url).includes('/quiz/'))).toBe(false)
})
it('discards cancelled draft when changing material eligibility',async()=>{
 const item={id:1,initialText:'b',finalText:'ā',tone:1,syllableText:'bā',speechText:'八',speechUrl:'',enabled:true}
 const fetcher=vi.spyOn(globalThis,'fetch').mockImplementation(async(_url,init)=>new Response(JSON.stringify(init?.method==='PATCH'?{...item,...JSON.parse(String(init.body))}:{items:[item],total:1}),{status:200}))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><SyllableMaterials/></QueryClientProvider>)
 await userEvent.click(await screen.findByRole('button',{name:'编辑例字'}))
 await userEvent.clear(screen.getByRole('textbox',{name:'bā例字'}))
 await userEvent.type(screen.getByRole('textbox',{name:'bā例字'}),'取消的草稿')
 await userEvent.click(screen.getByRole('button',{name:'取消'}))
 await userEvent.click(screen.getByRole('button',{name:'停用'}))
 const patch=fetcher.mock.calls.find(([,init])=>init?.method==='PATCH')
 expect(JSON.parse(String(patch?.[1]?.body))).toEqual({speechText:'八',enabled:false})
})
