import { useState, useEffect, useCallback } from 'react'
import { renderApi, type Version } from '@/lib/api-client'

type PreviewMode = 'editing' | 'previewing'

export interface UseVersionsReturn {
  versions: Version[]
  loading: boolean
  previewMode: PreviewMode
  previewVersion: Version | null
  previewHtml: string | null
  refreshList: () => Promise<void>
  startPreview: (version: Version) => Promise<void>
  exitPreview: () => void
  createSnapshot: (label: string) => Promise<void>
  rollback: () => Promise<string>
}

export function useVersions(draftId: number | null): UseVersionsReturn {
  const [versions, setVersions] = useState<Version[]>([])
  const [loading, setLoading] = useState(true)
  const [previewMode, setPreviewMode] = useState<PreviewMode>('editing')
  const [previewVersion, setPreviewVersion] = useState<Version | null>(null)
  const [previewHtml, setPreviewHtml] = useState<string | null>(null)

  const refreshList = useCallback(async () => {
    if (!draftId) return
    try {
      const data = await renderApi.listVersions(draftId)
      setVersions(data.items)
    } finally {
      setLoading(false)
    }
  }, [draftId])

  useEffect(() => {
    refreshList()
  }, [refreshList])

  const startPreview = useCallback(async (version: Version) => {
    if (!draftId) return
    const detail = await renderApi.getVersion(draftId, version.id)
    setPreviewVersion(version)
    setPreviewHtml(detail.html_snapshot)
    setPreviewMode('previewing')
  }, [draftId])

  const exitPreview = useCallback(() => {
    setPreviewMode('editing')
    setPreviewVersion(null)
    setPreviewHtml(null)
  }, [])

  const createSnapshot = useCallback(async (label: string) => {
    if (!draftId) return
    await renderApi.createVersion(draftId, label)
    await refreshList()
  }, [draftId, refreshList])

  const rollback = useCallback(async (): Promise<string> => {
    if (!draftId || !previewVersion) throw new Error('No version to rollback to')
    await renderApi.rollback(draftId, previewVersion.id)
    const html = previewHtml!
    exitPreview()
    await refreshList()
    return html
  }, [draftId, previewVersion, previewHtml, exitPreview, refreshList])

  return {
    versions,
    loading,
    previewMode,
    previewVersion,
    previewHtml,
    refreshList,
    startPreview,
    exitPreview,
    createSnapshot,
    rollback,
  }
}
