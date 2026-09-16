import {fireEvent,render,screen,waitFor} from '@testing-library/react'
import {beforeEach,describe,it,expect,vi} from 'vitest'
beforeEach(()=>{vi.spyOn(HTMLMediaElement.prototype,'play').mockResolvedValue();vi.spyOn(HTMLMediaElement.prototype,'pause').mockImplementation(()=>{})})
import {LiteracyPlayer} from '@kid-workbench/literacy-player'
const question={id:'q',interaction:'choice' as const,stem:{image:'glyph'},options:[{id:'b',text:'乙',image:'sense-b',audio:'audio-b'},{id:'a',text:'甲',image:'sense-a'}]}
describe('shared literacy player',()=>{
 it('keeps frozen option order and submits stable ids',async()=>{const submit=vi.fn().mockResolvedValue({correct:true});render(<LiteracyPlayer question={question} mediaResolver={x=>String(x)} onSubmit={submit}/>);expect(screen.getAllByRole('button',{name:/^选项/}).map(x=>x.getAttribute('aria-label'))).toEqual(['选项 1','选项 2']);fireEvent.keyDown(screen.getByRole('button',{name:'选项 2'}),{key:'Enter'});await waitFor(()=>expect(submit).toHaveBeenCalledWith({kind:'choice',selectedOptionId:'a'}))})
 it('does not select an option when its audio is played with keyboard',()=>{const submit=vi.fn();vi.spyOn(HTMLMediaElement.prototype,'play').mockResolvedValue();render(<LiteracyPlayer question={question} mediaResolver={x=>String(x)} onSubmit={submit}/>);const audio=screen.getByRole('button',{name:'播放选项 1 读音'});fireEvent.keyDown(audio,{key:'Enter'});fireEvent.click(audio);expect(submit).not.toHaveBeenCalled()})
 it('locks pending submit and allows retry after network failure',async()=>{let reject!:(e:Error)=>void;const submit=vi.fn().mockImplementationOnce(()=>new Promise((_,r)=>{reject=r})).mockResolvedValue({correct:true});render(<LiteracyPlayer question={question} mediaResolver={x=>String(x)} onSubmit={submit}/>);fireEvent.click(screen.getByRole('button',{name:'选项 1'}));fireEvent.click(screen.getByRole('button',{name:'选项 2'}));expect(submit).toHaveBeenCalledTimes(1);reject(new Error('离线'));await screen.findByRole('alert');fireEvent.click(screen.getByRole('button',{name:'重试提交'}));await waitFor(()=>expect(submit).toHaveBeenCalledTimes(2));expect(submit.mock.calls[1][0]).toEqual(submit.mock.calls[0][0])})
 it('does not reveal the writing answer and cannot submit blank strokes',()=>{render(<LiteracyPlayer question={{id:'w',interaction:'handwriting',stem:{text:'山',audio:'audio'}}} mediaResolver={x=>String(x)} onSubmit={vi.fn()}/>);expect(screen.queryByText('山')).not.toBeInTheDocument();expect(screen.getByRole('button',{name:'写完了'})).toBeDisabled()})
})
it('normalizes captured strokes, keeps retry payload and clears the pad',async()=>{
 vi.stubGlobal('PointerEvent',MouseEvent)
 const submit=vi.fn().mockResolvedValue({correct:false,canRetry:true})
 render(<LiteracyPlayer question={{id:'w',interaction:'handwriting',stem:{audio:'audio'}}} mediaResolver={x=>String(x)} onSubmit={submit}/>)
 const pad=screen.getByLabelText('写字板')
 vi.spyOn(pad,'getBoundingClientRect').mockReturnValue({left:10,top:20,width:200,height:200,right:210,bottom:220,x:10,y:20,toJSON(){}})
 fireEvent.pointerDown(pad,{clientX:30,clientY:60,pointerId:1});fireEvent.pointerMove(pad,{clientX:110,clientY:120,pointerId:1});fireEvent.pointerUp(pad,{pointerId:1})
 fireEvent.click(screen.getByRole('button',{name:'写完了'}))
 await waitFor(()=>expect(submit).toHaveBeenCalledTimes(1))
 expect(submit.mock.calls[0][0]).toMatchObject({kind:'handwriting',hintsUsed:0,strokes:[[{x:.1,y:.2},{x:.5,y:.5}]]})
 expect(submit.mock.calls[0][0].strokes[0][1].t).toBeGreaterThanOrEqual(submit.mock.calls[0][0].strokes[0][0].t)
 await screen.findByText('不太像，再听一次再写');fireEvent.click(screen.getByRole('button',{name:'再写一次'}));expect(screen.getByRole('button',{name:'写完了'})).toBeDisabled();expect(screen.getByText('先听读音，再写到格子里')).toBeInTheDocument()
})

