import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { intakeApi, parsingApi } from '@/lib/api-client'
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
    generateProject: vi.fn(),
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

    // First click shows the dialog, second click enters confirming state
    const dialogDeleteBtn = screen.getAllByRole('button', { name: '删除' }).pop()!
    await user.click(dialogDeleteBtn)

    await waitFor(() => {
      expect(screen.getAllByText(/确认删除/).length).toBeGreaterThan(0)
    })
  })

  it('calls generateProject and navigates to edit page when no draft exists', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.getProject).mockResolvedValue(mockProject)
    vi.mocked(intakeApi.listAssets).mockResolvedValue(mockAssets)
    vi.mocked(parsingApi.generateProject).mockResolvedValue({
      draft_id: 99,
      version_id: 101,
      html_content: '<p>Generated</p>',
    })

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

    const parseBtn = screen.getByText('下一步：生成初稿')
    await user.click(parseBtn)

    await waitFor(() => {
      expect(parsingApi.generateProject).toHaveBeenCalledWith(1)
      expect(screen.getByText('Editor Page')).toBeInTheDocument()
    })
  })

  it('navigates directly to edit page when current draft already exists', async () => {
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
      expect(screen.getByText('进入编辑页')).toBeInTheDocument()
    })

    await user.click(screen.getByText('进入编辑页'))

    await waitFor(() => {
      expect(parsingApi.generateProject).not.toHaveBeenCalled()
      expect(screen.getByText('Editor Page')).toBeInTheDocument()
    })
  })
})
