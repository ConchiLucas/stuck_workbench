import { useRef } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { literacyApi } from '../api/literacy'
import type { PracticeTask } from '../api/types'
import { useChildStore } from '../store/childStore'

export function TasksPage() {
  const [search]=useSearchParams()
  const type=search.get('questionType')
  const childId = useChildStore((s) => s.childId)
  const navigate = useNavigate()
  const claims = useRef(new Map<string, string>())
  const tasks = useQuery({ queryKey: ['practice-tasks', childId], queryFn: () => literacyApi.tasks(childId) })
  const claim = useMutation({
    mutationFn: async (task: PracticeTask) => {
      if (task.planId && task.planStatus !== 'done') return { plan: { id: task.planId } }
      const key = `${childId}:${task.id}:${task.revisionId}`
      if (!claims.current.has(key)) claims.current.set(key, crypto.randomUUID())
      return literacyApi.claimTask(childId, task.id, task.revisionId, claims.current.get(key)!)
    },
    onSuccess: (detail) => navigate(`/practice/${detail.plan.id}`),
  })
  const visible=(tasks.data?.items??[]).filter(task=>!type||task.questionTypes?.includes(type))
  return <section className="tasks-page">
    <Link to="/">返回首页</Link><h1>练习任务</h1>
    {tasks.isLoading && <p>正在准备练习……</p>}
    {tasks.isError && <button className="retry-card" onClick={() => tasks.refetch()}>练习没加载出来，点这里重试</button>}
    {tasks.isSuccess && visible.length === 0 && <p>还没有新的练习任务</p>}
    <div className="task-list">{visible.map((task) => <article className="task-card" key={task.id}>
      <div><h2>{task.title}</h2><p>{task.targetCount} 题{task.kind === 'review' ? ' · 复习练习' : ''}{(task.questionTypes?.length??0)>1?' · 混合题包（完整领取）':''}</p></div>
      <button className="primary-button" disabled={claim.isPending} onClick={() => claim.mutate(task)}>
        {task.planId ? task.planStatus === 'done' ? '再练一次' : '继续练习' : '开始练习'}
      </button>
    </article>)}</div>
    {claim.isError && <p role="alert">{claim.error.message || '暂时无法开始，请重试'}</p>}
  </section>
}
