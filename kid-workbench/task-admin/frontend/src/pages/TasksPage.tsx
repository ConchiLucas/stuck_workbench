import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import {
  createQuestionTask,
  deleteQuestionTask,
  getQuestionTask,
  listLiteracyModules,
  listQuestionTasks,
  publishQuestionTask,
  reshuffleQuestionTask,
  unpublishQuestionTask,
} from '../api/qtask'
import type { QuestionTask } from '../api/qtaskTypes'
import { QuizQuestionRow } from '../quiz/QuizQuestionRow'
import { qtaskItemToQuizQuestion } from '../quiz/qtaskToQuiz'

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}

export function TasksPage() {
  const queryClient = useQueryClient()
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [moduleCode, setModuleCode] = useState('')
  const [title, setTitle] = useState('')
  const [actionError, setActionError] = useState('')
  const [activeQuizId, setActiveQuizId] = useState<string | null>(null)

  const listQuery = useQuery({
    queryKey: ['question-tasks', 'literacy'],
    queryFn: () => listQuestionTasks({ subject: 'literacy' }),
  })

  const modulesQuery = useQuery({
    queryKey: ['question-tasks', 'literacy-modules'],
    queryFn: listLiteracyModules,
  })

  const detailQuery = useQuery({
    queryKey: ['question-tasks', selectedId],
    queryFn: () => getQuestionTask(selectedId!),
    enabled: selectedId != null,
  })

  const modules = useMemo(() => {
    const list = modulesQuery.data ?? []
    return [...list].sort((a, b) => a.order - b.order)
  }, [modulesQuery.data])

  const quizQuestions = useMemo(() => {
    const items = detailQuery.data?.items ?? []
    return items
      .map((it) => qtaskItemToQuizQuestion(it))
      .filter((q): q is NonNullable<typeof q> => q != null)
  }, [detailQuery.data?.items])

  const activeQuestion = quizQuestions.find((q) => q.id === activeQuizId) ?? null

  useEffect(() => {
    setActiveQuizId(null)
  }, [selectedId, detailQuery.data?.updatedAt])

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ['question-tasks'] })
  }

  const createMut = useMutation({
    mutationFn: () =>
      createQuestionTask({
        subjectCode: 'literacy',
        moduleCode,
        title: title.trim() || undefined,
      }),
    onSuccess: async (task) => {
      setActionError('')
      setShowCreate(false)
      setTitle('')
      await invalidate()
      setSelectedId(task.id)
    },
    onError: (err: Error) => setActionError(err.message),
  })

  const runAction = async (fn: () => Promise<QuestionTask | void>, stayOnDetail = true) => {
    setActionError('')
    try {
      const result = await fn()
      await invalidate()
      if (!stayOnDetail) {
        setSelectedId(null)
      } else if (result && typeof result === 'object' && 'id' in result) {
        await queryClient.setQueryData(['question-tasks', result.id], result)
      }
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err))
    }
  }

  if (selectedId != null) {
    const task = detailQuery.data
    return (
      <section className="literacy-page qtask-page">
        <div className="page-heading">
          <div>
            <div className="eyebrow">TASKS / LITERACY</div>
            <h1>{task?.title ?? '任务详情'}</h1>
            {task && !activeQuestion ? (
              <p className="page-description muted">
                两类题预览 · 看字选义 / 看义选字 · 左题干 / 右两列选项 · 点击进入详情试答
              </p>
            ) : null}
          </div>
          <div className="page-heading-actions">
            {activeQuestion ? (
              <button
                type="button"
                className="refresh-button"
                onClick={() => setActiveQuizId(null)}
              >
                ← 返回题目列表
              </button>
            ) : (
              <button type="button" className="refresh-button" onClick={() => setSelectedId(null)}>
                返回列表
              </button>
            )}
            {task?.status === 'draft' && !activeQuestion ? (
              <>
                <button
                  type="button"
                  className="refresh-button"
                  disabled={detailQuery.isFetching}
                  onClick={() => runAction(() => reshuffleQuestionTask(selectedId))}
                >
                  重新抽题
                </button>
                <button
                  type="button"
                  className="refresh-button"
                  onClick={() => runAction(() => publishQuestionTask(selectedId))}
                >
                  发布
                </button>
              </>
            ) : null}
            {task?.status === 'published' && !activeQuestion ? (
              <button
                type="button"
                className="refresh-button"
                onClick={() => runAction(() => unpublishQuestionTask(selectedId))}
              >
                撤回草稿
              </button>
            ) : null}
          </div>
        </div>

        {actionError ? <div className="error-panel">{actionError}</div> : null}
        {detailQuery.isLoading ? <div className="loading-panel">加载中…</div> : null}
        {detailQuery.isError ? (
          <div className="error-panel">
            {(detailQuery.error as Error).message || '加载失败'}
          </div>
        ) : null}

        {task && !activeQuestion ? (
          <div className="info-panel qtask-meta">
            <span>
              状态：<strong className={`qtask-status ${task.status}`}>{task.status}</strong>
            </span>
            <span>
              组：{task.moduleName} ({task.moduleCode})
            </span>
            <span>题数：{task.items?.length ?? task.targetCount}</span>
            <span>更新：{formatTime(task.updatedAt)}</span>
          </div>
        ) : null}

        {task ? (
          activeQuestion ? (
            <QuizQuestionRow question={activeQuestion} mode="detail" />
          ) : (
            <div className="quiz-list">
              {quizQuestions.map((q) => (
                <QuizQuestionRow
                  key={q.id}
                  question={q}
                  mode="list"
                  onOpen={() => setActiveQuizId(q.id)}
                />
              ))}
            </div>
          )
        ) : null}
      </section>
    )
  }

  return (
    <section className="literacy-page qtask-page">
      <div className="page-heading">
        <div>
          <div className="eyebrow">TASKS / LITERACY</div>
          <h1>识字任务</h1>
          <p className="page-description muted">从已有题库抽 10 题组成一份可发布的题包。</p>
        </div>
        <div className="page-heading-actions">
          <button
            type="button"
            className="refresh-button"
            onClick={() => {
              setShowCreate((v) => !v)
              setActionError('')
            }}
          >
            新建识字任务
          </button>
          <button
            type="button"
            className="refresh-button"
            disabled={listQuery.isFetching}
            onClick={() => listQuery.refetch()}
          >
            刷新
          </button>
        </div>
      </div>

      {actionError ? <div className="error-panel">{actionError}</div> : null}
      {listQuery.isError ? (
        <div className="error-panel">{(listQuery.error as Error).message || '加载失败'}</div>
      ) : null}

      {showCreate ? (
        <div className="info-panel qtask-create">
          <label>
            识字组
            <select
              value={moduleCode}
              onChange={(e) => setModuleCode(e.target.value)}
              disabled={modulesQuery.isLoading}
            >
              <option value="">选择组…</option>
              {modules.map((m) => (
                <option key={m.code} value={m.code}>
                  {m.name} ({m.code})
                </option>
              ))}
            </select>
          </label>
          <label>
            标题（可选）
            <input
              type="text"
              value={title}
              placeholder="默认：识字 · 组名"
              onChange={(e) => setTitle(e.target.value)}
            />
          </label>
          <button
            type="button"
            className="refresh-button"
            disabled={!moduleCode || createMut.isPending}
            onClick={() => createMut.mutate()}
          >
            {createMut.isPending ? '生成中…' : '生成 10 题'}
          </button>
        </div>
      ) : null}

      {listQuery.isLoading ? <div className="loading-panel">加载中…</div> : null}

      {!listQuery.isLoading && (listQuery.data?.length ?? 0) === 0 ? (
        <div className="empty-panel">暂无识字任务，点击「新建识字任务」生成题包。</div>
      ) : null}

      <div className="group-list">
        {(listQuery.data ?? []).map((task) => (
          <div key={task.id} className="group-card qtask-row">
            <div className="qtask-row-main">
              <strong>{task.title}</strong>
              <span className={`qtask-status ${task.status}`}>{task.status}</span>
              <span className="qtask-sub">
                {task.moduleName} · {task.targetCount} 题 · {formatTime(task.updatedAt)}
              </span>
            </div>
            <div className="qtask-row-actions">
              <button type="button" className="mini-btn" onClick={() => setSelectedId(task.id)}>
                查看
              </button>
              {task.status === 'draft' ? (
                <>
                  <button
                    type="button"
                    className="mini-btn"
                    onClick={() => runAction(() => publishQuestionTask(task.id), false)}
                  >
                    发布
                  </button>
                  <button
                    type="button"
                    className="mini-btn"
                    onClick={() =>
                      runAction(async () => {
                        await deleteQuestionTask(task.id)
                      }, false)
                    }
                  >
                    删除
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  className="mini-btn"
                  onClick={() => runAction(() => unpublishQuestionTask(task.id), false)}
                >
                  撤回
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
