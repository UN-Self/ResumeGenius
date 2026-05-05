import { render, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import EditorPage from '@/pages/EditorPage'
import { intakeApi, request, workbenchApi } from '@/lib/api-client'

let firstSaveCallback: ((html: string) => Promise<void>) | null = null

vi.mock('@/hooks/useAutoSave', () => ({
  useAutoSave: (options: { save: (html: string) => Promise<void> }) => {
    if (firstSaveCallback === null) {
      firstSaveCallback = options.save
    }
    return {
      scheduleSave: vi.fn(),
      flush: vi.fn(),
      retry: vi.fn(),
      status: 'idle' as const,
      lastSavedAt: null,
      error: null,
    }
  },
}))

vi.mock('@/hooks/useExport', () => ({
  useExport: () => ({
    exportPdf: vi.fn(),
    status: 'idle',
  }),
}))

vi.mock('@tiptap/react', () => ({
  useEditor: () => ({
    commands: { setContent: vi.fn() },
    on: vi.fn(),
    off: vi.fn(),
    getHTML: vi.fn(() => '<p>editor html</p>'),
  }),
}))

vi.mock('@/components/editor/A4Canvas', () => ({
  A4Canvas: () => <div data-testid="a4-canvas" />,
}))

vi.mock('@/components/editor/ActionBar', () => ({
  ActionBar: () => <div>mock-action-bar</div>,
}))

vi.mock('@/components/editor/FormatToolbar', () => ({
  FormatToolbar: () => <div>mock-format-toolbar</div>,
}))

vi.mock('@/components/editor/SaveIndicator', () => ({
  SaveIndicator: () => <div>mock-save-indicator</div>,
}))

vi.mock('@/components/chat/ChatPanel', () => ({
  ChatPanel: () => <div>mock-chat-panel</div>,
}))

vi.mock('@/components/intake/AssetSidebar', () => ({
  default: () => <div>mock-asset-sidebar</div>,
}))

vi.mock('@/components/ui/full-page-state', () => ({
  FullPageState: ({ message }: { message?: string }) => <div>{message ?? 'loading'}</div>,
}))

vi.mock('@/lib/api-client', () => ({
  request: vi.fn(),
  intakeApi: {
    getProject: vi.fn(),
    listAssets: vi.fn(),
  },
  workbenchApi: {
    getDraft: vi.fn(),
  },
  ApiError: class extends Error {
    code: number
    constructor(code: number, message: string) {
      super(message)
      this.code = code
    }
  },
}))

describe('EditorPage auto-save', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    firstSaveCallback = null

    vi.mocked(intakeApi.getProject).mockResolvedValue({
      id: 1,
      title: 'Test Project',
      status: 'active',
      current_draft_id: 1,
      created_at: '2026-05-05T00:00:00Z',
    })
    vi.mocked(intakeApi.listAssets).mockResolvedValue([])
    vi.mocked(workbenchApi.getDraft).mockResolvedValue({
      id: 1,
      project_id: 1,
      html_content: '<p>Hello</p>',
      updated_at: '2026-05-05T00:00:00Z',
    })
    vi.mocked(request).mockResolvedValue(null)
  })

  it('uses the latest draft id even when the original save callback was created before load finished', async () => {
    render(
      <MemoryRouter initialEntries={['/projects/1/edit']}>
        <Routes>
          <Route path="/projects/:projectId/edit" element={<EditorPage />} />
        </Routes>
      </MemoryRouter>
    )

    expect(firstSaveCallback).not.toBeNull()

    await waitFor(() => {
      expect(intakeApi.getProject).toHaveBeenCalledWith(1)
      expect(workbenchApi.getDraft).toHaveBeenCalledWith(1)
    })

    await firstSaveCallback?.('<p>updated</p>')

    expect(request).toHaveBeenCalledWith('/drafts/1', {
      method: 'PUT',
      body: JSON.stringify({ html_content: '<p>updated</p>' }),
    })
  })
})
