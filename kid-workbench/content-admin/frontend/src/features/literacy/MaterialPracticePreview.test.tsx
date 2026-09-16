import {render,screen,waitFor,cleanup} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {it,expect,vi,afterEach} from 'vitest'
import {MaterialPracticePreview} from './MaterialPracticePreview'
afterEach(()=>{cleanup();vi.restoreAllMocks()})
it('freezes real materials and sends preview answers without writing attempts',async()=>{
 const snapshot={schemaVersion:2,questionType:'glyph_sense',stem:{image:{revisionId:'r1',kind:'glyph'}},options:[{id:'a',text:'山',image:{revisionId:'r1',kind:'sense'},audio:{revisionId:'r1',kind:'speech'}}],materialRevisionIds:['r1']}
 const request=vi.spyOn(globalThis,'fetch').mockImplementation(async(url)=>String(url).endsWith('/writing-template')?new Response('{}',{status:404}):new Response(JSON.stringify(String(url).endsWith('/answer')?{correct:true}:snapshot),{status:200}))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MaterialPracticePreview kpId={4}/></QueryClientProvider>)
 await screen.findByLabelText('看字选义题面');expect(document.querySelector('img[src="/api/v1/material-revisions/r1/media/sense"]')).toBeInTheDocument();await userEvent.click(await screen.findByRole('button',{name:'山'}));await screen.findByText('答对啦')
 const calls=request.mock.calls.filter(c=>String(c[0]).includes('generation-preview'));expect(JSON.parse(String(calls[0][1]?.body))).toEqual({kpId:4,questionType:'glyph_sense'})
 await waitFor(()=>expect(request).toHaveBeenCalledTimes(3));expect(JSON.parse(String(calls[1][1]?.body))).toEqual({snapshot,response:{kind:'choice',selectedOptionId:'a'}})
 request.mockRestore()
})
it('matches App audio placement for meaning-to-character questions',async()=>{
 const snapshot={schemaVersion:2,questionType:'sense_char',stem:{image:{revisionId:'r1',kind:'sense'},audio:{revisionId:'r1',kind:'speech'}},options:[{id:'a',text:'山',image:{revisionId:'r1',kind:'glyph'},audio:{revisionId:'r1',kind:'speech'}}]}
 const request=vi.spyOn(globalThis,'fetch').mockImplementation(async(url)=>new Response(JSON.stringify(String(url).endsWith('/writing-template')?{}:snapshot),{status:String(url).endsWith('/writing-template')?404:200}))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MaterialPracticePreview kpId={4}/></QueryClientProvider>)
 await screen.findByRole('button',{name:'选项 1'})
 expect(screen.getByRole('button',{name:'播放读音'})).toBeInTheDocument()
 expect(screen.queryByRole('button',{name:'播放选项 1 读音'})).not.toBeInTheDocument()
 request.mockRestore()
})
