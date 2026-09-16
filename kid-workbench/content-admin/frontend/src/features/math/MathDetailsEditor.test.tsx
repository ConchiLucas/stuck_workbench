import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MathDetailsEditor } from './MathDetailsEditor'
const detail={id:'addition-equation',groupId:'addition',title:'算式题',moduleTitle:'加法',learningGoal:'合起来',rules:['先计算'],revision:1,publishedRevision:1,example:{kind:'choice',prompt:'2 + 3 = ?',options:['4','5'],answer:'5',operation:'add',counts:[2,3]}}
function mount(){render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}})}><MathDetailsEditor/></QueryClientProvider>)}
afterEach(()=>{cleanup();vi.restoreAllMocks()})
describe('material details editor',()=>{
 it('keeps shape option keys, visible options and the selected answer together',async()=>{
  const shape={...detail,id:'shape-find',groupId:'shape',title:'听图形',example:{kind:'audio-shape',prompt:'听到：圆形',options:['○','△'],shapeKeys:['circle','triangle'],answer:'○'}}
  const fetcher=vi.spyOn(globalThis,'fetch').mockImplementation(async(_input,init)=>init?.method==='PUT'?new Response(String(init.body)):new Response(JSON.stringify({schemaVersion:1,items:[shape]})))
  mount()
  fireEvent.click(await screen.findByRole('button',{name:/图形 · 听图形/}))
  fireEvent.change(screen.getByLabelText('选项图形 1'),{target:{value:'square'}})
  fireEvent.click(screen.getByRole('button',{name:'保存草稿'}))
  await screen.findByText('草稿已保存')
  const body=JSON.parse(String(fetcher.mock.calls.find(([,init])=>init?.method==='PUT')![1]?.body))
  expect(body.example.shapeKeys[0]).toBe('square')
  expect(body.example.options[0]).toBe('square')
  expect(body.example.answer).toBe('square')
  expect(body.example.prompt).toBe('听到：正方形')
 })
 it('generates audio on a saved revision before publication',async()=>{
  const fetcher=vi.spyOn(globalThis,'fetch').mockImplementation(async(_input,init)=>init?.method==='POST'?new Response(JSON.stringify({...detail,revision:2,example:{...detail.example,audioUrl:'/api/audio.mp3'}})):new Response(JSON.stringify({schemaVersion:1,items:[detail]})))
  mount()
  fireEvent.click(await screen.findByRole('button',{name:/加法 · 算式题/}))
  fireEvent.click(screen.getByRole('button',{name:'生成题干音频'}))
  expect(await screen.findByText('题干音频已生成，请预览后发布')).toBeInTheDocument()
  expect(screen.getByLabelText('播放题目读音')).toHaveAttribute('src','/api/audio.mp3')
  expect(fetcher.mock.calls.some(([input,init])=>String(input).endsWith('/addition-equation/audio')&&init?.body==='{"revision":1}')).toBe(true)
 })
 it('edits ordinary fields, saves a new revision and explicitly publishes it',async()=>{
  let saved=detail
  const fetcher=vi.spyOn(globalThis,'fetch').mockImplementation(async(_input,init)=>{
   if(init?.method==='PUT'){saved={...JSON.parse(String(init.body)),revision:2,publishedRevision:1};return new Response(JSON.stringify(saved))}
   if(init?.method==='POST')return new Response(JSON.stringify({...saved,publishedRevision:2}))
   return new Response(JSON.stringify({schemaVersion:1,items:[saved]}))
  })
  mount()
  fireEvent.click(await screen.findByRole('button',{name:/加法 · 算式题/}))
  fireEvent.change(screen.getByLabelText('题干'),{target:{value:'改后的题干'}})
  expect(screen.getByRole('button',{name:'发布此版本'})).toBeDisabled()
  fireEvent.click(screen.getByRole('button',{name:'保存草稿'}))
  expect(await screen.findByText('草稿已保存')).toBeInTheDocument()
  expect(screen.getByLabelText('题干')).toHaveValue('改后的题干')
  fireEvent.click(screen.getByRole('button',{name:'发布此版本'}))
  expect(await screen.findByText('版本 2 已发布')).toBeInTheDocument()
  const publish=fetcher.mock.calls.find(([,init])=>init?.method==='POST')!
  expect(String(publish[0])).toBe('/api/v1/math/details/addition-equation/publish')
  expect(JSON.parse(String(publish[1]?.body))).toEqual({revision:2})
  fireEvent.click(screen.getByRole('button',{name:'5'}))
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
  expect(fetcher.mock.calls.every(([input])=>String(input).startsWith('/api/v1/math/details'))).toBe(true)
 })
 it('keeps unsaved edits on version conflict and offers reload',async()=>{
  vi.spyOn(globalThis,'fetch').mockImplementation(async(_input,init)=>init?.method==='PUT'?new Response(JSON.stringify({error:'版本冲突，请刷新后重试'}),{status:409}):new Response(JSON.stringify({schemaVersion:1,items:[detail]})))
  mount()
  fireEvent.click(await screen.findByRole('button',{name:/加法 · 算式题/}))
  fireEvent.change(screen.getByLabelText('题干'),{target:{value:'保留编辑'}})
  fireEvent.click(screen.getByRole('button',{name:'保存草稿'}))
  expect(await screen.findByRole('alert')).toHaveTextContent('版本冲突')
  expect(screen.getByLabelText('题干')).toHaveValue('保留编辑')
  expect(screen.getByRole('button',{name:'重新加载已保存版本'})).toBeInTheDocument()
 })
 it('reports catalog errors',async()=>{
  vi.spyOn(globalThis,'fetch').mockResolvedValue(new Response(JSON.stringify({schemaVersion:2,items:[]})))
  mount()
  await waitFor(()=>expect(screen.getByRole('alert')).toHaveTextContent('素材目录格式或版本不受支持'))
 })
})
