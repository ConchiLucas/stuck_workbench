import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, expect, it, vi } from 'vitest'
import { literacyApi } from '../api/literacy'
import { TasksPage } from './TasksPage'

function page() {return render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><MemoryRouter initialEntries={['/tasks']}><Routes><Route path="/tasks" element={<TasksPage />} /><Route path="/practice/:id" element={<p>已经进入练习</p>} /></Routes></MemoryRouter></QueryClientProvider>)}
afterEach(() => vi.restoreAllMocks())
it('continues the existing plan without claiming another', async () => {
  vi.spyOn(literacyApi,'tasks').mockResolvedValue({items:[{id:1,title:'第一组',kind:'practice',revisionId:7,targetCount:8,planId:40,planStatus:'doing'}]})
  const claim = vi.spyOn(literacyApi,'claimTask')
  page();await userEvent.click(await screen.findByRole('button',{name:'继续练习'}))
  expect(await screen.findByText('已经进入练习')).toBeInTheDocument();expect(claim).not.toHaveBeenCalled()
})
it('retries a failed claim with its original key', async () => {
  vi.spyOn(literacyApi,'tasks').mockResolvedValue({items:[{id:2,title:'第二组',kind:'review',revisionId:8,targetCount:4}]})
  const claim=vi.spyOn(literacyApi,'claimTask').mockRejectedValue(new Error('连接暂时中断'))
  page();await userEvent.click(await screen.findByRole('button',{name:'开始练习'}));await screen.findByRole('alert')
  await userEvent.click(screen.getByRole('button',{name:'开始练习'}));expect(claim).toHaveBeenCalledTimes(2)
  expect(claim.mock.calls[0]).toEqual(claim.mock.calls[1])
})
it('filters by published question types while retaining a complete mixed task',async()=>{
 vi.spyOn(literacyApi,'tasks').mockResolvedValue({items:[{id:1,title:'混合练习',kind:'practice',revisionId:1,targetCount:12,questionTypes:['glyph_sense','write_char']},{id:2,title:'只有选择',kind:'practice',revisionId:2,targetCount:4,questionTypes:['sense_char']}]})
 render(<QueryClientProvider client={new QueryClient()}><MemoryRouter initialEntries={['/tasks?questionType=write_char']}><TasksPage/></MemoryRouter></QueryClientProvider>)
 expect(await screen.findByText('混合练习')).toBeInTheDocument();expect(screen.queryByText('只有选择')).not.toBeInTheDocument();expect(screen.getByText(/12 题.*混合题包/)).toBeInTheDocument()
})
