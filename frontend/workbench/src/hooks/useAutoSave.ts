import { useState, useRef, useCallback, useEffect } from 'react'

export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

interface UseAutoSaveOptions {
  save: (html: string) => Promise<void>
  saveUrl?: string
  /** Transform HTML before saving (e.g. reconstruct full document). Applied in both normal saves and best-effort beforeunload saves. */
  reconstruct?: (html: string) => string
  debounceMs?: number
  maxRetries?: number
}

interface UseAutoSaveReturn {
  scheduleSave: (html: string) => void
  flush: () => Promise<void>
  retry: () => void
  status: SaveStatus
  lastSavedAt: Date | null
  error: Error | null
}

export function useAutoSave({
  save,
  saveUrl,
  reconstruct,
  debounceMs = 2000,
  maxRetries = 4 // 3 retries after initial = 4 total attempts
}: UseAutoSaveOptions): UseAutoSaveReturn {
  const [status, setStatus] = useState<SaveStatus>('idle')
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null)
  const [error, setError] = useState<Error | null>(null)

  const pendingHtml = useRef<string | null>(null)
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const retryCount = useRef(0)
  const retryTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isSaving = useRef(false)
  const currentSaveHtml = useRef<string | null>(null)
  const activeSave = useRef<Promise<void> | null>(null)

  const clearTimers = () => {
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current)
      debounceTimer.current = null
    }
    if (retryTimer.current) {
      clearTimeout(retryTimer.current)
      retryTimer.current = null
    }
  }

  const doSave = useCallback((html: string): Promise<void> => {
    if (isSaving.current) return activeSave.current ?? Promise.resolve()
    isSaving.current = true
    currentSaveHtml.current = html
    setStatus('saving')

    const run = (async () => {
      try {
        await save(reconstruct ? reconstruct(html) : html)
        setStatus('saved')
        setLastSavedAt(new Date())
        setError(null)
        retryCount.current = 0
        if (pendingHtml.current === html) {
          pendingHtml.current = null
        }

        // Reset to idle after 5 seconds
        setTimeout(() => {
          setStatus('idle')
        }, 5000)
      } catch (err) {
        retryCount.current++

        if (retryCount.current < maxRetries) {
          // Exponential backoff: 1s, 2s, 4s
          const delay = Math.pow(2, retryCount.current - 1) * 1000
          setStatus('saving')
          retryTimer.current = setTimeout(() => {
            doSave(html) // Use captured html, not pendingHtml
          }, delay)
        } else {
          setStatus('error')
          setError(err as Error)
        }
      } finally {
        isSaving.current = false
        currentSaveHtml.current = null
        activeSave.current = null
      }
    })()

    activeSave.current = run
    return run
  }, [save, reconstruct, maxRetries])

  const scheduleSave = useCallback((html: string) => {
    retryCount.current = 0
    pendingHtml.current = html
    clearTimers()

    debounceTimer.current = setTimeout(() => {
      if (pendingHtml.current !== null) {
        doSave(pendingHtml.current)
        // Note: we don't clear pendingHtml here - let doSave clear it on success
        // This allows retry to work if needed
      }
    }, debounceMs)
  }, [doSave, debounceMs])

  const flush = useCallback(async (): Promise<void> => {
    clearTimers()
    if (activeSave.current) {
      await activeSave.current
    }
    if (pendingHtml.current !== null) {
      const htmlToSave = pendingHtml.current
      currentSaveHtml.current = htmlToSave
      await doSave(htmlToSave)
    }
  }, [doSave])

  const retry = useCallback(() => {
    // Retry with the current save html if available, otherwise use pending
    const htmlToRetry = currentSaveHtml.current || pendingHtml.current
    if (htmlToRetry !== null) {
      retryCount.current = 0
      doSave(htmlToRetry)
    }
  }, [doSave])

  // Best-effort save using keepalive fetch (survives tab close)
  const bestEffortSave = useCallback((html: string) => {
    if (!saveUrl) return
    const htmlToSend = reconstruct ? reconstruct(html) : html
    fetch(saveUrl, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ html_content: htmlToSend }),
      keepalive: true,
    }).catch(() => {})
  }, [saveUrl, reconstruct])

  // Cleanup on unmount - flush pending saves
  useEffect(() => {
    const handleBeforeUnload = () => {
      if (pendingHtml.current !== null) {
        bestEffortSave(pendingHtml.current)
      }
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload)
      clearTimers()
      if (pendingHtml.current !== null) {
        bestEffortSave(pendingHtml.current)
      }
    }
  }, [bestEffortSave])

  return { scheduleSave, flush, retry, status, lastSavedAt, error }
}
