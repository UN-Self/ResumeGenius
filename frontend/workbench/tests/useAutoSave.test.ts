import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAutoSave } from '@/hooks/useAutoSave'

// Use fake timers
vi.useFakeTimers()

describe('useAutoSave', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('debounces draft saves for 2 seconds', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoSave({ save }))

    act(() => result.current.scheduleSave('<p>changed</p>'))
    expect(save).not.toHaveBeenCalled()

    // Advance time and flush promises
    await act(async () => {
      vi.advanceTimersByTime(2000)
      await Promise.resolve() // flush microtasks
    })
    expect(save).toHaveBeenCalledTimes(1)
    expect(save).toHaveBeenCalledWith('<p>changed</p>')
  })

  it('retries failed saves 3 times with exponential backoff', async () => {
    const save = vi.fn().mockRejectedValue(new Error('network error'))
    const { result } = renderHook(() => useAutoSave({ save }))

    act(() => result.current.scheduleSave('<p>changed</p>'))

    // First call happens after 2s debounce
    await act(async () => {
      vi.advanceTimersByTime(2000)
      await Promise.resolve()
    })
    expect(save).toHaveBeenCalledTimes(1)

    // Retry 1: after 1s (2^0 * 1000)
    await act(async () => {
      vi.advanceTimersByTime(1000)
      await Promise.resolve()
    })
    expect(save).toHaveBeenCalledTimes(2)

    // Retry 2: after 2s (2^1 * 1000)
    await act(async () => {
      vi.advanceTimersByTime(2000)
      await Promise.resolve()
    })
    expect(save).toHaveBeenCalledTimes(3)

    // Retry 3: after 4s (2^2 * 1000) - final retry
    await act(async () => {
      vi.advanceTimersByTime(4000)
      await Promise.resolve()
    })
    expect(save).toHaveBeenCalledTimes(4)

    // After 3 retries total, should give up (status should be 'error')
    expect(result.current.status).toBe('error')
  })

  it('returns correct status transitions', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoSave({ save }))

    expect(result.current.status).toBe('idle')

    act(() => result.current.scheduleSave('<p>test</p>'))
    expect(result.current.status).toBe('idle')

    // Advance time and flush microtasks
    await act(async () => {
      vi.advanceTimersByTime(2000)
      await Promise.resolve()
    })
    expect(result.current.status).toBe('saved')

    // After 5 seconds, should reset to idle
    await act(async () => {
      vi.advanceTimersByTime(5000)
      await Promise.resolve()
    })
    expect(result.current.status).toBe('idle')
  })

  it('flush cancels debounce and saves immediately', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoSave({ save }))

    act(() => result.current.scheduleSave('<p>changed</p>'))
    expect(save).not.toHaveBeenCalled()

    await act(async () => {
      result.current.flush()
      await Promise.resolve()
    })
    expect(save).toHaveBeenCalledTimes(1)
    expect(save).toHaveBeenCalledWith('<p>changed</p>')
  })

  it('retry does nothing when no pending save', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoSave({ save }))

    // Call retry with no pending save
    act(() => result.current.retry())
    expect(save).not.toHaveBeenCalled()
  })

  it('flush returns a promise that resolves after save completes', async () => {
    let saveResolve!: () => void
    const save = vi.fn().mockImplementation(
      () => new Promise<void>((r) => { saveResolve = r }),
    )
    const { result } = renderHook(() => useAutoSave({ save }))

    act(() => result.current.scheduleSave('<p>changed</p>'))

    // flush() should return a Promise (not void/undefined)
    const flushPromise = result.current.flush()
    expect(flushPromise).toBeInstanceOf(Promise)
    expect(save).toHaveBeenCalledTimes(1)

    // Resolve the pending save
    await act(async () => {
      saveResolve()
      await flushPromise
    })
  })

  it('flush waits for active save before saving latest pending html', async () => {
    let firstSaveResolve!: () => void
    const save = vi.fn().mockImplementation((html: string) => {
      if (html === '<p>first</p>') {
        return new Promise<void>((resolve) => { firstSaveResolve = resolve })
      }
      return Promise.resolve()
    })
    const { result } = renderHook(() => useAutoSave({ save }))

    act(() => result.current.scheduleSave('<p>first</p>'))
    await act(async () => {
      vi.advanceTimersByTime(2000)
      await Promise.resolve()
    })
    expect(save).toHaveBeenCalledTimes(1)

    act(() => result.current.scheduleSave('<p>latest</p>'))
    const flushPromise = result.current.flush()
    expect(save).toHaveBeenCalledTimes(1)

    await act(async () => {
      firstSaveResolve()
      await flushPromise
    })

    expect(save).toHaveBeenCalledTimes(2)
    expect(save).toHaveBeenNthCalledWith(1, '<p>first</p>')
    expect(save).toHaveBeenNthCalledWith(2, '<p>latest</p>')
  })

  it('applies reconstruct function before saving', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const reconstruct = vi.fn((html: string) => `<full>${html}</full>`)
    const { result } = renderHook(() => useAutoSave({ save, reconstruct }))

    act(() => result.current.scheduleSave('<p>body</p>'))

    await act(async () => {
      vi.advanceTimersByTime(2000)
      await Promise.resolve()
    })

    expect(reconstruct).toHaveBeenCalledWith('<p>body</p>')
    expect(save).toHaveBeenCalledWith('<full><p>body</p></full>')
  })

  it('applies reconstruct in flush', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const reconstruct = vi.fn((html: string) => `<full>${html}</full>`)
    const { result } = renderHook(() => useAutoSave({ save, reconstruct }))

    act(() => result.current.scheduleSave('<p>body</p>'))

    await act(async () => {
      result.current.flush()
      await Promise.resolve()
    })

    expect(reconstruct).toHaveBeenCalledWith('<p>body</p>')
    expect(save).toHaveBeenCalledWith('<full><p>body</p></full>')
  })

  it('setLastSavedAt updates after successful save', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoSave({ save }))

    expect(result.current.lastSavedAt).toBeNull()

    act(() => result.current.scheduleSave('<p>test</p>'))
    await act(async () => {
      vi.advanceTimersByTime(2000)
      await Promise.resolve()
    })
    expect(result.current.lastSavedAt).not.toBeNull()
  })
})
