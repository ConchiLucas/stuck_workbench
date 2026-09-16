import {cleanup,fireEvent,render,screen} from '@testing-library/react'
import {MemoryRouter} from 'react-router-dom'
import '@testing-library/jest-dom/vitest'
import {afterEach,expect,test,vi} from 'vitest'
import App from './App'
import {useDemoAnswerStore} from './store/demoAnswerStore'
import {useLiveQuizStore} from './store/liveQuizStore'

function stubQuizFail(){
  vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response('<html>502</html>',{status:502})))
}

async function renderLocal(path:string){
  stubQuizFail()
  render(<MemoryRouter initialEntries={[path]}><App/></MemoryRouter>)
  fireEvent.click(await screen.findByRole('button',{name:'用示例题'}))
}

afterEach(()=>{
  cleanup()
  useDemoAnswerStore.setState({picks:{}})
  useLiveQuizStore.getState().invalidate('audio-choice')
  useLiveQuizStore.getState().invalidate('image-text')
  vi.unstubAllGlobals()
})
test('home is a full-screen gallery of five practice types',()=>{
  render(<MemoryRouter initialEntries={['/']}><App/></MemoryRouter>)
  expect(screen.queryByRole('heading',{name:'7套脚本找到孩子该怎么学'})).not.toBeInTheDocument()
  expect(screen.queryByText('WORD')).not.toBeInTheDocument()
  expect(screen.queryByRole('link',{name:/查看通用脚本/})).not.toBeInTheDocument()
  const titles=['听音选词','看图选词','组句子','写单词','读一读']
  expect(screen.getAllByRole('link',{name:/查看题型：/})).toHaveLength(5)
  expect(screen.getByRole('link',{name:'查看题型：听音选词'})).toHaveAttribute('href','/types/audio-choice')
  expect(screen.getByRole('link',{name:'查看题型：读一读'})).toHaveAttribute('href','/types/reading-qa')
  expect(screen.queryByRole('link',{name:'查看题型：听音辨词'})).not.toBeInTheDocument()
  expect(screen.queryByRole('link',{name:'查看题型：换一种练'})).not.toBeInTheDocument()
  const gallery=document.querySelector('.type-gallery')
  expect(gallery).toBeTruthy()
  expect(gallery?.children).toHaveLength(5)
  for(const title of titles){
    const link=screen.getByRole('link',{name:`查看题型：${title}`})
    expect(link.querySelector('img')).toBeTruthy()
    expect(link.querySelector('strong')).toHaveTextContent(title)
  }
})

test('catalogue uses the same five-type gallery',()=>{
  render(<MemoryRouter initialEntries={['/types']}><App/></MemoryRouter>)
  expect(screen.queryByRole('heading',{name:'最值得开发的通用脚本'})).not.toBeInTheDocument()
  expect(screen.getAllByRole('link',{name:/查看题型：/})).toHaveLength(5)
  expect(screen.queryByText('认大写字母')).not.toBeInTheDocument()
  expect(screen.queryByText('词汇宾果')).not.toBeInTheDocument()
  expect(screen.queryByText('听音辨词')).not.toBeInTheDocument()
  expect(screen.queryByText('换一种练')).not.toBeInTheDocument()
})

