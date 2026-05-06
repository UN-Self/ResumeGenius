import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'

vi.mock('@/lib/api-client', () => ({
  renderApi: {
    listVersions: vi.fn(),
    getVersion: vi.fn(),
    createVersion: vi.fn(),
    rollback: vi.fn(),
  },
}))

import { useVersions } from '@/hooks/useVersions'
import { renderApi } from '@/lib/api-client'

const mockListVersions = vi.mocked(renderApi.listVersions)
const mockGetVersion = vi.mocked(renderApi.getVersion)
const mockCreateVersion = vi.mocked(renderApi.createVersion)
const mockRollback = vi.mocked(renderApi.rollback)

const sampleVersions = [
  { id: 2, label: 'v2', created_at: '2026-05-06T10:10:00Z' },
  { id: 1, label: 'v1', created_at: '2026-05-06T10:00:00Z' },
]

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useVersions', () => {
  it('loads version list on mount', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.versions).toEqual(sampleVersions)
    expect(result.current.previewMode).toBe('editing')
  })

  it('enters preview mode on startPreview', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    const html = '<html>resume v1</html>'
    mockGetVersion.mockResolvedValue({
      ...sampleVersions[1],
      html_snapshot: html,
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(() => result.current.startPreview(sampleVersions[1]))

    expect(result.current.previewMode).toBe('previewing')
    expect(result.current.previewVersion).toEqual(sampleVersions[1])
    expect(result.current.previewHtml).toBe(html)
  })

  it('exits preview mode on exitPreview', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    mockGetVersion.mockResolvedValue({
      ...sampleVersions[1],
      html_snapshot: '<html>v1</html>',
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(() => result.current.startPreview(sampleVersions[1]))
    expect(result.current.previewMode).toBe('previewing')

    act(() => result.current.exitPreview())

    expect(result.current.previewMode).toBe('editing')
    expect(result.current.previewVersion).toBeNull()
    expect(result.current.previewHtml).toBeNull()
  })

  it('creates snapshot and refreshes list', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    mockCreateVersion.mockResolvedValue({
      id: 3,
      label: '校招版',
      created_at: '2026-05-06T11:00:00Z',
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(() => result.current.createSnapshot('校招版'))

    expect(mockCreateVersion).toHaveBeenCalledWith(1, '校招版')
    expect(mockListVersions).toHaveBeenCalledTimes(2) // mount + after create
  })

  it('rollback returns html and refreshes list', async () => {
    mockListVersions.mockResolvedValue({ items: sampleVersions, total: 2 })
    const rollbackHtml = '<html>restored</html>'
    mockGetVersion.mockResolvedValue({
      ...sampleVersions[1],
      html_snapshot: rollbackHtml,
    })
    mockRollback.mockResolvedValue({
      draft_id: 1,
      updated_at: '2026-05-06T12:00:00Z',
      new_version_id: 5,
      new_version_label: '回退到版本 2026-05-06 10:00:00',
      new_version_created_at: '2026-05-06T12:00:00Z',
    })

    const { result } = renderHook(() => useVersions(1))

    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(() => result.current.startPreview(sampleVersions[1]))

    const html = await act(() => result.current.rollback())

    expect(html).toBe(rollbackHtml)
    expect(result.current.previewMode).toBe('editing')
    expect(mockRollback).toHaveBeenCalledWith(1, 1)
  })

  it('handles null draftId gracefully', async () => {
    const { result } = renderHook(() => useVersions(null))

    await waitFor(() => expect(result.current.loading).toBe(true))

    expect(mockListVersions).not.toHaveBeenCalled()
    expect(result.current.versions).toEqual([])
  })
})
