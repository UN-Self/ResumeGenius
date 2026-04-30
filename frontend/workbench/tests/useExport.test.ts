import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'

// Mock request function from api-client — must be before useExport import
vi.mock('@/lib/api-client', () => ({
  request: vi.fn(),
}))

import { useExport } from '@/hooks/useExport'
import { request } from '@/lib/api-client'
const mockRequest = vi.mocked(request)

// Mock fetch for download step
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

describe('useExport', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockFetch.mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(new Blob(['pdf-data'], { type: 'application/pdf' })),
    })
  })

  it('polls with correct task ID (template literal interpolation)', async () => {
    // The bug was using single quotes: '/tasks/${taskId}' instead of `/tasks/${taskId}`
    // This test verifies the polling URL contains the actual task ID

    mockRequest.mockImplementation((url: string) => {
      if (url.includes('/drafts/1/export')) {
        return Promise.resolve({
          task_id: 'abc-123',
          status: 'pending',
          progress: 0,
        })
      }
      // Polling call - should use actual task ID
      if (url.includes('/tasks/abc-123') && !url.includes('/file')) {
        return Promise.resolve({
          task_id: 'abc-123',
          status: 'completed',
          progress: 100,
        })
      }
      throw new Error('Unexpected URL: ' + url)
    })

    const { result } = renderHook(() => useExport({ pollInterval: 100, maxPollDuration: 5000 }))

    await act(async () => {
      await result.current.exportPdf(1, '<p>test</p>', 'resume')
    })

    // Verify polling used actual task ID in URL, not literal '${taskId}'
    const pollingCalls = mockRequest.mock.calls.filter(
      ([url]: [string]) => url.includes('/tasks/') && !url.includes('/export') && !url.includes('/file')
    )
    expect(pollingCalls.length).toBeGreaterThan(0)
    expect(pollingCalls[0][0]).toContain('abc-123')
    expect(pollingCalls[0][0]).not.toContain('${')
  })

  it('handles failed status correctly (not "faild")', async () => {
    mockRequest.mockImplementation((url: string) => {
      if (url.includes('/drafts/1/export')) {
        return Promise.resolve({
          task_id: 'task-fail',
          status: 'pending',
          progress: 0,
        })
      }
      if (url.includes('/tasks/task-fail') && !url.includes('/file')) {
        return Promise.resolve({
          task_id: 'task-fail',
          status: 'failed',
          error: 'render error',
        })
      }
      throw new Error('Unexpected URL: ' + url)
    })

    const { result } = renderHook(() => useExport({ pollInterval: 100, maxPollDuration: 5000 }))

    await act(async () => {
      try {
        await result.current.exportPdf(1, '<p>test</p>')
      } catch {
        // expected
      }
    })

    expect(result.current.status).toBe('failed')
    expect(result.current.error).toBe('render error')
  })
})