test('script detail is a kid practice screen, not a design brief',async ()=>{
  const scrollTo=vi.fn()
  Object.defineProperty(window,'scrollTo',{value:scrollTo,writable:true})
  stubQuizFail()
  render(<MemoryRouter initialEntries={['/']}><App/></MemoryRouter>)
  fireEvent.click(screen.getByRole('link',{name:'查看题型：听音选词'}))
  fireEvent.click(await screen.findByRole('button',{name:'用示例题'}))
  expect(scrollTo).toHaveBeenCalledWith({top:0,behavior:'auto'})
  expect(screen.queryByRole('heading',{name:'听一听，哪个是 apple？'})).not.toBeInTheDocument()
  expect(screen.queryByText('听一听，哪个是 apple？')).not.toBeInTheDocument()
  expect(screen.getByRole('button',{name:'播放'})).toBeInTheDocument()
  expect(screen.queryByText('效果预览')).not.toBeInTheDocument()
  expect(screen.queryByText('覆盖题型')).not.toBeInTheDocument()
  expect(screen.queryByText('孩子答错说明什么')).not.toBeInTheDocument()
  expect(screen.queryByText('下一步怎么学')).not.toBeInTheDocument()
  expect(screen.queryByText('HOW IT FEELS')).not.toBeInTheDocument()
  expect(screen.queryByRole('button',{name:'提交答案'})).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button',{name:'苹果'}))
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
  expect(screen.getByRole('button',{name:'苹果'})).toHaveClass('is-right')
})

test('a wrong first tap can be retried once',async ()=>{
  await renderLocal('/types/audio-choice')
  fireEvent.click(screen.getByRole('button',{name:'香蕉'}))
  expect(screen.getByRole('status')).toHaveTextContent('再试一次')
  fireEvent.click(screen.getByRole('button',{name:'苹果'}))
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
})

test('practice top bar uses close, a progress track, and a primary next control',async ()=>{
  await renderLocal('/types/audio-choice')
  const back=screen.getByRole('link',{name:'返回'})
  expect(back).toHaveClass('kid-circle')
  expect(back.querySelector('svg')).toBeTruthy()
  expect(screen.queryByText('听音选词')).not.toBeInTheDocument()
  const bar=screen.getByRole('progressbar',{name:'听音选词，第 1 / 4 题'})
  expect(bar).toHaveTextContent('1 / 4')
  expect(bar.querySelector('.kid-progress-label')).toHaveAttribute('aria-hidden','true')
  expect(bar).toHaveAttribute('aria-valuenow','1')
  expect(bar).toHaveAttribute('aria-valuemax','4')
  expect(screen.queryByRole('link',{name:'上一题'})).not.toBeInTheDocument()
  const top=screen.getByRole('navigation',{name:'练习导航'})
  expect(top.firstElementChild).toHaveAttribute('aria-label','返回')
  expect(top.children[1]).toHaveAttribute('role','progressbar')
  expect(top.lastElementChild).toHaveClass('kid-nav')
  expect(top.lastElementChild).toContainElement(screen.getByRole('link',{name:'下一题'}))
  expect(screen.getByRole('link',{name:'下一题'})).toHaveClass('is-next')
  expect(screen.getByRole('link',{name:'下一题'})).toHaveAttribute('href','/types/audio-choice/2')
  const playBtn=screen.getByRole('button',{name:'播放'})
  expect(playBtn.querySelector('[data-icon="play"] svg')).toHaveAttribute('viewBox','0 0 24 24')
  expect(playBtn.querySelector('[data-icon="play"] svg')).toHaveAttribute('fill','currentColor')
  expect(playBtn.querySelector('.kid-play-rings')).toBeTruthy()
  fireEvent.click(playBtn)
  expect(playBtn).toHaveClass('is-playing')
  expect(screen.queryByText('播放')).not.toBeInTheDocument()
  expect(screen.queryByText('返回')).not.toBeInTheDocument()
  expect(screen.queryByText('下一题')).not.toBeInTheDocument()
})

test('next and previous stay inside the same question type',async ()=>{
  await renderLocal('/types/audio-choice')
  expect(screen.getByRole('button',{name:'苹果'})).toBeInTheDocument()
  fireEvent.click(screen.getByRole('link',{name:'下一题'}))
  expect(screen.getByRole('progressbar',{name:'听音选词，第 2 / 4 题'})).toHaveTextContent('2 / 4')
  expect(screen.getByRole('button',{name:'小猫'})).toBeInTheDocument()
  expect(screen.queryByRole('button',{name:'苹果'})).not.toBeInTheDocument()
  expect(screen.queryByRole('heading',{name:'哪一张图是 apple？'})).not.toBeInTheDocument()
  expect(screen.getByRole('link',{name:'上一题'})).toHaveAttribute('href','/types/audio-choice')
  expect(screen.getByRole('link',{name:'下一题'})).toHaveAttribute('href','/types/audio-choice/3')
})

