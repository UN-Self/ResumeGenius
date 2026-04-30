import { useState, useCallback, useEffect } from 'react'
import { intakeApi, ApiError, type Project, type Asset } from '@/lib/api-client'

interface UseProjectDataReturn {
  project: Project | null
  assets: Asset[]
  loading: boolean
  error: string
  reload: () => void
}

export function useProjectData(pid: number): UseProjectDataReturn {
  const [project, setProject] = useState<Project | null>(null)
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [version, setVersion] = useState(0)

  // Keep fetch logic in a callback to avoid stale closures
  const doFetch = useCallback(async () => {
    try {
      setLoading(true)
      const [proj, asts] = await Promise.all([
        intakeApi.getProject(pid),
        intakeApi.listAssets(pid),
      ])
      setProject(proj)
      setAssets(asts)
      setError('')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [pid])

  // Schedule fetch via requestAnimationFrame so setState calls happen in the RAF
  // callback (async context), not in the effect's synchronous body.
  // This satisfies react-hooks/set-state-in-effect.
  useEffect(() => {
    let cancelled = false
    const id = requestAnimationFrame(() => {
      if (!cancelled) doFetch()
    })
    return () => {
      cancelled = true
      cancelAnimationFrame(id)
    }
  }, [doFetch, version])

  const reload = useCallback(() => { setVersion((v) => v + 1) }, [])

  return { project, assets, loading, error, reload }
}
