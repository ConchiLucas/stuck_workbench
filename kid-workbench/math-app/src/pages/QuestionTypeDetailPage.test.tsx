import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { QuestionTypeDetailPage } from './QuestionTypeDetailPage'

const detail={id:'addition-equation',groupId:'addition',title:'发布的算式题',moduleTitle:'加法',learningGoal:'合起来',rules:['先计算'],revision:3,example:{kind:'choice',prompt:'2 + 6 = ?',options:['7','8'],answer:'8'}}
function renderDetail(path='/types/addition-equation') {
  const client=new QueryClient({defaultOptions:{queries:{retry:false}}})
  render(<QueryClientProvider client={client}><MemoryRouter initialEntries={[path]}><Routes><Route path="/types/:typeId" element={<QuestionTypeDetailPage/>}/></Routes></MemoryRouter></QueryClientProvider>)
}
afterEach(()=>vi.restoreAllMocks())
describe('published math detail',()=>{
  it('loads published revision and checks options without navigation or writes',async()=>{
    const fetcher=vi.spyOn(globalThis,'fetch').mockResolvedValue(new Response(JSON.stringify({data:{schemaVersion:1,items:[detail]}})))
    renderDetail()
    expect(screen.getByText('正在加载题目…')).toBeInTheDocument()
    expect(await screen.findByRole('heading',{name:'发布的算式题'})).toBeInTheDocument()
    expect(screen.queryByText(/素材版本/)).not.toBeInTheDocument()
    expect(screen.getByLabelText('2 + 6')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button',{name:'7'}))
    expect(screen.getByRole('status')).toHaveTextContent('再想一想')
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow','1')
    expect(fetcher).toHaveBeenCalledTimes(1)
    expect(String(fetcher.mock.calls[0][0])).toBe('/api/v1/math/details')
  })
  it('keeps the answer when returning to the previous question',async()=>{
    const second={...detail,id:'subtraction-equation',example:{kind:'choice',prompt:'9 − 4 = ?',options:['5','6'],answer:'5'}}
    vi.spyOn(globalThis,'fetch').mockImplementation(async()=>new Response(JSON.stringify({data:{schemaVersion:1,items:[detail,second]}})))
    renderDetail()
    fireEvent.click(await screen.findByRole('button',{name:'8'}))
    fireEvent.click(screen.getByRole('link',{name:'下一题'}))
    expect(await screen.findByLabelText('9 − 4')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('link',{name:'上一题'}))
    expect(await screen.findByRole('button',{name:'8'})).toHaveAttribute('aria-pressed','true')
    expect(screen.getByRole('button',{name:'7'})).toBeDisabled()
  })
  it('shows failures without a hardcoded fallback',async()=>{
    vi.spyOn(globalThis,'fetch').mockResolvedValue(new Response(JSON.stringify({error:{message:'素材服务不可用'}}),{status:503}))
    renderDetail()
    expect(await screen.findByRole('alert')).toHaveTextContent('素材服务不可用')
    expect(screen.queryByText('3 + 5 = ?')).not.toBeInTheDocument()
    expect(screen.getByRole('button',{name:'重新加载'})).toBeInTheDocument()
  })
  it('rejects unknown catalog versions',async()=>{
    vi.spyOn(globalThis,'fetch').mockResolvedValue(new Response(JSON.stringify({data:{schemaVersion:2,items:[detail]}})))
    renderDetail()
    expect(await screen.findByRole('alert')).toHaveTextContent('素材目录格式或版本不受支持')
  })
  it('explains missing published content',async()=>{
    vi.spyOn(globalThis,'fetch').mockResolvedValue(new Response(JSON.stringify({data:{schemaVersion:1,items:[]}})))
    renderDetail()
    expect(await screen.findByRole('heading',{name:'这个题型还没有发布素材'})).toBeInTheDocument()
  })
})
