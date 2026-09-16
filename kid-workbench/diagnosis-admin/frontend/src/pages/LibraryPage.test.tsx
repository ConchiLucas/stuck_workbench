import { render,screen } from '@testing-library/react'
import { QueryClient,QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { afterEach,expect,it,vi } from 'vitest'
import { LibraryPage } from './LibraryPage'
afterEach(()=>vi.restoreAllMocks())
it('shows specific mastered abilities and does not show a medical dashboard',async()=>{
 vi.spyOn(globalThis,'fetch').mockImplementation(async input=>new Response(JSON.stringify(String(input).includes('/summary')?{child:{id:1,name:'小朋友'},subjects:[],practicedCount:1,masteredAbilityCount:1,wrongPointCount:1,wrongCount:2}:{items:[{kpId:10,title:'山',subjectName:'识字',moduleName:'第一组',stats:{observedAttempts:3},wrongCount:2,skills:[{skillCode:'glyph_sense',label:'看字选义',masteryStatus:'mastered',practiced:true,stats:{observedAttempts:3}}]}],hasMore:false}),{status:200}))
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter><LibraryPage/></MemoryRouter></QueryClientProvider>);expect(await screen.findByText('山')).toBeInTheDocument();expect(screen.getByText(/看字选义/)).toBeInTheDocument();expect(screen.queryByText('学科健康度')).not.toBeInTheDocument()
})
