import { useState, useCallback, useRef, useEffect } from 'react'
import { request } from '@/lib/api-client'

export type ExportStatus = 'idle' | 'exporting' | 'completed' | 'failed'

interface ExportTask {
  task_id: string
  status: string
  progress: number
  download_url?: string
  error?: string
}

interface UseExportOptions {
  pollInterval?: number
  maxPollDuration?: number
}

interface UseExportReturn {
  exportPdf: (draftId: number, filename?: string) => Promise<void>
  status: ExportStatus
  error: string | null
  progress: number
  message: string
}

export function useExport({
  pollInterval = 800,
  maxPollDuration = 30000,
}: UseExportOptions = {}): UseExportReturn {
  const [status, setStatus] = useState<ExportStatus>('idle')
  const [error, setError] = useState<string | null>(null)
  const [progress, setProgress] = useState(0)
  const [message, setMessage] = useState('')
  const abortRef = useRef(false)

  const clearState = useCallback(() => {
    abortRef.current = true
    setStatus('idle')
    setError(null)
    setProgress(0)
    setMessage('')
  }, [])

  const downloadFile = useCallback(async (taskId: string, filename: string) => {
    const res = await fetch(`/api/v1/tasks/${taskId}/file`, { credentials: 'include' })
    if (!res.ok) throw new Error('下载失败')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename.endsWith('.pdf') ? filename : `${filename}.pdf`
    a.click()
    URL.revokeObjectURL(url)
  }, [])

  const pollUntilDone = useCallback(async (
    taskId: string,
    filename: string,
  ): Promise<void> => {
    const deadline = Date.now() + maxPollDuration

    while (!abortRef.current && Date.now() < deadline) {
      await new Promise((r) => setTimeout(r, pollInterval))

      if (abortRef.current) return

      const task = await request<ExportTask>(`/tasks/${taskId}`)
      setProgress(Math.max(10, Math.min(task.progress || 0, 95)))
      setMessage(task.status === 'processing' ? '正在渲染简历页面并生成 PDF...' : '正在排队准备导出...')
      if (task.status === 'completed') {
        setProgress(96)
        setMessage('PDF 已生成，正在准备下载...')
        await downloadFile(taskId, filename)
        return
      }
      if (task.status === 'failed') {
        throw new Error(task.error)
      }
    }

    throw new Error('导出超时')
  }, [pollInterval, maxPollDuration, downloadFile])

  useEffect(() => clearState, [clearState])

  const exportPdf = useCallback(async (
    draftId: number,
    filename = 'resume',
  ) => {
    abortRef.current = false
    setStatus('exporting')
    setError(null)
    setProgress(5)
    setMessage('正在创建导出任务...')

    try {
      const task = await request<ExportTask>(`/drafts/${draftId}/export`, {
        method: 'POST',
      })

      setProgress(Math.max(8, task.progress || 8))
      setMessage('导出任务已创建，等待后端处理...')
      await pollUntilDone(task.task_id, filename)

      if (!abortRef.current) {
        setProgress(100)
        setMessage('导出完成')
        setStatus('completed')
        setTimeout(() => setStatus('idle'), 3000)
      }
    } catch (err) {
      if (abortRef.current) return
      setStatus('failed')
      setError(err instanceof Error ? err.message : '导出失败')
      setProgress(0)
      setMessage(err instanceof Error ? err.message : '导出失败')
      setTimeout(() => {
        setStatus('idle')
        setError(null)
        setMessage('')
      }, 5000)
    }
  }, [pollUntilDone])

  return { exportPdf, status, error, progress, message }
}
