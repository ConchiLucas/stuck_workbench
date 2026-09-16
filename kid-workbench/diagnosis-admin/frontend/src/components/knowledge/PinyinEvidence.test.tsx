import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AnswerEvidence } from './AnswerEvidence'
import type { Evidence } from '../../api/knowledgeTypes'

afterEach(cleanup)

function fixture(type: string): Evidence {
 return {attemptId:42, subjectCode:'pinyin', questionType:type, questionFidelity:'instance_snapshot', selectionFidelity:'stable_option', isCorrect:false, mediaFidelity:'mutable_reference', response:{kind:'choice',selectedOptionId:'b'}, question:{interaction:'choice',targetText:'ā',stem:{text:'',audioMediaId:'stem-audio'},visual:{kind:type,text:type==='inword'?'阿':'ā',initial:'b',final:'ā'},options:[{id:'b',label:'ō'},{id:'a',label:'ā'},{id:'c',label:'ē'},{id:'d',label:'ī'}],answerOptionId:'a'}} as Evidence
}
describe('pinyin frozen evidence',()=>{
 for(const type of ['listen','inword','shape','blend']) it(`${type} uses a read-only shared question with frozen selection`,()=>{
  const {container}=render(<AnswerEvidence evidence={fixture(type)}/>);
  expect(container.querySelector('[data-readonly="true"]')).toBeInTheDocument();
  expect(container.querySelector('.is-picked')).toBeInTheDocument();
  expect(screen.getByText('当时答错')).toBeInTheDocument();
  const picks=type==='shape'||type==='blend'?screen.getAllByRole('button',{name:/选择读音/}):container.querySelectorAll('.option-button');
  Array.from(picks).forEach(button=>expect(button).toBeDisabled());
  expect(screen.getByText(/孩子选了：ō/)).toBeInTheDocument();
 });
 it('never synthesizes missing audio',()=>{
  const speak=vi.fn();vi.stubGlobal('speechSynthesis',{speak,cancel:vi.fn()});
  render(<AnswerEvidence evidence={fixture('shape')}/>);fireEvent.click(screen.getByRole('button',{name:'播放读音 1'}));
  expect(screen.getByRole('alert')).toHaveTextContent('读音素材暂不可用');expect(speak).not.toHaveBeenCalled();vi.unstubAllGlobals();
 });
 it('does not upgrade current material or incomplete visuals to historical scenes',()=>{
  const e=fixture('blend');e.questionFidelity='current_reference';
  const {container,rerender}=render(<AnswerEvidence evidence={e}/>);expect(container.querySelector('[data-readonly]')).toBeNull();
  e.questionFidelity='instance_snapshot';e.question!.visual={kind:'blend'};rerender(<AnswerEvidence evidence={e}/>);
  expect(container.querySelector('[data-readonly]')).toBeNull();expect(screen.getByText(/未保存完整拼音题面/)).toBeInTheDocument();
 });
});

it('rejects incomplete or mismatched frozen choice evidence',()=>{
 for(const patch of [ {options:[{id:'a',label:'ā'}]}, {answerOptionId:'absent'}, {options:[{id:'b',label:''},{id:'a',label:'ā'},{id:'c',label:'ē'},{id:'d',label:'ī'}]} ]) {
  const e=fixture('listen');Object.assign(e.question!,patch);const {container,unmount}=render(<AnswerEvidence evidence={e}/>);expect(container.querySelector('[data-readonly]')).toBeNull();unmount();
 }
 const e=fixture('listen');e.response.selectedOptionId='absent';const {container}=render(<AnswerEvidence evidence={e}/>);expect(container.querySelector('[data-readonly]')).toBeNull();
});
