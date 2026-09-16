import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { AnswerEvidence } from './AnswerEvidence'
import type { Evidence } from '../../api/knowledgeTypes'

describe('actual answer evidence',()=>{
 it('shows actual selected and correct content without exposing raw indices',()=>{
 const evidence={attemptId:1,isCorrect:false,questionFidelity:'frozen_version',selectionFidelity:'stable_option',evidenceReasonCodes:[],question:{interaction:'choice',stem:{text:'看字选义'},targetText:'山',answerOptionId:'mountain',options:[{id:'water',label:'水'},{id:'mountain',label:'山'}]},response:{kind:'choice',selectedOptionId:'water'}} as unknown as Evidence
 render(<AnswerEvidence evidence={evidence}/>);expect(screen.getByText('孩子选了')).toBeInTheDocument();expect(screen.getByText('正确答案')).toBeInTheDocument();expect(screen.getByText('水')).toBeInTheDocument();expect(screen.queryByText(/picks/)).not.toBeInTheDocument()
 })
 it('does not claim a current reference is the historical question',()=>{
 render(<AnswerEvidence evidence={{attemptId:2,questionFidelity:'current_reference',selectionFidelity:'missing',evidenceReasonCodes:[],question:{interaction:'reference',stem:{text:'当前题干'},options:[]},response:{kind:'unknown'}} as unknown as unknown as Evidence}/>);expect(screen.getByText(/未保存当时的选项/)).toBeInTheDocument()
 })
})
it('preserves the pinyin blending prompt without adding the syllable answer to the stem',()=>{
 const e={attemptId:4,questionFidelity:'instance_snapshot',response:{kind:'choice'},question:{interaction:'choice',stem:{text:'把声母和韵母拼在一起'},visual:{kind:'blend',initial:'b',final:'a',syllable:'ba'},options:[]}} as unknown as Evidence
 render(<AnswerEvidence evidence={e}/>);expect(screen.getByLabelText('拼读现场')).toHaveTextContent('b + a = ?');expect(screen.queryByText('把声母和韵母拼在一起 · ba')).not.toBeInTheDocument()
})
