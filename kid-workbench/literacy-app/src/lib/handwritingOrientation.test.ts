import {afterEach, expect, it, vi} from 'vitest'
import HanziWriter from 'hanzi-writer'
import water from '../../../shared-go/handwriting/testdata/水.json'
import de from '../../../shared-go/handwriting/testdata/的.json'
import {looksLikeCharacter} from './handwriting'

afterEach(()=>vi.restoreAllMocks())
const screenInk=(medians:number[][][])=>medians.map(stroke=>stroke.map(([x,y])=>({x:x*320/1024,y:(1024-y)*320/1024})))
it('compares screen ink with water medians in the same vertical direction',async()=>{
 vi.spyOn(HanziWriter,'loadCharacterData').mockResolvedValue(water)
 expect(await looksLikeCharacter(screenInk(water.medians),'水')).toEqual({ok:true,score:1})
})
it('does not accept 的 with its last stroke missing due to a flipped template',async()=>{
 vi.spyOn(HanziWriter,'loadCharacterData').mockResolvedValue(de)
 expect(await looksLikeCharacter(screenInk(de.medians).slice(0,-1),'的')).toEqual({ok:false,score:.5})
})
