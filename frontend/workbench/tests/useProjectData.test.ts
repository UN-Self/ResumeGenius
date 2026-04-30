import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useProjectData } from '@/hooks/useProjectData'
import { intakeApi, ApiError } from '@/lib/api-client'

vi.mock('@/lib/api-client', () => ({
  intakeApi: {
    getProject: vi.fn(),
    listAssets: vi.fn(),
  },
  ApiError: class extends Error {
    code: number
    constructor(c: number, m: string) {
      super(m)
      this.code = c
    }
  },
}))

const mockProject = {
  id: 1,
  title: '前端工程师简历',
  status: 'active',
  current_draft_id: null,
  created_at: '2026-04-28T00:00:00Z',
}

const mockAssets = [
  { id: 1, project_id: 1, type: 'resume_pdf', uri: 'uploads/1/resume.pdf', created_at: '2026-04-28T00:00:00Z' },
  { id: 2, project_id: 1, type: 'note', content: '目标岗位', label: '求职意向', created_at: '2026-04-28T00:00:00Z' },
]

describe('useProjectData', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  // The hook uses requestAnimationFrame internally.
  // With fake timers, we must advance timers to fire the RAF callback,
  // then flush microtasks (promises) to resolve the API calls.
  async function flushAsync() {
    await act(async () => {
      await vi.advanceTimersToNextTimerAsync()
    })
  }

  it('loads project and assets on mount', async () => {
    vi.mocked(intakeApi.getProject).mockResolvedValue(mockProject)
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

    const { result } = renderHook(() => useProjectData(1))
    await flushAsync()

    expect(intakeApi.getProject).toHaveBeenCalledWith(1)
    expect(intakeApi.listAssets).toHaveBeenCalledWith(1)
    expect(result.current.project).toEqual(mockProject)
    expect(result.current.assets).toEqual(mockAssets)
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBe('')
  })

  it('sets error when API fails', async () => {
    vi.mocked(intakeApi.getProject).mockRejectedValue(new ApiError(10100, '项目不存在'))
    vi.mocked(intakeApi.listAssets).mockResolvedValue([])

    const { result } = renderHook(() => useProjectData(999))
    await flushAsync()

    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBe('项目不存在')
    expect(result.current.project).toBeNull()
  })

  it('uses default error message for non-ApiError exceptions', async () => {
    vi.mocked(intakeApi.getProject).mockRejectedValue(new Error('Network error'))
    vi.mocked(intakeApi.listAssets).mockResolvedValue([])

    const { result } = renderHook(() => useProjectData(1))
    await flushAsync()

    expect(result.current.error).toBe('加载失败')
  })

  it('reload() triggers a fresh fetch', async () => {
    const updatedProject = { ...mockProject, title: 'Updated Title' }
    vi.mocked(intakeApi.getProject)
      .mockResolvedValueOnce(mockProject)
      .mockResolvedValueOnce(updatedProject)
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

    const { result } = renderHook(() => useProjectData(1))
    await flushAsync()

    expect(result.current.project?.title).toBe('前端工程师简历')
    expect(intakeApi.getProject).toHaveBeenCalledTimes(1)

    await act(async () => {
      result.current.reload()
    })
    await flushAsync()

    expect(intakeApi.getProject).toHaveBeenCalledTimes(2)
    expect(result.current.project?.title).toBe('Updated Title')
  })

  it('refetches when pid changes', async () => {
    const project2 = { ...mockProject, id: 2, title: 'Another Project' }
    vi.mocked(intakeApi.getProject)
      .mockResolvedValueOnce(mockProject)
      .mockResolvedValueOnce(project2)
    vi.mocked(intakeApi.listAssets)
      .mockResolvedValueOnce(mockAssets)
      .mockResolvedValueOnce([])

    const { result, rerender } = renderHook(
      ({ pid }) => useProjectData(pid),
      { initialProps: { pid: 1 } },
    )
    await flushAsync()

    expect(result.current.project?.id).toBe(1)

    rerender({ pid: 2 })
    await flushAsync()

    expect(intakeApi.getProject).toHaveBeenCalledWith(2)
    expect(result.current.project?.id).toBe(2)
  })

  it('cancels pending fetch on unmount without crashing', async () => {
    let resolveProject: (v: unknown) => void
    vi.mocked(intakeApi.getProject).mockReturnValue(new Promise((r) => { resolveProject = r }))
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

    const { unmount } = renderHook(() => useProjectData(1))
    await flushAsync()

    unmount()

    resolveProject!(mockProject)
    await act(async () => {
      await Promise.resolve()
    })

    expect(intakeApi.getProject).toHaveBeenCalledWith(1)
  })
})
