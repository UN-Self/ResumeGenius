import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import EditorPage from '@/pages/EditorPage'
import { server } from './setup'

describe('EditorPage', () => {
  describe('Empty state', () => {
    it('shows the empty state when draft html is missing', async () => {
      // Override the default handler to return empty content
      server.use(
        http.get('/api/v1/drafts/:draftId', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              id: 1,
              project_id: 1,
              html_content: '',
              updated_at: '2026-04-28T12:00:00Z',
            },
            message: 'ok',
          })
        })
      )

      render(<EditorPage />)
      await waitFor(() => {
        expect(screen.getByText('暂无简历内容')).toBeInTheDocument()
      })
    })
  })

  describe('Error state', () => {
    it('shows the error state when load fails', async () => {
      // Override the default handler to return an error
      server.use(
        http.get('/api/v1/drafts/:draftId', () => {
          return HttpResponse.json({
            code: 50000,
            data: null,
            message: 'Internal Server Error',
          })
        })
      )

      render(<EditorPage />)
      await waitFor(() => {
        expect(screen.getByText('加载失败')).toBeInTheDocument()
      })
    })
  })
})