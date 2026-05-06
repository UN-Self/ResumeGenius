import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { intakeApi } from '@/lib/api-client'
import ProjectDetail from '@/pages/ProjectDetail'

vi.mock('@/lib/api-client', () => ({
  intakeApi: {
    getProject: vi.fn(),
    listAssets: vi.fn(),
    deleteAsset: vi.fn(),
    deleteProject: vi.fn(),
    uploadFile: vi.fn(),
    createGitRepo: vi.fn(),
    createNote: vi.fn(),
    updateNote: vi.fn(),
  },
  parsingApi: {
    parseProject: vi.fn(),
  },
  request: vi.fn(),
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

describe('ProjectDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders project title and assets', async () => {
    vi.mocked(intakeApi.getProject).mockResolvedValue(mockProject)
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

    render(
      <MemoryRouter initialEntries={['/projects/1']}>
        <Routes>
          <Route path="/projects/:projectId" element={<ProjectDetail />} />
          <Route path="/projects/:projectId/edit" element={<div>Editor Page</div>} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('前端工程师简历')).toBeInTheDocument()
      expect(screen.getByText('求职意向')).toBeInTheDocument()
    })
  })

  it('renders empty asset state', async () => {
    vi.mocked(intakeApi.getProject).mockResolvedValue(mockProject)
    vi.mocked(intakeApi.listAssets).mockResolvedValue([])

    render(
      <MemoryRouter initialEntries={['/projects/1']}>
        <Routes>
          <Route path="/projects/:projectId" element={<ProjectDetail />} />
          <Route path="/projects/:projectId/edit" element={<div>Editor Page</div>} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('还没有添加任何资料')).toBeInTheDocument()
    })
  })

  it('shows delete confirmation on project delete button click', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.getProject).mockResolvedValue(mockProject)
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

    render(
      <MemoryRouter initialEntries={['/projects/1']}>
        <Routes>
          <Route path="/projects/:projectId" element={<ProjectDetail />} />
          <Route path="/projects/:projectId/edit" element={<div>Editor Page</div>} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('前端工程师简历')).toBeInTheDocument()
    })

    const deleteBtn = screen.getByText('删除项目')
    await user.click(deleteBtn)

    const dialogDeleteBtn = screen.getAllByRole('button', { name: '删除' }).pop()!
    await user.click(dialogDeleteBtn)

    await waitFor(() => {
      expect(screen.getAllByText(/确认删除/).length).toBeGreaterThan(0)
    })
  })

  it('navigates to edit page when clicking start editing button', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.getProject).mockResolvedValue(mockProject)
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

    render(
      <MemoryRouter initialEntries={['/projects/1']}>
        <Routes>
          <Route path="/projects/:projectId" element={<ProjectDetail />} />
          <Route path="/projects/:projectId/edit" element={<div>Editor Page</div>} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('前端工程师简历')).toBeInTheDocument()
    })

    const editBtn = screen.getByText('开始编辑')
    await user.click(editBtn)

    await waitFor(() => {
      expect(screen.getByText('Editor Page')).toBeInTheDocument()
    })
  })

  it('shows continue editing button when current draft already exists', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.getProject).mockResolvedValue({ ...mockProject, current_draft_id: 88 })
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)

    render(
      <MemoryRouter initialEntries={['/projects/1']}>
        <Routes>
          <Route path="/projects/:projectId" element={<ProjectDetail />} />
          <Route path="/projects/:projectId/edit" element={<div>Editor Page</div>} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('继续编辑')).toBeInTheDocument()
    })

    await user.click(screen.getByText('继续编辑'))

    await waitFor(() => {
      expect(screen.getByText('Editor Page')).toBeInTheDocument()
    })
  })
})
