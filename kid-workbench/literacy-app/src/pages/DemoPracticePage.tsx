import { GlyphSenseQuestion } from '@kid-workbench/literacy-player'
import { useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { PracticeStage } from '../components/PracticeStage'
import { ListenIcon } from '../components/Icons'
import { ListenWritePad } from '../components/ListenWritePad'
import { OptionTile } from '../components/OptionTile'
import { Tianzige } from '../components/Tianzige'
import { useQuery } from '@tanstack/react-query'
import { literacyApi } from '../api/literacy'
import { useLiteracyAudio } from '../hooks/useLiteracyAudio'

type DemoType = 'glyph' | 'sense' | 'write'

type DemoOption = { label: string }

type DemoQuestion = {
  options: DemoOption[]
  answer: number
  character: string
}

const pictures: DemoOption[] = [
  { label: '山' },
  { label: '水' },
  { label: '日' },
  { label: '木' },
]

const demoBanks: Record<DemoType, DemoQuestion[]> = {
  glyph: [
    { character: '山', answer: 0, options: pictures },
    { character: '水', answer: 1, options: pictures },
    { character: '日', answer: 2, options: pictures },
    { character: '木', answer: 3, options: pictures },
  ],
  sense: [
    { character: '水', answer: 1, options: [{ label: '火' }, { label: '水' }, { label: '土' }, { label: '石' }] },
    { character: '山', answer: 0, options: [{ label: '山' }, { label: '水' }, { label: '日' }, { label: '木' }] },
    { character: '日', answer: 2, options: [{ label: '火' }, { label: '木' }, { label: '日' }, { label: '石' }] },
    { character: '木', answer: 3, options: [{ label: '山' }, { label: '水' }, { label: '土' }, { label: '木' }] },
  ],
  write: [
    { options: [], answer: 0, character: '一' },
    { options: [], answer: 0, character: '二' },
    { options: [], answer: 0, character: '三' },
    { options: [], answer: 0, character: '十' },
  ],
}

export function DemoPracticePage() {
  const type = useParams().type as string
  const audio = useLiteracyAudio()
  // App 自己组织练习，仅从素材目录解析媒体，不领取后台任务。
  const materials = useQuery({
    queryKey: ['practice-material-catalog'],
    queryFn: async () => {
      const modules = await literacyApi.modules()
      return (await Promise.all(modules.map(module => literacyApi.items(module.code)))).flat()
    },
  })
  const material = (character: string) => materials.data?.find(item => item.character === character)
  const play = (character: string) => {
    const item = material(character)
    if (item?.hasSpeech) audio.play(literacyApi.speechUrl(item.kpId))
  }
  const picture = (character: string, kind: 'glyph' | 'sense') => {
    const item = material(character)
    return item && (kind === 'glyph' ? item.hasGlyph : item.hasSense)
      ? (kind === 'glyph' ? literacyApi.glyphUrl(item.kpId) : literacyApi.senseUrl(item.kpId))
      : undefined
  }
  const bank = demoBanks[type as DemoType]
  const [index, setIndex] = useState(0)
  const [selected, setSelected] = useState<number | null>(null)
  const [written, setWritten] = useState(false)

  if (!bank) return <Navigate to="/" replace />
  const question = bank[index]
  const correct = type === 'write' ? written : selected === question.answer
  const answered = selected !== null || written
  const pictureChoices = type === 'glyph'
  const go = (next: number) => {
    setIndex(next)
    setSelected(null)
    setWritten(false)
  }

  return <PracticeStage current={index + 1} total={bank.length} onPrev={() => go(index - 1)} onNext={() => go(index + 1)}>
    {materials.isError && <button className="retry-card" onClick={() => void materials.refetch()}>素材没加载出来，点这里再试一次</button>}
    {audio.state === 'error' && <p role="alert">读音无法播放，请再试一次</p>}
    {type === 'glyph' ? <GlyphSenseQuestion
      question={{id: `glyph:${index}`, questionType: 'glyph_sense', interaction: 'choice',
        stem: {text: question.character, image: picture(question.character, 'glyph')},
        options: question.options.map((option, i) => ({id: String(i), text: option.label,
          image: picture(option.label, 'sense'), audio: material(option.label)?.hasSpeech ? literacyApi.speechUrl(material(option.label)!.kpId) : undefined}))}}
      mediaResolver={ref => typeof ref === 'string' ? ref : undefined}
      selectedOptionId={selected === null ? undefined : String(selected)} correct={selected === null ? undefined : correct}
      onPick={id => setSelected(Number(id))} footer={correct ? <Link to="/">回首页</Link> : undefined}
    /> : type === 'write'
      ? <ListenWritePad key={question.character} character={question.character} kpId={material(question.character)?.kpId} passed={written} showHomeLink onPass={() => setWritten(true)} />
      : <>
        <div className="stage-body">
          <div className="stage-visual">
            <Tianzige className="stage">{picture(question.character, type === 'sense' ? 'sense' : 'glyph') ? <img src={picture(question.character, type === 'sense' ? 'sense' : 'glyph')} alt={type === 'sense' ? '题目图片' : question.character} /> : <span>图片素材暂不可用</span>}</Tianzige>
            {type === 'sense' ? <button className="listen-inline" type="button" aria-label="播放读音" onClick={() => play(question.character)}><ListenIcon size={22} /></button> : null}
          </div>
        </div>
        <section className="choice-row" aria-label="当前题目">
          <p className={`feedback-bar demo-feedback${answered ? ' show' : ''}${answered && correct ? ' is-correct' : ''}${answered && !correct ? ' is-wrong' : ''}`}>
            {answered && <>
              <span>{correct ? '答对啦' : '再试一次'}</span>
              {correct && <Link to="/">回首页</Link>}
            </>}
          </p>
          <div className="option-grid">{question.options.map((option, optionIndex) =>
            <OptionTile
              key={`${question.character}-${option.label}`}
              label={option.label}
              imageSrc={pictureChoices ? picture(option.label, 'sense') : picture(option.label, 'glyph')}
              showPlay={pictureChoices}
              pressed={selected === optionIndex}
              tone={selected === optionIndex ? correct ? 'correct' : 'wrong' : undefined}
              onPick={() => setSelected(optionIndex)}
              onPlay={() => play(option.label)}
            />)}
          </div>
        </section>
      </>}
  </PracticeStage>
}
