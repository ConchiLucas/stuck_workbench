import {cleanup,fireEvent,render,screen} from '@testing-library/react'
import {MemoryRouter} from 'react-router-dom'
import {afterEach,expect,test} from 'vitest'
import App from './App'
import {usePoemStore} from './store'

afterEach(()=>{cleanup();usePoemStore.setState({picks:{},sequence:{},wrongs:{}})})

test('home is a four-type poetry gallery',()=>{
  render(<MemoryRouter initialEntries={['/']}><App/></MemoryRouter>)
  expect(screen.getByRole('link',{name:'选诗名'})).toHaveAttribute('href','/practice/type/title')
  expect(screen.getByRole('link',{name:'补字'})).toHaveAttribute('href','/practice/type/fill')
  expect(screen.getByRole('link',{name:'选下一句'})).toHaveAttribute('href','/practice/type/couplet')
  expect(screen.getByRole('link',{name:'排顺序'})).toHaveAttribute('href','/practice/type/recite')
  expect(document.querySelectorAll('.type-card img')).toHaveLength(4)
  expect(screen.getByRole('link',{name:'选诗名'}).querySelector('img')).toHaveAttribute('src','/cards/title.png')
  expect(screen.queryByRole('navigation',{name:'诗园'})).not.toBeInTheDocument()
  expect(screen.queryByText('七亭读诗')).not.toBeInTheDocument()
})

test('title practice picks a poem name',()=>{
  const {container}=render(<MemoryRouter initialEntries={['/practice/type/title']}><App/></MemoryRouter>)
  expect(screen.getByText('选诗名')).toBeInTheDocument()
  expect(screen.getByRole('heading',{name:'这首诗叫什么？'})).toBeInTheDocument()
  expect(screen.getByText('床前明月光')).toBeInTheDocument()
  expect(container.querySelector('.ink-glyph')).toBeNull()
  fireEvent.click(screen.getByRole('button',{name:'春晓'}))
  expect(screen.getByRole('button',{name:'春晓'})).toHaveClass('is-wrong')
  expect(screen.getByRole('status')).toHaveTextContent('再试一次')
  fireEvent.click(screen.getByRole('button',{name:'静夜思'}))
  expect(screen.getByRole('button',{name:'静夜思'})).toHaveClass('is-right')
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
})

test('fill practice completes a missing character',()=>{
  const {container}=render(<MemoryRouter initialEntries={['/practice/type/fill']}><App/></MemoryRouter>)
  expect(screen.getByText('补字')).toBeInTheDocument()
  expect(screen.getByText('锄禾日当□')).toBeInTheDocument()
  expect(container.querySelector('.ink-glyph')).toBeNull()
  fireEvent.click(screen.getByRole('button',{name:'午'}))
  expect(screen.getByRole('button',{name:'午'})).toHaveClass('is-right')
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
})

test('couplet practice picks the next line',()=>{
  render(<MemoryRouter initialEntries={['/practice/type/couplet']}><App/></MemoryRouter>)
  expect(screen.getByText('选下一句')).toBeInTheDocument()
  expect(screen.getByRole('heading',{name:'下一句是哪一句？'})).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button',{name:'处处闻啼鸟'}))
  expect(screen.getByRole('button',{name:'处处闻啼鸟'})).toHaveClass('is-wrong')
  fireEvent.click(screen.getByRole('button',{name:'黄河入海流'}))
  expect(screen.getByRole('button',{name:'黄河入海流'})).toHaveClass('is-right')
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
})

test('recite practice orders Jing Ye Si',()=>{
  render(<MemoryRouter initialEntries={['/practice/type/recite']}><App/></MemoryRouter>)
  expect(screen.getByText('排顺序')).toBeInTheDocument()
  expect(screen.getByRole('heading',{name:'按顺序点出这4句'})).toBeInTheDocument()
  expect(screen.getByRole('listitem',{name:'第 1 句'})).toHaveTextContent('？')
  for (const line of ['床前明月光','疑是地上霜','举头望明月','低头思故乡']) {
    fireEvent.click(screen.getByRole('button',{name:line}))
  }
  expect(screen.getByRole('listitem',{name:'第 1 句'})).toHaveTextContent('床前明月光')
  expect(screen.getByRole('button',{name:'床前明月光'})).toHaveClass('is-right')
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
})

test('last next control is 查看结果',()=>{
  render(<MemoryRouter initialEntries={['/practice/type/title/4']}><App/></MemoryRouter>)
  expect(screen.getByRole('link',{name:'查看结果'})).toHaveAttribute('href','/practice/type/title/result')
  expect(screen.getByRole('link',{name:'上一题'})).toHaveAttribute('href','/practice/type/title/3')
})

test('result lists picks and retries the same type',()=>{
  usePoemStore.setState({picks:{'title:1':'pm001','title:2':'pm002'}})
  render(<MemoryRouter initialEntries={['/practice/type/title/result']}><App/></MemoryRouter>)
  expect(screen.getByRole('heading',{name:'答题结果'})).toBeInTheDocument()
  expect(screen.getByText('答对 2 / 4 题')).toBeInTheDocument()
  expect(screen.getByText('你选了 静夜思')).toBeInTheDocument()
  expect(screen.getByRole('link',{name:'再做一次'})).toHaveAttribute('href','/practice/type/title')
  expect(screen.getByRole('link',{name:'回到首页'})).toHaveAttribute('href','/')
})
