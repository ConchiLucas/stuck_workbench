import { fireEvent, render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { MathPlayer } from '../../../packages/math-player/src/index'

it('fills the missing number and locks a correct answer', () => {
  render(<MathPlayer mode="kid" example={{kind:'missing',prompt:'3 + □ = 8',options:['4','5'],answer:'5'}} />)
  fireEvent.click(screen.getByRole('button',{name:'5'}))
  expect(screen.getByLabelText('3 + 5 = 8')).toBeInTheDocument()
  expect(screen.getByRole('button',{name:'4'})).toBeDisabled()
})
it('does not reveal the listening answer when audio is missing', () => {
  render(<MathPlayer mode="kid" example={{kind:'audio-shape',prompt:'听到：圆形',options:['○'],answer:'○'}} />)
  expect(screen.queryByText('听到：圆形')).not.toBeInTheDocument()
  expect(screen.getByRole('alert')).toHaveTextContent('读音')
})
it('uses supplied counting images and marks removed objects', () => {
  render(<MathPlayer mode="kid" example={{kind:'objects',prompt:'还剩几个',counts:[3,1],operation:'sub',object:'苹果',objectImageUrl:'/apple.png',options:['2'],answer:'2'}} />)
  expect(screen.getAllByRole('img')).toHaveLength(3)
  expect(screen.getByLabelText('拿走的苹果')).toBeInTheDocument()
})
it('classifies by selecting a card and a destination', () => {
  render(<MathPlayer mode="kid" example={{kind:'shape-sort',prompt:'分一分',shapeImageUrls:{circle:'/circle.png',triangle:'/triangle.png'},buckets:[{label:'没有角',items:['circle']},{label:'有角',items:['triangle']}]}} />)
  fireEvent.click(screen.getByRole('button',{name:'选择圆形'}))
  fireEvent.click(screen.getByRole('button',{name:'放入没有角'}))
  fireEvent.click(screen.getByRole('button',{name:'选择三角形'}))
  fireEvent.click(screen.getByRole('button',{name:'放入有角'}))
  fireEvent.click(screen.getByRole('button',{name:'检查分类'}))
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
})
it('shows only the arithmetic expression without optional narration', () => {
  render(<MathPlayer mode="kid" example={{kind:'choice',prompt:'3 + 5 = ?',audioUrl:'/question.mp3',options:['8'],answer:'8'}} />)
  expect(screen.getByLabelText('3 + 5')).toBeInTheDocument()
  expect(screen.queryByRole('button',{name:'播放题目读音'})).not.toBeInTheDocument()
})
it('preserves the equality needed for judgement questions', () => {
  render(<MathPlayer mode="kid" example={{kind:'judgement',prompt:'判断下面的算式',statements:['7 + 6 = 12'],options:['对','错'],answer:'错'}} />)
  expect(screen.getByText('7 + 6 = 12')).toBeInTheDocument()
})
it('falls back to countable glyphs when object images are not published', () => {
  render(<MathPlayer mode="kid" example={{kind:'objects',prompt:'一共几个',counts:[2,1],operation:'add',object:'star',options:['3'],answer:'3'}} />)
  expect(screen.getAllByLabelText('剩下的星星')).toHaveLength(3)
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
})
it('draws shape glyphs when bitmap urls are missing', () => {
  render(<MathPlayer mode="kid" example={{kind:'shape-name',prompt:'这是什么图形？',shapeKeys:['triangle'],options:['三角形','圆形'],answer:'三角形'}} />)
  expect(screen.getByRole('img',{name:'三角形'})).toBeInTheDocument()
  expect(screen.queryByText('图片待准备')).not.toBeInTheDocument()
})
it('keeps history read-only and still allows listening', () => {
  const onAnswer = () => { throw new Error('history must not write an answer') }
  render(<MathPlayer mode="kid" readOnly example={{kind:'choice',prompt:'3 + 5',options:['7','8'],answer:'8',audioUrl:'/no-choice-audio.mp3'}} initialAnswer={{selected:'7',placements:{},correct:false}} onAnswer={onAnswer} />)
  expect(screen.getByRole('status')).toHaveTextContent('当时答错')
  expect(screen.getByRole('button',{name:'8'})).toBeDisabled()
  expect(screen.queryByRole('button',{name:'播放题目读音'})).not.toBeInTheDocument()
})
it('plays listening audio without choosing an option', () => {
  render(<MathPlayer mode="kid" example={{kind:'audio-shape',prompt:'听到：圆形',options:['○','△'],answer:'○',audioUrl:'/shape.mp3',shapeKeys:['circle','triangle']}} />)
  fireEvent.click(screen.getByRole('button',{name:'播放题目读音'}))
  expect(screen.getByRole('button',{name:'圆形'})).toHaveAttribute('aria-pressed','false')
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
})
