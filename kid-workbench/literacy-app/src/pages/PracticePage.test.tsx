import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {render,screen} from '@testing-library/react'
import {MemoryRouter,Route,Routes} from 'react-router-dom'
import {it,expect,vi,afterEach} from 'vitest'
import {PracticePage} from './PracticePage'
import {literacyApi} from '../api/literacy'
import type {PlanDetail} from '../api/types'
afterEach(()=>vi.restoreAllMocks())
it('directs pending legacy handwriting to published tasks without accepting or finishing it',async()=>{
 vi.spyOn(literacyApi,'plan').mockResolvedValue({plan:{id:4,status:'doing',planDate:'2026-09-12',seqNo:1,subjectCode:'literacy',targetCount:1,doneCount:0,correctCount:0,stars:0,durationSec:0},items:[{id:12,kpId:1,status:'pending',tries:0,seq:1,character:'一',bucket:'new',picks:'',optionOrder:'',question:{id:1,code:'write_char',type:'write',options:[],stem:'',visual:{},speech:{}}}]} as PlanDetail)
 const answer=vi.spyOn(literacyApi,'answer'),finish=vi.spyOn(literacyApi,'finish')
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={['/practice/4']}><Routes><Route path="/practice/:planId" element={<PracticePage/>}/></Routes></MemoryRouter></QueryClientProvider>)
 expect(await screen.findByText('这道旧听写题缺少冻结的书写模板，无法使用新版判题。')).toBeInTheDocument()
 expect(screen.getByRole('link',{name:'前往新版听写任务'})).toHaveAttribute('href','/tasks?questionType=write_char')
 expect(screen.queryByLabelText('写字板')).not.toBeInTheDocument();expect(screen.queryByRole('button',{name:'写完了'})).not.toBeInTheDocument()
 expect(answer).not.toHaveBeenCalled();expect(finish).not.toHaveBeenCalled()
})

it('uses the frozen prompt for both versioned interactions', async()=>{
 const {toPlayerQuestion}=await import('./PracticePage')
 for(const code of ['write_char','sense_char']){
  const item={id:1,kpId:1,question:{versionId:1,code,type:code==='write_char'?'handwriting':'choice',stem:'冻结题目中的完整提示语',snapshot:{stem:{}},options:[]}} as unknown as import('../api/types').PlanItem
  expect(toPlayerQuestion(item).prompt).toBe('冻结题目中的完整提示语')
 }
})
