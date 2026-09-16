import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { AnswerEvidence } from './AnswerEvidence'
import type { Evidence } from '../../api/knowledgeTypes'

afterEach(cleanup)

function fixture(): Evidence {
 return {attemptId:42, subjectCode:'math', questionType:'calc', skillCode:'calc', questionFidelity:'instance_snapshot', selectionFidelity:'stable_option', isCorrect:false, mediaFidelity:'immutable', response:{kind:'choice',selectedOptionId:'o1'}, mathExample:{kind:'choice',prompt:'2 + 5',options:['6','7','8','9'],answer:'7',optionIds:['o1','o2','o3','o4'],answerOptionId:'o2'}, question:{interaction:'choice',targetText:'2+5',stem:{text:'2 + 5'},options:[{id:'o1',label:'6'},{id:'o2',label:'7'},{id:'o3',label:'8'},{id:'o4',label:'9'}],answerOptionId:'o2'}} as Evidence
}

describe('math frozen evidence',()=>{
 it('uses a read-only shared question with frozen selection',()=>{
  const {container}=render(<AnswerEvidence evidence={fixture()}/>)
  expect(container.querySelector('[data-readonly="true"]')).toBeInTheDocument()
  expect(screen.getByText('当时答错')).toBeInTheDocument()
  expect(screen.getByRole('button',{name:'6'})).toBeDisabled()
  expect(screen.getByText(/孩子选了：6/)).toBeInTheDocument()
  expect(screen.getByText(/正确答案：7/)).toBeInTheDocument()
 })
})
