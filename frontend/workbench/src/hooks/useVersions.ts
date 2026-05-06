import { useState, useEffect, useCallback } from 'react'
import { renderApi, type Version } from '@/lib/api-client'

type PreviewMode = 'editing' | 'previewing'

export interface UseVersionsReturn {
  versions: Version[]
  loading: boolean
  error: string | null
  previewMode: PreviewMode
  previewVersion: Version | null
  previewHtml: string | null
  refreshList: () => Promise<void>
  startPreview: (version: Version) => Promise<void>
  exitPreview: () => void
  createSnapshot: (label: string) => Promise<void>
  rollback: () => Promise<string>
  clearError: () => void
}

export function useVersions(draftId: number | null): UseVersionsReturn {
  const [versions, setVersions] = useState<Version[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [previewMode, setPreviewMode] = useState<PreviewMode>('editing')
  const [previewVersion, setPreviewVersion] = useState<Version | null>(null)
  const [previewHtml, setPreviewHtml] = useState<string | null>(null)

  const refreshList = useCallback(async () => {
    if (!draftId) return
    setLoading(true)
    setError(null)
    try {
      const data = await renderApi.listVersions(draftId)
      setVersions(data.items)
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载版本列表失败')
    } finally {
      setLoading(false)
    }
  }, [draftId])

  useEffect(() => {
    refreshList()
  }, [refreshList])

  const startPreview = useCallback(async (version: Version) => {
    if (!draftId) return
    setError(null)
    try {
      const detail = await renderApi.getVersion(draftId, version.id)
      setPreviewVersion(version)
      setPreviewHtml(detail.html_snapshot)
      setPreviewMode('previewing')
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载版本内容失败')
    }
  }, [draftId])

  const exitPreview = useCallback(() => {
    setPreviewMode('editing')
    setPreviewVersion(null)
    setPreviewHtml(null)
  }, [])

  const createSnapshot = useCallback(async (label: string) => {
    if (!draftId) return
    setError(null)
    try {
      await renderApi.createVersion(draftId, label)
      await refreshList()
    } catch (e) {
      setError(e instanceof Error ? e.message : '保存快照失败')
      throw e
    }
  }, [draftId, refreshList])

  const rollback = useCallback(async (): Promise<string> => {
    if (!draftId || !previewVersion) throw new Error('No version to rollback to')
    if (!previewHtml) throw new Error('Preview HTML is not loaded')
    setError(null)
    try {
      await renderApi.rollback(draftId, previewVersion.id)
      const html = previewHtml
      exitPreview()
      await refreshList()
      return html
    } catch (e) {
      setError(e instanceof Error ? e.message : '回滚失败')
      throw e
    }
  }, [draftId, previewVersion, previewHtml, exitPreview, refreshList])

  const clearError = useCallback(() => setError(null), [])

  return {
    versions,
    loading,
    error,
    previewMode,
    previewVersion,
    previewHtml,
    refreshList,
    startPreview,
    exitPreview,
    createSnapshot,
    rollback,
    clearError,
  }
}
