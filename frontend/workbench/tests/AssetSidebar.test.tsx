import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AssetSidebar from '@/components/intake/AssetSidebar'
import { intakeApi, parsingApi } from '@/lib/api-client'

vi.mock('@/lib/api-client', () => ({
  intakeApi: {
    uploadFile: vi.fn(),
    createGitRepo: vi.fn(),
    createNote: vi.fn(),
    updateAsset: vi.fn(),
    deleteAsset: vi.fn(),
  },
  parsingApi: {
    parseProject: vi.fn(),
  },
}))

vi.mock('@/components/intake/AssetList', () => ({
  default: ({ assets, onEditAsset }: { assets: Array<Record<string, unknown>>; onEditAsset: (asset: Record<string, unknown>) => void }) => (
    <button type="button" onClick={() => onEditAsset(assets[0])}>
      mock-edit-asset
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

vi.mock('@/components/intake/NoteDialog', () => ({
  default: ({ open, onSubmit }: { open: boolean; onSubmit: (content: string, label: string) => Promise<void> }) => (
    open
      ? (
          <button type="button" onClick={() => onSubmit('note content', 'Note label')}>
            mock-note-submit
          </button>
        )
      : null
  ),
}))

vi.mock('@/components/intake/AssetEditorDialog', () => ({
  default: ({ open, onSubmit }: { open: boolean; onSubmit: (content: string, label: string) => Promise<void> }) => (
    open
      ? (
          <button type="button" onClick={() => onSubmit('updated content', 'Updated label')}>
            mock-edit-submit
          </button>
        )
      : null
  ),
}))

vi.mock('@/components/intake/DeleteConfirm', () => ({
  default: () => null,
}))

vi.mock('@/components/intake/GitRepoDialog', () => ({
  default: () => null,
}))

const baseAssets = [
  {
    id: 7,
    project_id: 1,
    type: 'note',
    label: 'Existing note',
    content: 'Existing content',
    created_at: '2026-05-05T00:00:00Z',
  },
]

describe('AssetSidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('refreshes assets and shows an error when parse fails after upload', async () => {
    const user = userEvent.setup()
    const onReload = vi.fn().mockResolvedValue(undefined)
    vi.mocked(intakeApi.uploadFile).mockResolvedValue({
      id: 8,
      project_id: 1,
      type: 'resume_pdf',
      created_at: '2026-05-05T00:00:00Z',
    })
    vi.mocked(parsingApi.parseProject).mockRejectedValue(new Error('parse failed'))

    render(<AssetSidebar projectId={1} assets={baseAssets} onReload={onReload} />)

    await user.click(screen.getByRole('button', { name: /上传文件/i }))
    await user.click(screen.getByRole('button', { name: 'mock-upload-submit' }))

    await waitFor(() => {
      expect(onReload).toHaveBeenCalledTimes(1)
      expect(screen.getByText('parse failed')).toBeInTheDocument()
    })
  })

  it('refreshes assets and shows an error when note creation fails', async () => {
    const user = userEvent.setup()
    const onReload = vi.fn().mockResolvedValue(undefined)
    vi.mocked(intakeApi.createNote).mockRejectedValue(new Error('create note failed'))

    render(<AssetSidebar projectId={1} assets={baseAssets} onReload={onReload} />)

    await user.click(screen.getByRole('button', { name: /添加备注/i }))
    await user.click(screen.getByRole('button', { name: 'mock-note-submit' }))

    await waitFor(() => {
      expect(onReload).toHaveBeenCalledTimes(1)
      expect(screen.getByText('create note failed')).toBeInTheDocument()
    })
  })

  it('refreshes assets and shows an error when asset update fails', async () => {
    const user = userEvent.setup()
    const onReload = vi.fn().mockResolvedValue(undefined)
    vi.mocked(intakeApi.updateAsset).mockRejectedValue(new Error('update asset failed'))

    render(<AssetSidebar projectId={1} assets={baseAssets} onReload={onReload} />)

    await user.click(screen.getByRole('button', { name: 'mock-edit-asset' }))
    await user.click(screen.getByRole('button', { name: 'mock-edit-submit' }))

    await waitFor(() => {
      expect(onReload).toHaveBeenCalledTimes(1)
      expect(screen.getByText('update asset failed')).toBeInTheDocument()
    })
  })
})
