import { readFileSync } from 'node:fs'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MathPlayer, type MathDetail } from '../../../packages/math-player/src/index'

describe('shared math player', () => {
  const defaults=JSON.parse(readFileSync('../shared-go/mathcontent/defaults.json','utf8')).items as MathDetail[]
  it.each(defaults.filter(detail=>detail.example.kind!=='shape-sort'))('checks canonical published $id',detail=>{
    render(<MathPlayer detail={detail}/>)
    const index=detail.example.options!.indexOf(detail.example.answer!)
    const options=document.querySelectorAll<HTMLButtonElement>('.math-player-options button')
    fireEvent.click(options[index])
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
  })
  it('explains missing prepared audio and uses supplied asset URL mapping', () => {
    const {rerender}=render(<MathPlayer example={{kind:'audio-shape',prompt:'听圆形',options:['circle'],answer:'circle'}} />)
    expect(screen.getByText('题干音频尚未准备好，可以先看文字试做。')).toBeInTheDocument()
    rerender(<MathPlayer example={{kind:'audio-shape',prompt:'听圆形',options:['circle'],answer:'circle',audioUrl:'/api/audio.mp3'}} resolveAssetUrl={url=>'/apps/math'+url}/>)
    expect(screen.getByLabelText('播放题目读音')).toHaveAttribute('src','/apps/math/api/audio.mp3')
  })
  it('draws the material object rather than generic dots',()=>{
    render(<MathPlayer example={{kind:'objects',prompt:'数星星',operation:'add',counts:[3,2],object:'star',options:['5'],answer:'5'}}/>)
    expect(screen.getAllByText('★')).toHaveLength(5)
  })
  it('checks an answer locally and resets feedback', () => {
    render(<MathPlayer example={{kind:'choice',prompt:'3 + 5 = ?',options:['7','8'],answer:'8'}} />)
    fireEvent.click(screen.getByRole('button',{name:'7'}))
    expect(screen.getByRole('status')).toHaveTextContent('再想一想')
    fireEvent.click(screen.getByRole('button',{name:'8'}))
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
    fireEvent.click(screen.getByRole('button',{name:'重新试做'}))
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
  it('marks the removed quantity rather than drawing two addition groups', () => {
    render(<MathPlayer example={{kind:'objects',prompt:'还剩几个？',operation:'sub',counts:[8,3],object:'苹果',options:['5','6'],answer:'5'}} />)
    expect(screen.getByLabelText('原来 8 个，拿走 3 个')).toBeInTheDocument()
    expect(screen.getAllByLabelText('拿走的苹果')).toHaveLength(3)
    expect(screen.getAllByLabelText('剩下的苹果')).toHaveLength(5)
  })
  it('requires classification and checks each selected bucket', () => {
    render(<MathPlayer example={{kind:'shape-sort',prompt:'按角分类',buckets:[{label:'没有角',items:['circle']},{label:'有角',items:['triangle']}]}} />)
    const circle = screen.getByRole('combobox',{name:'圆形放在哪一组'})
    fireEvent.change(circle,{target:{value:'1'}})
    fireEvent.change(screen.getByRole('combobox',{name:'三角形放在哪一组'}),{target:{value:'1'}})
    fireEvent.click(screen.getByRole('button',{name:'检查分类'}))
    expect(screen.getByRole('status')).toHaveTextContent('再想一想')
    fireEvent.change(circle,{target:{value:'0'}})
    fireEvent.click(screen.getByRole('button',{name:'检查分类'}))
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
  })
  it('judges the selected statement and resets when a detail changes', () => {
    const {rerender}=render(<MathPlayer example={{kind:'judgement',prompt:'判断',statements:['7 + 6 = 12'],options:['正确','错误'],answer:'错误'}} />)
    fireEvent.click(screen.getByRole('button',{name:'错误'}))
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
    rerender(<MathPlayer example={{kind:'judgement',prompt:'找错',statements:['11 − 3 = 8','14 − 6 = 9'],answer:'14 − 6 = 9'}} />)
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button',{name:'14 − 6 = 9'}))
    expect(screen.getByRole('status')).toHaveTextContent('答对了')
  })
})
