import { Fragment, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import type { QuestionTypeExample as ExampleData } from '../content/questionTypePrototype'

const OPERATOR = /^[+\u2212\-=]$/

function EquationText({ text }: { text: string }) {
  return <>{text.split(/(\s+)/).map((part, index) => OPERATOR.test(part)
    ? <i className="eq-op" aria-hidden="true" key={index}>{part}</i>
    : part)}</>
}

function EquationPrompt({ text, className }: { text: string; className: string }) {
  return <div className={className} aria-label={text}><EquationText text={text} /></div>
}

function optionClass(option: string) {
  return `detail-example-option${/^[○△□◇◯]$/.test(option) ? ' is-glyph' : ''}`
}

function Options({ options, href }: { options: string[]; href?: string }) {
  return <div className="detail-example-options">{options.map((option, index) => href
    ? <Link className={optionClass(option)} key={`${option}-${index}`} to={href}>{option}</Link>
    : <span className={optionClass(option)} key={`${option}-${index}`}>{option}</span>)}</div>
}

function JudgeMark({ kind }: { kind: 'yes' | 'no' }) {
  return <svg className={`judge-mark is-${kind}`} viewBox="0 0 88 88" aria-hidden="true">
    <rect x="4" y="4" width="80" height="80" rx="22" />
    {kind === 'yes'
      ? <path d="M26 46.5 38.5 59 64 30" />
      : <path d="M30 30l28 28M58 30 30 58" />}
  </svg>
}

function JudgeOptions({ href }: { href?: string }) {
  const marks = [{ kind: 'yes' as const, label: '对' }, { kind: 'no' as const, label: '错' }]
  return <div className="detail-example-options">{marks.map((mark) => href
    ? <Link className="detail-example-option is-judge" aria-label={mark.label} to={href} key={mark.label}><JudgeMark kind={mark.kind} /></Link>
    : <span className="detail-example-option is-judge" aria-label={mark.label} key={mark.label}><JudgeMark kind={mark.kind} /></span>)}</div>
}

function Stage({ visual, prompt, children }: { visual: ReactNode; prompt?: string; children: ReactNode }) {
  return <>
    <div className="kid-stage-visual">{visual}</div>
    <div className="kid-stage-choices">
      {prompt && <p className="kid-question-prompt">{prompt}</p>}
      {children}
    </div>
  </>
}

export function QuestionTypeExample({ example, nextHref }: { example: ExampleData; nextHref?: string }) {
  if (example.kind === 'objects') return <Stage visual={<div className="object-preview">{example.groups.map((group, index) => <Fragment key={`${group}-${index}`}>{index > 0 && <i className="object-op" aria-hidden="true" />}<span>{group}</span></Fragment>)}</div>} prompt={example.prompt}>
    <Options options={example.options} href={nextHref} />
  </Stage>

  if (example.kind === 'judgement') {
    if (example.statements.length === 1) {
      return <Stage visual={<EquationPrompt text={example.statements[0]} className="detail-example-prompt judgement" />}>
        <JudgeOptions href={nextHref} />
      </Stage>
    }

    return <Stage visual={<p className="kid-question-prompt kid-stage-title">{example.prompt}</p>}>
      <div className="judgement-list">{example.statements.map((statement) => nextHref
        ? <Link className="detail-example-option judgement-option" aria-label={statement} to={nextHref} key={statement}><EquationText text={statement} /></Link>
        : <span className="detail-example-option judgement-option" aria-label={statement} key={statement}><EquationText text={statement} /></span>)}</div>
    </Stage>
  }

  if (example.kind === 'shape-sort') return <Stage visual={<p className="kid-question-prompt kid-stage-title">{example.prompt}</p>}>
    <div className="shape-buckets">{example.buckets.map((bucket) => <section key={bucket.label}><strong>{bucket.label}</strong><div>{bucket.items.map((item, index) => <span key={`${item}-${index}`}>{item}</span>)}</div></section>)}</div>
  </Stage>

  return <Stage visual={<EquationPrompt text={example.prompt} className={`detail-example-prompt ${example.kind}`} />}>
    <Options options={example.options} href={nextHref} />
  </Stage>
}
