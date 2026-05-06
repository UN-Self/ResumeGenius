import { describe, it, expect, vi, beforeAll, afterAll, beforeEach } from 'vitest'
import { renderApi } from '@/lib/api-client'
import { server } from './setup'

// Stop MSW server for this test file — we mock fetch directly
beforeAll(() => server.close())
afterAll(() => server.listen({ onUnhandledRequest: 'error' }))

const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

function mockResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve({ code: 0, data, message: 'ok' }),
  } as Response)
}

beforeEach(() => {
  mockFetch.mockReset()
})

describe('renderApi', () => {
  describe('listVersions', () => {
    it('calls GET /drafts/:draftId/versions', async () => {
      const items = [
        { id: 1, label: 'v1', created_at: '2026-05-06T10:00:00Z' },
      ]
      mockFetch.mockReturnValue(mockResponse({ items, total: 1 }))

      const result = await renderApi.listVersions(42)

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/versions',
        expect.objectContaining({ credentials: 'include' }),
      )
      expect(result.items).toHaveLength(1)
      expect(result.items[0].label).toBe('v1')
    })
  })

  describe('getVersion', () => {
    it('calls GET /drafts/:draftId/versions/:versionId', async () => {
      const detail = {
        id: 1,
        label: 'v1',
        html_snapshot: '<html>resume</html>',
        created_at: '2026-05-06T10:00:00Z',
      }
      mockFetch.mockReturnValue(mockResponse(detail))

      const result = await renderApi.getVersion(42, 1)

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/versions/1',
        expect.objectContaining({ credentials: 'include' }),
      )
      expect(result.html_snapshot).toBe('<html>resume</html>')
    })
  })

  describe('createVersion', () => {
    it('calls POST /drafts/:draftId/versions with label', async () => {
      const created = { id: 3, label: '校招版', created_at: '2026-05-06T11:00:00Z' }
      mockFetch.mockReturnValue(mockResponse(created))

      const result = await renderApi.createVersion(42, '校招版')

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/versions',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ label: '校招版' }),
        }),
      )
      expect(result.label).toBe('校招版')
    })
  })

  describe('rollback', () => {
    it('calls POST /drafts/:draftId/rollback with version_id', async () => {
      const resp = {
        draft_id: 42,
        updated_at: '2026-05-06T12:00:00Z',
        new_version_id: 5,
        new_version_label: '回退到版本 2026-05-06 10:00:00',
        new_version_created_at: '2026-05-06T12:00:00Z',
      }
      mockFetch.mockReturnValue(mockResponse(resp))

      const result = await renderApi.rollback(42, 1)

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/drafts/42/rollback',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ version_id: 1 }),
        }),
      )
      expect(result.new_version_id).toBe(5)
    })
  })
})
