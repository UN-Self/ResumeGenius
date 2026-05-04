import { beforeEach, describe, expect, it, vi } from 'vitest'
import { intakeApi, authApi, workbenchApi, ApiError } from '@/lib/api-client'

describe('apiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('sends credentials on all requests', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: [] }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.listProjects()

    expect(mock).toHaveBeenCalledWith('/api/v1/projects', expect.objectContaining({
      credentials: 'include',
    }))
  })

  it('auth login calls POST /auth/login', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 'u1', username: 'alice' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await authApi.login('alice', 'secret123')
    expect(mock).toHaveBeenCalledWith('/api/v1/auth/login', expect.objectContaining({
      method: 'POST',
    }))
    expect(result.username).toBe('alice')
  })

  it('auth me calls GET /auth/me', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 'u1', username: 'alice' } }),
    })
    vi.stubGlobal('fetch', mock)

    await authApi.me()
    expect(mock).toHaveBeenCalledWith('/api/v1/auth/me', expect.objectContaining({
      credentials: 'include',
    }))
  })

  it('auth logout calls POST /auth/logout', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: null }),
    })
    vi.stubGlobal('fetch', mock)

    await authApi.logout()
    expect(mock).toHaveBeenCalledWith('/api/v1/auth/logout', expect.objectContaining({
      method: 'POST',
    }))
  })

  it('listProjects calls GET /projects', async () => {
    const projects = [{ id: 1, title: 'test' }]
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: projects }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.listProjects()
    expect(mock).toHaveBeenCalledWith('/api/v1/projects', expect.objectContaining({
      credentials: 'include',
    }))
    expect(result).toEqual(projects)
  })

  it('createProject calls POST /projects', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, title: 'test' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.createProject('test')
    expect(mock).toHaveBeenCalledWith('/api/v1/projects', expect.objectContaining({
      method: 'POST',
    }))
    expect(result.title).toBe('test')
  })

  it('getProject calls GET /projects/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, title: 'test' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.getProject(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/projects/1', expect.objectContaining({
      credentials: 'include',
    }))
    expect(result.id).toBe(1)
  })

  it('deleteProject calls DELETE /projects/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: null }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.deleteProject(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/projects/1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('uploadFile sends multipart/form-data', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'resume_pdf' } }),
    })
    vi.stubGlobal('fetch', mock)

    const file = new File(['pdf-content'], 'resume.pdf', { type: 'application/pdf' })
    await intakeApi.uploadFile(1, file)

    expect(mock).toHaveBeenCalledWith('/api/v1/assets/upload', expect.objectContaining({
      method: 'POST',
      credentials: 'include',
    }))
    const body = mock.mock.calls[0][1].body
    expect(body).toBeInstanceOf(FormData)
    expect(body.get('project_id')).toBe('1')
    expect(body.get('file')).toBe(file)
  })

  it('uploadFile includes replace_asset_id when replacing a same-name file', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 2, type: 'resume_docx' } }),
    })
    vi.stubGlobal('fetch', mock)

    const file = new File(['docx-content'], 'sample_resume.docx', {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })
    await intakeApi.uploadFile(3, file, 9)

    const body = mock.mock.calls[0][1].body as FormData
    expect(body.get('project_id')).toBe('3')
    expect(body.get('replace_asset_id')).toBe('9')
    expect(body.get('file')).toBe(file)
  })

  it('createGitRepo calls POST /assets/git', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'git_repo' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.createGitRepo(1, 'https://github.com/user/repo')
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/git', expect.objectContaining({
      method: 'POST',
    }))
    expect(result.type).toBe('git_repo')
  })

  it('listAssets calls GET /assets?project_id=X', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: [] }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.listAssets(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/assets?project_id=1', expect.objectContaining({
      credentials: 'include',
    }))
  })

  it('deleteAsset calls DELETE /assets/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: null }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.deleteAsset(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('createNote calls POST /assets/notes', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'note' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.createNote(1, 'content', 'label')
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/notes', expect.objectContaining({ method: 'POST' }))
    expect(result.type).toBe('note')
  })

  it('updateNote calls PUT /assets/notes/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'note', content: 'updated' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.updateNote(1, 'updated', 'label')
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/notes/1', expect.objectContaining({ method: 'PUT' }))
    expect(result.content).toBe('updated')
  })

  it('throws ApiError on non-zero code', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 1004, message: 'project not found' }),
    })
    vi.stubGlobal('fetch', mock)

    await expect(intakeApi.getProject(999)).rejects.toThrow(ApiError)
    await expect(intakeApi.getProject(999)).rejects.toMatchObject({
      message: 'project not found',
    })
  })

  it('ApiError carries code property', () => {
    const err = new ApiError(1004, 'project not found')
    expect(err.code).toBe(1004)
    expect(err.message).toBe('project not found')
  })

  it('workbenchApi.createDraft calls POST /drafts', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({
        code: 0,
        data: { id: 2, project_id: 1, html_content: '', updated_at: '2026-04-29T12:00:00Z' },
      }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await workbenchApi.createDraft(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/drafts', expect.objectContaining({
      method: 'POST',
    }))
    expect(result.id).toBe(2)
    expect(result.project_id).toBe(1)
    expect(result.html_content).toBe('')
  })

  it('workbenchApi.getDraft calls GET /drafts/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({
        code: 0,
        data: { id: 1, project_id: 1, html_content: '<p>hello</p>', updated_at: '2026-04-29T12:00:00Z' },
      }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await workbenchApi.getDraft(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/drafts/1', expect.objectContaining({
      credentials: 'include',
    }))
    expect(result.html_content).toBe('<p>hello</p>')
  })
})
