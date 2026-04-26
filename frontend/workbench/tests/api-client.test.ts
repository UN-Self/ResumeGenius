import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient, ApiError } from '@/lib/api-client'

describe('apiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('calls GET with correct path', async () => {
    const mock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: { id: 1 } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await apiClient.get('/drafts/1')
    expect(mock).toHaveBeenCalledWith('/api/v1/drafts/1', expect.objectContaining({ method: 'GET' }))
    expect(result).toEqual({ id: 1 })
  })

  it('calls POST with correct path and body', async () => {
    const mock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: { id: 2 } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await apiClient.post('/projects', { title: 'test' })
    expect(mock).toHaveBeenCalledWith('/api/v1/projects', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ title: 'test' }),
    }))
    expect(result).toEqual({ id: 2 })
  })

  it('throws ApiError on non-zero code', async () => {
    const mock = vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ code: 40001, message: '参数错误' }),
    })
    vi.stubGlobal('fetch', mock)

    await expect(apiClient.get('/bad')).rejects.toThrow(ApiError)
    await expect(apiClient.get('/bad')).rejects.toMatchObject({
      message: '参数错误',
    })
  })

  it('ApiError has code property', () => {
    const err = new ApiError(50001, '服务器错误')
    expect(err.code).toBe(50001)
    expect(err.message).toBe('服务器错误')
  })
})
