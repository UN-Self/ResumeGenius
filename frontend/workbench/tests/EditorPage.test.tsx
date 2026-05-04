import { describe, expect, it } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
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

function mockEditorLoad(overrides?: {
  currentDraftId?: number | null
  assets?: Array<Record<string, unknown>>
  htmlContent?: string
}) {
  const currentDraftId = overrides?.currentDraftId !== undefined ? overrides.currentDraftId : 1
  const assets = overrides?.assets ?? []
  const htmlContent = overrides?.htmlContent ?? '<p>Hello</p>'

  server.use(
    http.get('/api/v1/projects/:projectId', () => {
      return HttpResponse.json({
        code: 0,
        data: {
          id: 1,
          title: 'Test Project',
          status: 'active',
          current_draft_id: currentDraftId,
          created_at: '2026-04-28T12:00:00Z',
        },
        message: 'ok',
      })
    }),
    http.get('/api/v1/assets', () => {
      return HttpResponse.json({
        code: 0,
        data: assets,
        message: 'ok',
      })
    }),
    http.get('/api/v1/drafts/1', () => {
      return HttpResponse.json({
        code: 0,
        data: {
          id: 1,
          project_id: 1,
          html_content: htmlContent,
          updated_at: '2026-04-28T12:00:00Z',
        },
        message: 'ok',
      })
    })
  )
}

describe('EditorPage', () => {
  describe('Route guard', () => {
    it('redirects to project detail when no current_draft_id', async () => {
      mockEditorLoad({ currentDraftId: null })

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('project-detail')).toBeInTheDocument()
      })
    })
  })

  describe('Editor loads', () => {
    it('renders editor when project has current_draft_id', async () => {
      mockEditorLoad()

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })
    })
  })

  describe('Export button', () => {
    it('renders export button that is not disabled when editor is loaded', async () => {
      mockEditorLoad({ htmlContent: '<p>Hello</p>' })

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })

      const exportBtn = screen.getByText('\u5bfc\u51fa PDF')
      expect(exportBtn).toBeInTheDocument()
      expect(exportBtn).not.toBeDisabled()
    })
  })

  describe('Panel collapse', () => {
    it('renders collapse buttons for left and right panels', async () => {
      mockEditorLoad()

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })

      expect(screen.getByLabelText('\u6536\u8d77\u5de6\u4fa7\u680f')).toBeInTheDocument()
      expect(screen.getByLabelText('\u6536\u8d77\u53f3\u4fa7\u680f')).toBeInTheDocument()
    })

    it('collapses and expands left panel on button click', async () => {
      mockEditorLoad()

      const user = userEvent.setup()
      renderWithRouter()

      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })

      await user.click(screen.getByLabelText('\u6536\u8d77\u5de6\u4fa7\u680f'))
      expect(screen.getByLabelText('\u5c55\u5f00\u5de6\u4fa7\u680f')).toBeInTheDocument()

      await user.click(screen.getByLabelText('\u5c55\u5f00\u5de6\u4fa7\u680f'))
      expect(screen.getByLabelText('\u6536\u8d77\u5de6\u4fa7\u680f')).toBeInTheDocument()
    })
  })

  describe('Sidebar actions', () => {
    it('renders upload button in left panel', async () => {
      mockEditorLoad()

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })

      expect(screen.getByText('\u4e0a\u4f20\u6587\u4ef6')).toBeInTheDocument()
    })

    it('shows dirty-state hint and reparse action for parsed assets', async () => {
      mockEditorLoad({
        assets: [
          {
            id: 11,
            project_id: 1,
            type: 'resume_pdf',
            label: 'Resume.pdf',
            content: 'Parsed content',
            metadata: {
              parsing: {
                updated_by_user: true,
                last_parsed_at: '2026-05-04T06:00:00Z',
              },
            },
            created_at: '2026-05-04T06:00:00Z',
          },
        ],
      })

      renderWithRouter()
      await waitFor(() => {
        expect(screen.getByTestId('a4-canvas')).toBeInTheDocument()
      })

      expect(screen.getByText('有 1 项素材已手动修改，重新解析会覆盖当前正文。')).toBeInTheDocument()
      expect(screen.getByText('重新解析')).toBeInTheDocument()
    })
  })
})
