import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import ProjectDetail from '@/pages/ProjectDetail'
import { intakeApi, parsingApi, ApiError } from '@/lib/api-client'
import { useProjectData } from '@/hooks/useProjectData'

vi.mock('@/lib/api-client', () => ({
  intakeApi: {
    uploadFile: vi.fn(),
    createGitRepo: vi.fn(),
    createNote: vi.fn(),
    updateNote: vi.fn(),
    deleteAsset: vi.fn(),
    deleteProject: vi.fn(),
  },
  parsingApi: {
    parseProject: vi.fn(),
  },
  ApiError: class extends Error {
    code: number
    constructor(code: number, message: string) {
      super(message)
      this.code = code
    }
  },
}))

vi.mock('@/hooks/useProjectData', () => ({
  useProjectData: vi.fn(),
}))

vi.mock('@/components/intake/AssetList', () => ({
  default: ({ onDelete }: { onDelete: (id: number) => void }) => (
    <button type="button" onClick={() => onDelete(11)}>
      mock-delete-asset
    </button>
  ),
}))

vi.mock('@/components/intake/UploadDialog', () => ({
  default: ({ open, onUpload }: { open: boolean; onUpload: (file: File) => Promise<void> }) => (
    open
      ? (
          <button
            type="button"
            onClick={() => onUpload(new File(['pdf-content'], 'resume.pdf', { type: 'application/pdf' }))}
          >
            mock-upload-submit
          </button>
        )
      : null
  ),
}))

vi.mock('@/components/intake/GitRepoDialog', () => ({
  default: ({ open, onSubmit }: { open: boolean; onSubmit: (repoUrl: string) => Promise<void> }) => (
    open
      ? (
          <button type="button" onClick={() => onSubmit('https://github.com/example/repo')}>
            mock-git-submit
          </button>
        )
      : null
  ),
}))

vi.mock('@/components/intake/NoteDialog', () => ({
  default: () => null,
}))

vi.mock('@/components/intake/DeleteConfirm', () => ({
  default: ({ open, onConfirm }: { open: boolean; onConfirm: () => Promise<void> }) => (
    open
      ? (
          <button type="button" onClick={() => void onConfirm()}>
            mock-confirm-delete
          </button>
        )
      : null
  ),
}))

vi.mock('@/components/ui/full-page-state', () => ({
  FullPageState: ({ message }: { message?: string }) => <div>{message ?? 'loading'}</div>,
}))

const reload = vi.fn()

describe('ProjectDetail action handlers', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    reload.mockReset()
    vi.mocked(useProjectData).mockReturnValue({
      project: {
        id: 1,
        title: 'Test Project',
        status: 'active',
        current_draft_id: null,
        created_at: '2026-05-05T00:00:00Z',
      },
      assets: [
        {
          id: 11,
          project_id: 1,
          type: 'resume_pdf',
          uri: 'user-1/hash.pdf',
          created_at: '2026-05-05T00:00:00Z',
        },
      ],
      loading: false,
      error: '',
      reload,
    })
  })

  function renderPage() {
    return render(
      <MemoryRouter initialEntries={['/projects/1']}>
        <Routes>
          <Route path="/projects/:projectId" element={<ProjectDetail />} />
          <Route path="/projects/:projectId/edit" element={<div>Editor</div>} />
        </Routes>
      </MemoryRouter>
    )
  }

  it('reloads and surfaces an error when upload parsing fails', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.uploadFile).mockResolvedValue({
      id: 12,
      project_id: 1,
      type: 'resume_pdf',
      created_at: '2026-05-05T00:00:00Z',
    })
    vi.mocked(parsingApi.parseProject).mockRejectedValue(new Error('parse failed'))

    renderPage()

    await user.click(screen.getByRole('button', { name: /上传文件/i }))
    await user.click(screen.getByRole('button', { name: 'mock-upload-submit' }))

    await waitFor(() => {
      expect(reload).toHaveBeenCalledTimes(1)
      expect(screen.getByText('parse failed')).toBeInTheDocument()
    })
  })

  it('reloads and surfaces an error when creating a Git asset fails', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.createGitRepo).mockRejectedValue(new Error('create git failed'))

    renderPage()

    await user.click(screen.getByRole('button', { name: /接入 Git/i }))
    await user.click(screen.getByRole('button', { name: 'mock-git-submit' }))

    await waitFor(() => {
      expect(reload).toHaveBeenCalledTimes(1)
      expect(screen.getByText('create git failed')).toBeInTheDocument()
    })
  })

  it('shows delete errors and closes the confirmation dialog', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.deleteAsset).mockRejectedValue(new ApiError(1999, 'asset delete failed'))

    renderPage()

    await user.click(screen.getByRole('button', { name: 'mock-delete-asset' }))
    await user.click(screen.getByRole('button', { name: 'mock-confirm-delete' }))

    await waitFor(() => {
      expect(screen.getByText(/asset delete failed/)).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'mock-confirm-delete' })).not.toBeInTheDocument()
    })
  })
})
