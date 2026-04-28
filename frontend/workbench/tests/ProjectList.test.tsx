import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { intakeApi } from '@/lib/api-client'
import ProjectList from '@/pages/ProjectList'

vi.mock('@/lib/api-client', () => ({
  intakeApi: {
    listProjects: vi.fn(),
    createProject: vi.fn(),
  },
  ApiError: class extends Error {
    code: number
    constructor(c: number, m: string) {
      super(m)
      this.code = c
    }
  },
}))

function renderWithRouter(ui: React.ReactNode) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('ProjectList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading state', () => {
    vi.mocked(intakeApi.listProjects).mockImplementation(() => new Promise(() => {}))
    renderWithRouter(<ProjectList />)
    expect(screen.getByText('加载中...')).toBeInTheDocument()
  })

  it('renders empty state when no projects', async () => {
    vi.mocked(intakeApi.listProjects).mockResolvedValue([])
    renderWithRouter(<ProjectList />)
    await waitFor(() => {
      expect(screen.getByText('还没有项目')).toBeInTheDocument()
    })
  })

  it('renders project list', async () => {
    vi.mocked(intakeApi.listProjects).mockResolvedValue([
      { id: 1, title: '前端工程师简历', status: 'active', current_draft_id: null, created_at: '2026-04-28T00:00:00Z' },
      { id: 2, title: '产品经理简历', status: 'active', current_draft_id: null, created_at: '2026-04-27T00:00:00Z' },
    ])
    renderWithRouter(<ProjectList />)
    await waitFor(() => {
      expect(screen.getByText('前端工程师简历')).toBeInTheDocument()
      expect(screen.getByText('产品经理简历')).toBeInTheDocument()
    })
  })

  it('creates project on Enter key', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.listProjects).mockResolvedValue([])
    vi.mocked(intakeApi.createProject).mockResolvedValue({
      id: 1, title: '新项目', status: 'active', current_draft_id: null, created_at: '2026-04-28T00:00:00Z',
    })

    renderWithRouter(<ProjectList />)
    await waitFor(() => {
      expect(screen.getByPlaceholderText('输入项目名称，按 Enter 创建')).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText('输入项目名称，按 Enter 创建')
    await user.type(input, '新项目{Enter}')

    await waitFor(() => {
      expect(intakeApi.createProject).toHaveBeenCalledWith('新项目')
    })
  })

  it('creates project on button click', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.listProjects).mockResolvedValue([])
    vi.mocked(intakeApi.createProject).mockResolvedValue({
      id: 1, title: '新项目', status: 'active', current_draft_id: null, created_at: '2026-04-28T00:00:00Z',
    })

    renderWithRouter(<ProjectList />)
    await waitFor(() => {
      expect(screen.getByPlaceholderText('输入项目名称，按 Enter 创建')).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText('输入项目名称，按 Enter 创建')
    await user.type(input, '新项目')

    const btn = screen.getByRole('button', { name: '创建' })
    await user.click(btn)

    await waitFor(() => {
      expect(intakeApi.createProject).toHaveBeenCalledWith('新项目')
    })
  })

  it('shows error on create failure', async () => {
    const user = userEvent.setup()
    vi.mocked(intakeApi.listProjects).mockResolvedValue([])
    vi.mocked(intakeApi.createProject).mockRejectedValue(new Error('创建失败'))

    renderWithRouter(<ProjectList />)
    await waitFor(() => {
      expect(screen.getByPlaceholderText('输入项目名称，按 Enter 创建')).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText('输入项目名称，按 Enter 创建')
    await user.type(input, '新项目{Enter}')

    await waitFor(() => {
      expect(screen.getByText('创建失败')).toBeInTheDocument()
    })
  })
})