test('the last item opens a result list instead of leaving the type',async ()=>{
  await renderLocal('/types/audio-choice/4')
  expect(screen.getByRole('progressbar',{name:'听音选词，第 4 / 4 题'})).toHaveTextContent('4 / 4')
  expect(screen.getByRole('link',{name:'下一题'})).toHaveAttribute('href','/types/audio-choice/result')
  expect(screen.getByRole('link',{name:'上一题'})).toHaveAttribute('href','/types/audio-choice/3')
})

test('does not lock after a wrong first tap',async ()=>{
  await renderLocal('/types/audio-choice')
  fireEvent.click(screen.getByRole('button',{name:'香蕉'}))
  expect(screen.getByRole('status')).toHaveTextContent('再试一次')
  expect(screen.getByRole('button',{name:'香蕉'})).not.toHaveClass('is-wrong')
  expect(screen.getByRole('button',{name:'苹果'})).not.toBeDisabled()
})

test('shows a result list after the last question',()=>{
  useDemoAnswerStore.setState({picks:{'audio-choice:1':'apple','audio-choice:2':'fish'}})
  render(<MemoryRouter initialEntries={['/types/audio-choice/result']}><App/></MemoryRouter>)
  expect(screen.getByRole('heading',{name:'答题结果'})).toBeInTheDocument()
  expect(screen.getByText('答对 1 / 4 题')).toBeInTheDocument()
  expect(screen.getAllByRole('listitem')).toHaveLength(4)
  expect(screen.getByText('答对')).toBeInTheDocument()
  expect(screen.getByText('答错')).toBeInTheDocument()
  expect(screen.getAllByText('未作答')).toHaveLength(2)
  expect(screen.getByRole('link',{name:'再练一次'})).toHaveAttribute('href','/types/audio-choice')
  expect(screen.getByRole('link',{name:'回到首页'})).toHaveAttribute('href','/')
})

test('look-and-choose cards hide the English word already named in the prompt',async ()=>{
  await renderLocal('/types/image-text')
  expect(screen.getByRole('heading',{name:'哪一张图是 apple？'})).toBeInTheDocument()
  const apple=screen.getByRole('button',{name:'apple'})
  expect(apple.querySelector('.kid-pic')).toBeTruthy()
  expect(apple.querySelector('strong')).toBeNull()
  expect(apple).not.toHaveTextContent('apple')
  expect(screen.getByRole('button',{name:'banana'})).not.toHaveTextContent('banana')
  expect(screen.getByRole('button',{name:'dog'})).not.toHaveTextContent('dog')
  expect(screen.getByRole('button',{name:'bird'})).not.toHaveTextContent('bird')
})

test('sound discrimination is removed from the kid app',()=>{
  render(<MemoryRouter initialEntries={['/types/sound-discrimination']}><App/></MemoryRouter>)
  expect(screen.queryByRole('button',{name:'播放'})).not.toBeInTheDocument()
  expect(screen.getAllByRole('link',{name:/查看题型：/})).toHaveLength(5)
  expect(screen.queryByRole('link',{name:'查看题型：听音辨词'})).not.toBeInTheDocument()
})

test('sentence builder uses large word cards, not a text hint',()=>{
  render(<MemoryRouter initialEntries={['/types/card-builder']}><App/></MemoryRouter>)
  expect(screen.getByRole('heading',{name:'把单词排成一句话'})).toBeInTheDocument()
  expect(screen.getByRole('button',{name:'播放'})).toBeInTheDocument()
  expect(screen.queryByText('点下面的单词')).not.toBeInTheDocument()
  for(const word of ['This','is','an','apple']){
    fireEvent.click(screen.getByRole('button',{name:word}))
  }
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
  expect(useDemoAnswerStore.getState().pick('card-builder',1)).toBe('This is an apple')
})

