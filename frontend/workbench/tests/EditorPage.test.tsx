import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import EditorPage from '@/pages/EditorPage'
import { server } from './setup'

function renderWithRouter(initialEntry = '/projects/1/edit') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/projects/:projectId/edit" element={<EditorPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('EditorPage', () => {
  describe('Empty state', () => {
    it('shows empty state when project has no current draft', async () => {
      server.use(
        http.get('/api/v1/projects/:projectId', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              id: 1,
              title: 'Test Project',
              status: 'active',
              current_draft_id: null,
              created_at: '2026-04-28T12:00:00Z',
            },
            message: 'ok',
          })
        })
      )

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByText('暂无简历内容')).toBeInTheDocument()
      })
    })

    it('shows empty state when draft html is empty', async () => {
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

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByText('暂无简历内容')).toBeInTheDocument()
      })
    })
  })

  describe('Error state', () => {
    it('shows error state when project fetch fails', async () => {
      server.use(
        http.get('/api/v1/projects/:projectId', () => {
          return HttpResponse.json({
            code: 10001,
            data: null,
            message: 'project not found',
          })
        })
      )

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByText('加载失败')).toBeInTheDocument()
      })
    })

    it('shows error state when draft fetch fails', async () => {
      server.use(
        http.get('/api/v1/drafts/:draftId', () => {
          return HttpResponse.json({
            code: 50000,
            data: null,
            message: 'Internal Server Error',
          })
        })
      )

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByText('加载失败')).toBeInTheDocument()
      })
    })
  })
})
