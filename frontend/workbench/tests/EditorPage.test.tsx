import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import EditorPage from '@/pages/EditorPage'
import { server } from './setup'

function renderWithRouter(initialEntry = '/projects/1/edit') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/projects/:projectId" element={<div data-testid="project-detail">Project Detail Page</div>} />
        <Route path="/projects/:projectId/edit" element={<EditorPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('EditorPage', () => {
  describe('Route guard', () => {
    it('redirects to project detail when no current_draft_id', async () => {
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
        expect(screen.getByTestId('project-detail')).toBeInTheDocument()
      })
    })
  })

  describe('Editor loads', () => {
    it('renders editor when project has current_draft_id', async () => {
      server.use(
        http.get('/api/v1/projects/:projectId', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              id: 1,
              title: 'Test Project',
              status: 'active',
              current_draft_id: 1,
              created_at: '2026-04-28T12:00:00Z',
            },
            message: 'ok',
          })
        }),
        http.get('/api/v1/drafts/1', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              id: 1,
              project_id: 1,
              html_content: '<p>Hello</p>',
              updated_at: '2026-04-28T12:00:00Z',
            },
            message: 'ok',
          })
        }),
        http.post('/api/v1/parsing/parse', () => {
          return HttpResponse.json({
            code: 0,
            data: { parsed_contents: [{ asset_id: 1, type: 'text', label: 'test', text: 'content' }] },
            message: 'ok',
          })
        })
      )

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })
    })
  })

  describe('Panel collapse', () => {
    it('renders collapse buttons for left and right panels', async () => {
      server.use(
        http.get('/api/v1/projects/:projectId', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              id: 1,
              title: 'Test Project',
              status: 'active',
              current_draft_id: 1,
              created_at: '2026-04-28T12:00:00Z',
            },
            message: 'ok',
          })
        }),
        http.get('/api/v1/drafts/1', () => {
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
        }),
        http.post('/api/v1/parsing/parse', () => {
          return HttpResponse.json({
            code: 0,
            data: { parsed_contents: [] },
            message: 'ok',
          })
        })
      )

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })

      expect(screen.getByLabelText('收起左面板')).toBeInTheDocument()
      expect(screen.getByLabelText('收起右面板')).toBeInTheDocument()
    })

    it('collapses and expands left panel on button click', async () => {
      server.use(
        http.get('/api/v1/projects/:projectId', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              id: 1,
              title: 'Test Project',
              status: 'active',
              current_draft_id: 1,
              created_at: '2026-04-28T12:00:00Z',
            },
            message: 'ok',
          })
        }),
        http.get('/api/v1/drafts/1', () => {
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
        }),
        http.post('/api/v1/parsing/parse', () => {
          return HttpResponse.json({
            code: 0,
            data: { parsed_contents: [] },
            message: 'ok',
          })
        })
      )

      const user = userEvent.setup()
      renderWithRouter()

      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })

      // Collapse left panel
      await user.click(screen.getByLabelText('收起左面板'))
      expect(screen.getByLabelText('展开左面板')).toBeInTheDocument()

      // Expand left panel
      await user.click(screen.getByLabelText('展开左面板'))
      expect(screen.getByLabelText('收起左面板')).toBeInTheDocument()
    })
  })
})
