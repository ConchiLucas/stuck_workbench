import { readFileSync } from 'node:fs'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AppRoutes } from './App'
const catalog=JSON.parse(readFileSync('../shared-go/mathcontent/defaults.json','utf8'))
beforeEach(()=>vi.spyOn(globalThis,'fetch').mockImplementation(async()=>new Response(JSON.stringify({data:catalog}))))
afterEach(()=>vi.restoreAllMocks())
describe('math published sample routes',()=>{
 it('keeps five entrances and opens published material samples',async()=>{
  render(<MemoryRouter><AppRoutes/></MemoryRouter>)
  expect(screen.getAllByTestId('question-type-card')).toHaveLength(5)
  expect(screen.getByRole('link',{name:'看算式选答案'})).toHaveAttribute('href','/types/addition-equation')
  fireEvent.click(screen.getByRole('link',{name:'看算式选答案'}))
  expect(await screen.findByLabelText('3 + 5')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button',{name:'8'}))
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
  expect(screen.getByRole('progressbar')).toBeInTheDocument()
  expect(vi.mocked(fetch).mock.calls.every(([url,init])=>String(url)==='/api/v1/math/details'&&!init?.method)).toBe(true)
 })
 it('resolves old demo links to published samples without local scoring',async()=>{
  render(<MemoryRouter initialEntries={['/practice/type/story']}><AppRoutes/></MemoryRouter>)
  expect(await screen.findByText('一共有几颗星星？')).toBeInTheDocument()
  expect(screen.getAllByLabelText('剩下的星星')).toHaveLength(5)
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  expect(screen.queryByText('试做示例，不计入学习进度')).not.toBeInTheDocument()
  expect(screen.getByRole('progressbar')).toBeInTheDocument()
 })
 it('draws shape glyphs when published bitmaps are missing',async()=>{
  render(<MemoryRouter initialEntries={['/types/shape-feature']}><AppRoutes/></MemoryRouter>)
  expect(await screen.findByRole('heading',{name:'按特征找图形'})).toBeInTheDocument()
  expect(document.querySelector('.math-kid-player svg')).not.toBeNull()
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
 })
})
