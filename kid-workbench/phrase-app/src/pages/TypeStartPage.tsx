import { useEffect, useRef } from 'react'
import { Navigate, useNavigate, useParams } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { phraseApi } from '../api/phrase'
import type { PhraseCode } from '../api/types'
import { isPhraseCode } from '../content/phraseTypes'
import { useChildStore } from '../store/childStore'

export function TypeStartPage() {
  const { code } = useParams()
  const childId = useChildStore((s) => s.childId)
  const navigate = useNavigate()
  const started = useRef(false)
  const create = useMutation({
    mutationFn: (questionCode: PhraseCode) => phraseApi.createPlan(childId, { mode: 'type', questionCode, count: 4 }),
    onSuccess: (detail) => navigate(`/practice/${detail.plan.id}/1`, { replace: true }),
  })
  const start = create.mutate

  useEffect(() => {
    if (!isPhraseCode(code) || started.current) return
    started.current = true
    start(code)
  }, [code, start])

  if (!isPhraseCode(code)) return <Navigate to="/" replace />
  if (create.isError) {
    return (
      <section className="practice-page">
        <button className="retry-card" type="button" onClick={() => create.mutate(code)}>
          {create.error instanceof Error ? create.error.message : '出题没有成功，点下面再试一次'}
        </button>
      </section>
    )
  }
  return <section className="practice-page"><div className="skeleton-block">正在准备今天的短句……</div></section>
}