test('spelling uses a text field to type the word',()=>{
  render(<MemoryRouter initialEntries={['/types/input-gap']}><App/></MemoryRouter>)
  expect(screen.getByRole('heading',{name:'写出这个单词'})).toBeInTheDocument()
  expect(screen.getByRole('button',{name:'播放'})).toBeInTheDocument()
  expect(screen.queryByText('你的答案')).not.toBeInTheDocument()
  expect(screen.queryByRole('button',{name:'a'})).not.toBeInTheDocument()
  expect(screen.queryByRole('button',{name:'重来'})).not.toBeInTheDocument()
  expect(screen.queryByRole('link',{name:'写好了'})).not.toBeInTheDocument()
  const field=screen.getByRole('textbox',{name:'单词'})
  fireEvent.change(field,{target:{value:'apple'}})
  expect(document.querySelector('.kid-cue img')).toBeTruthy()
  expect(screen.queryByText('🍎')).not.toBeInTheDocument()
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
  expect(field).not.toBeDisabled()
  const done=screen.getByRole('link',{name:'写好了'})
  expect(done).toHaveAttribute('href','/types/input-gap/2')
  fireEvent.click(done)
  expect(screen.getByRole('progressbar',{name:'写单词，第 2 / 4 题'})).toHaveTextContent('2 / 4')
  expect(screen.getByRole('textbox',{name:'单词'})).toHaveValue('')
})

test('spelling enter submits the word and opens the next item',()=>{
  render(<MemoryRouter initialEntries={['/types/input-gap']}><App/></MemoryRouter>)
  const field=screen.getByRole('textbox',{name:'单词'})
  fireEvent.change(field,{target:{value:'app'}})
  fireEvent.keyDown(field,{key:'Enter'})
  expect(screen.getByRole('progressbar',{name:'写单词，第 2 / 4 题'})).toHaveTextContent('2 / 4')
})

test('the last spelling item continues to the result list',()=>{
  render(<MemoryRouter initialEntries={['/types/input-gap/4']}><App/></MemoryRouter>)
  expect(screen.queryByRole('link',{name:'写好了'})).not.toBeInTheDocument()
  fireEvent.change(screen.getByRole('textbox',{name:'单词'}),{target:{value:'sun'}})
  expect(screen.getByRole('link',{name:'写好了'})).toHaveAttribute('href','/types/input-gap/result')
})

test('audio-choice loads four server questions',async ()=>{
  vi.stubGlobal('fetch',vi.fn().mockImplementation(async ()=>new Response(JSON.stringify({
    data:{
      instanceId:'q1',type:'listen',stem:'听一听，选出你听到的单词',targetId:101,
      speechText:'cat',visual:{kind:'sound'},
      options:[
        {id:101,label:'小猫'},{id:102,label:'小鱼'},
        {id:103,label:'太阳'},{id:104,label:'书本'},
      ],
      answerIndex:0,
    },
    error:null,
  }))))
  render(<MemoryRouter initialEntries={['/types/audio-choice']}><App/></MemoryRouter>)
  expect(await screen.findByRole('button',{name:'小猫'})).toBeInTheDocument()
  expect(screen.queryByRole('button',{name:'苹果'})).not.toBeInTheDocument()
})

test('audio-choice can fall back to the local demo bank',async ()=>{
  stubQuizFail()
  render(<MemoryRouter initialEntries={['/types/audio-choice']}><App/></MemoryRouter>)
  expect(await screen.findByText('服务返回了无法识别的内容')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button',{name:'用示例题'}))
  expect(await screen.findByRole('button',{name:'苹果'})).toBeInTheDocument()
})
