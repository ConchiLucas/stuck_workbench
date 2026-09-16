const labels:Record<string,string>={
 missing_text:'缺少汉字信息',missing_sense_image:'缺少义图',missing_glyph_image:'缺少字图',missing_speech_audio:'缺少读音',missing_writing_template:'缺少书写模板',
 invalid_sense_media:'义图无法读取',invalid_glyph_media:'字图无法读取',invalid_speech_media:'读音无法读取',invalid_writing_template:'书写模板无效',missing_or_invalid_glyph_media:'缺少有效字图',media_update_incomplete:'素材更新尚未完成',
 material_missing:'缺少所需素材',renderer_unavailable:'练习界面尚未就绪',evaluator_unavailable:'判题服务尚未就绪',
}
export function materialReasonLabel(reason:string):string{return labels[reason]??(/[\u3400-\u9fff]/.test(reason)?reason:'暂不可用')}
