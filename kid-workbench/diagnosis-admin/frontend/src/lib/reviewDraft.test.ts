import { afterEach, expect, it, vi } from 'vitest'
import { readReviewDraft, writeReviewDraft, refreshReviewChoices } from './reviewDraft'
const item={key:'1:listen',kpId:1,subjectCode:'pinyin',count:3,mode:'mixed'}
afterEach(()=>{localStorage.clear();vi.restoreAllMocks()})
it('keeps drafts child scoped and rejects invalid stored limits',()=>{
 writeReviewDraft(1,[item]);expect(readReviewDraft(1)).toEqual([item]);expect(readReviewDraft(2)).toEqual([])
 localStorage.setItem('knowledge-review-draft:2',JSON.stringify([{...item,count:99}]))
 expect(readReviewDraft(2)).toEqual([])
})
it('restores choices with fresh evidence and drops unavailable candidates',async()=>{
 vi.spyOn(globalThis,'fetch').mockResolvedValue(new Response(JSON.stringify({items:[{key:'1:listen',kpId:1,subjectCode:'pinyin',title:'a',evidence:[{attemptId:9}]}],evidenceAsOf:'2026-09-13T00:00:00Z',hasMore:false})))
 const result=await refreshReviewChoices(1,[item,{...item,key:'1:shape'}])
 expect(result.choices['1:listen'].candidate.evidence[0].attemptId).toBe(9)
 expect(result.missing).toBe(1)
})