it('highlights a correct selected option without an answer id and places feedback above options',async()=>{
 render(<LiteracyPlayer question={question} mediaResolver={x=>String(x)} onSubmit={vi.fn().mockResolvedValue({correct:true})}/>);
 const option=screen.getByRole('button',{name:'选项 2'});fireEvent.click(option);
 const feedback=await screen.findByText('答对啦');expect(option).toHaveClass('lp-correct');
 expect(feedback.compareDocumentPosition(option)&Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
})
it('shows writing checking feedback, locks restart while pending, and resets the actual preview pad after success',async()=>{
 vi.stubGlobal('PointerEvent',MouseEvent)
 let finish!:(value:{correct:boolean})=>void;
 const submit=vi.fn().mockImplementation(()=>new Promise(resolve=>{finish=resolve}));
 render(<LiteracyPlayer mode="preview" question={{id:'w',interaction:'handwriting',stem:{text:'山',audio:'audio'}}} mediaResolver={x=>String(x)} onSubmit={submit}/>);
 expect(screen.getByText('先听读音，再写到格子里')).toBeInTheDocument();
 expect(screen.getByRole('button',{name:'再听一次'})).toBeInTheDocument();
 const pad=screen.getByLabelText('写字板');
 fireEvent.pointerDown(pad,{clientX:1,clientY:1,pointerId:1});fireEvent.pointerMove(pad,{clientX:2,clientY:2,pointerId:1});fireEvent.pointerUp(pad,{pointerId:1});
 fireEvent.click(screen.getByRole('button',{name:'写完了'}));
 expect(screen.getByText('正在看你写的字……')).toBeInTheDocument();expect(screen.getByRole('button',{name:'再写一次'})).toBeDisabled();
 finish({correct:true});await screen.findByText('答对啦');
 fireEvent.click(screen.getByRole('button',{name:'再试一次'}));
 expect(screen.getByLabelText('写字板')).not.toBe(pad);expect(screen.getByRole('button',{name:'写完了'})).toBeDisabled();
 expect(screen.getByText('先听读音，再写到格子里')).toBeInTheDocument();expect(screen.queryByText('山')).not.toBeInTheDocument();
})
it('preserves handwriting for a network retry and removes the stale retry after restart',async()=>{
 vi.stubGlobal('PointerEvent',MouseEvent)
 const submit=vi.fn().mockRejectedValue(new Error('离线'))
 render(<LiteracyPlayer question={{id:'w',interaction:'handwriting',stem:{audio:'audio'}}} mediaResolver={x=>String(x)} onSubmit={submit}/>)
 const pad=screen.getByLabelText('写字板')
 fireEvent.pointerDown(pad,{clientX:1,clientY:1,pointerId:1});fireEvent.pointerMove(pad,{clientX:2,clientY:2,pointerId:1});fireEvent.pointerUp(pad,{pointerId:1})
 fireEvent.click(screen.getByRole('button',{name:'写完了'}));await screen.findByText('离线')
 fireEvent.click(screen.getByRole('button',{name:'重试提交'}));await waitFor(()=>expect(submit).toHaveBeenCalledTimes(2));await screen.findByText('离线')
 expect(submit.mock.calls[1][0]).toEqual(submit.mock.calls[0][0])
 fireEvent.click(screen.getByRole('button',{name:'再写一次'}));expect(screen.queryByRole('button',{name:'重试提交'})).not.toBeInTheDocument();expect(screen.queryByText('离线')).not.toBeInTheDocument();expect(screen.getByRole('button',{name:'写完了'})).toBeDisabled()
})

it('does not autoplay preview handwriting, stops replaced audio and pauses on unmount',()=>{
 const play=vi.spyOn(HTMLMediaElement.prototype,'play').mockResolvedValue()
 const pause=vi.spyOn(HTMLMediaElement.prototype,'pause').mockImplementation(()=>{})
 play.mockClear();pause.mockClear()
 const {unmount}=render(<LiteracyPlayer mode="preview" question={{id:'w',interaction:'handwriting',stem:{audio:'audio'}}} mediaResolver={x=>String(x)} onSubmit={vi.fn()}/>);
 expect(play).not.toHaveBeenCalled()
 fireEvent.click(screen.getByRole('button',{name:'再听一次'}));expect(play).toHaveBeenCalledTimes(1)
 fireEvent.click(screen.getByRole('button',{name:'再听一次'}));expect(pause).toHaveBeenCalledTimes(1)
 unmount();expect(pause).toHaveBeenCalledTimes(2)
})
it('autoplays practice once and stops it when switching questions',()=>{
 const play=vi.spyOn(HTMLMediaElement.prototype,'play').mockResolvedValue()
 const pause=vi.spyOn(HTMLMediaElement.prototype,'pause').mockImplementation(()=>{})
 play.mockClear();pause.mockClear()
 const props={mediaResolver:(x:unknown)=>String(x),onSubmit:vi.fn()}
 const writing={id:'w',interaction:'handwriting' as const,stem:{audio:'audio'}}
 const {rerender}=render(<LiteracyPlayer {...props} question={writing}/>);
 expect(play).toHaveBeenCalledTimes(1)
 rerender(<LiteracyPlayer {...props} question={writing}/>);expect(play).toHaveBeenCalledTimes(1)
 rerender(<LiteracyPlayer {...props} question={question}/>);expect(pause).toHaveBeenCalledTimes(1)
})
it('ignores a replaced playback rejection',async()=>{
 let reject!:(error:Error)=>void
 const play=vi.spyOn(HTMLMediaElement.prototype,'play').mockImplementationOnce(()=>new Promise((_,r)=>{reject=r})).mockResolvedValue()
 vi.spyOn(HTMLMediaElement.prototype,'pause').mockImplementation(()=>{})
 render(<LiteracyPlayer mode="preview" question={{id:'w',interaction:'handwriting',stem:{audio:'audio'}}} mediaResolver={x=>String(x)} onSubmit={vi.fn()}/>);
 fireEvent.click(screen.getByRole('button',{name:'再听一次'}));fireEvent.click(screen.getByRole('button',{name:'再听一次'}));
 reject(new Error('old playback cancelled'));await waitFor(()=>expect(play).toHaveBeenCalled());
 expect(screen.queryByText('读音无法播放，请重试')).not.toBeInTheDocument()
})
