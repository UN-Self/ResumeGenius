import { describe, it, expect } from 'vitest'
import { apiClient } from '@/lib/api-client'
import './setup'

describe('Draft API', () => {
  it('returns the sample draft html through the mock handler', async () => {
    const draft = await apiClient.get<{
      id: number
      project_id: number
      html_content: string
      updated_at: string
    }>('/drafts/1')

    expect(draft.html_content).toContain('Sample Draft')
  })

  it('updates draft html through the mock handler', async () => {
    const newHtml = '<h1>Updated Content</h1><p>New paragraph</p>'

    const result = await apiClient.put<{
      id: number
      updated_at: string
    }>('/drafts/1', { html_content: newHtml })

    expect(result.id).toBe(1)
    expect(result.updated_at).toBeDefined()
  })
})
