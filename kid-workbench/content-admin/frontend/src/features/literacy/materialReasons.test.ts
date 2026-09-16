import {expect,it} from 'vitest'
import {materialReasonLabel} from '@kid-workbench/literacy-player'
it('shows material problems in plain Chinese and hides unknown codes',()=>{
 expect(materialReasonLabel('missing_writing_template')).toBe('缺少书写模板')
 expect(materialReasonLabel('missing_sense_image')).toBe('缺少义图')
 expect(materialReasonLabel('future_internal_code')).toBe('暂不可用')
 expect(materialReasonLabel('识字App尚未部署该题型')).toBe('识字App尚未部署该题型')
})
