import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { HomePage } from './HomePage'

describe('math home', () => {
  it('shows five mixed type picture cards without splitting addition and subtraction', () => {
    render(<MemoryRouter><HomePage /></MemoryRouter>)
    expect(screen.getAllByTestId('question-type-card')).toHaveLength(5)
    expect(screen.getAllByRole('link')).toHaveLength(5)
    expect(screen.getByRole('link', { name: '看算式选答案' })).toHaveAttribute('href', '/types/addition-equation')
    expect(screen.getByRole('link', { name: '看数量图选答案' })).toHaveAttribute('href', '/types/addition-story')
    expect(screen.getByRole('link', { name: '补全算式' })).toHaveAttribute('href', '/types/addition-missing')
    expect(screen.getByRole('link', { name: '判断算式对错' })).toHaveAttribute('href', '/types/addition-judge')
    expect(screen.getByRole('link', { name: '认识图形' })).toHaveAttribute('href', '/types/shape-find')
    expect(screen.getByRole('link', { name: '看算式选答案' }).querySelector('img')).toHaveAttribute('src', '/cards/equation.svg')
    expect(screen.getByRole('link', { name: '认识图形' }).querySelector('img')).toHaveAttribute('src', '/cards/shape.svg')
    expect(screen.queryByRole('heading', { name: '算数可以怎么练？' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: '20 以内加法' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: '20 以内减法' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /图形分类/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /听名称找图形/ })).not.toBeInTheDocument()
    expect(screen.queryByText('当前支持')).not.toBeInTheDocument()
    expect(screen.queryByText('可以扩展')).not.toBeInTheDocument()
    expect(screen.queryByText('406')).not.toBeInTheDocument()
    expect(screen.queryByText('我的算数地图')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /开始今日练习/ })).not.toBeInTheDocument()
  })
})
