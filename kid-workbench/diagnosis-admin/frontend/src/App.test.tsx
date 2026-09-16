import { render,screen } from '@testing-library/react'
import { QueryClient,QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { afterEach,expect,it,vi } from 'vitest'
import { App } from './App'
afterEach(()=>vi.restoreAllMocks())
it('opens an actual attempt id from the application route',async()=>{
 const calls:string[]=[];vi.spyOn(globalThis,'fetch').mockImplementation(async input=>{calls.push(String(input));return new Response(JSON.stringify(String(input).includes('/summary')?{child:{name:'小朋友'}}:{attemptId:42,kpId:10,title:'山的现场',isCorrect:false,question:null,questionFidelity:'unavailable',response:{kind:'unknown'},occurredAt:'2026-09-12T00:00:00Z',costMs:1000}),{status:200})})
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={['/attempts/42']}><App/></MemoryRouter></QueryClientProvider>);expect(await screen.findByText('山的现场')).toBeInTheDocument();expect(calls.some(c=>c.endsWith('/attempts/42'))).toBe(true);expect(calls.some(c=>c.includes('undefined'))).toBe(false)
})
