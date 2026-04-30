import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  intakeApi, parsingApi, workbenchApi,
  ApiError, type Asset,
} from '@/lib/api-client'
import AssetList from '@/components/intake/AssetList'
import DeleteConfirm from '@/components/intake/DeleteConfirm'
import UploadDialog from '@/components/intake/UploadDialog'
import GitRepoDialog from '@/components/intake/GitRepoDialog'
import NoteDialog from '@/components/intake/NoteDialog'
import { useProjectData } from '@/hooks/useProjectData'

export default function ProjectDetail() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  const { project, assets, loading, error, reload } = useProjectData(pid)

  // UI state
  const [parseLoading, setParseLoading] = useState(false)
  const [parseError, setParseError] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [gitOpen, setGitOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [editingNote, setEditingNote] = useState<Asset | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ type: 'project' | 'asset'; id: number } | null>(null)
  const [deleteError, setDeleteError] = useState('')
  const [deleting, setDeleting] = useState(false)

  // --- Intake handlers ---
  const handleUpload = async (file: File) => {
    await intakeApi.uploadFile(pid, file)
    reload()
  }

  const handleCreateGit = async (repoUrl: string) => {
    await intakeApi.createGitRepo(pid, repoUrl)
    reload()
  }

  const handleCreateNote = async (content: string, label: string) => {
    await intakeApi.createNote(pid, content, label)
    reload()
  }

  const handleUpdateNote = async (content: string, label: string) => {
    if (!editingNote) return
    await intakeApi.updateNote(editingNote.id, content, label)
    setEditingNote(null)
    reload()
  }

  const handleDeleteAsset = async () => {
    if (!deleteTarget || deleteTarget.type !== 'asset') return
    try {
      setDeleting(true)
      await intakeApi.deleteAsset(deleteTarget.id)
      reload()
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  const handleDeleteProject = async () => {
    if (!deleteTarget || deleteTarget.type !== 'project') return
    try {
      setDeleting(true)
      await intakeApi.deleteProject(pid)
      navigate('/')
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : '删除失败')
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  const handleEditNote = (asset: Asset) => {
    setEditingNote(asset)
    setNoteOpen(true)
  }

  // --- Parse handler ---
  const handleParse = async () => {
    try {
      setParseLoading(true)
      setParseError('')
      await parsingApi.parseProject(pid)

      // Ensure a draft exists for editing
      const proj = await intakeApi.getProject(pid)
      if (!proj.current_draft_id) {
        await workbenchApi.createDraft(pid)
      }

      navigate(`/projects/${pid}/edit`)
    } catch (err) {
      setParseError(err instanceof ApiError ? err.message : '解析失败')
    } finally {
      setParseLoading(false)
    }
  }

  // --- Loading / error states ---
  if (loading) {
    return (
      <div className="h-screen bg-[var(--color-page-bg)] flex items-center justify-center">
        <p className="text-[var(--color-text-secondary)] text-sm">加载中...</p>
      </div>
    )
  }

  if (!project) {
    return (
      <div className="h-screen bg-[var(--color-page-bg)] flex items-center justify-center">
        <p className="text-red-500 text-sm">{error || '项目不存在'}</p>
      </div>
    )
  }

  // --- Intake content (rendered inside left panel) ---
  const intakeContent = (
    <div className="max-w-2xl mx-auto px-6 py-10">
      <div className="flex items-center justify-between mb-6">
        <div className="min-w-0 flex-1">
          <button
            onClick={() => navigate('/')}
            className="text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-main)] transition-colors mb-1"
          >
            &larr; 返回项目列表
          </button>
          <h1 className="font-serif text-xl font-semibold text-[var(--color-text-main)] truncate">
            {project.title}
          </h1>
        </div>
        <button
          onClick={() => setDeleteTarget({ type: 'project', id: pid })}
          className="shrink-0 ml-4 text-xs text-[var(--color-text-secondary)] hover:text-red-500 px-3 py-1.5 rounded-lg border border-[var(--color-divider)] hover:border-red-300 transition-colors"
        >
          删除项目
        </button>
      </div>

      {error && (
        <div className="mb-4 px-4 py-2.5 text-sm rounded-lg bg-red-50 text-red-600 border border-red-200">
          {error}
        </div>
      )}
      {parseError && (
        <div className="mb-4 px-4 py-2.5 text-sm rounded-lg bg-red-50 text-red-600 border border-red-200">
          解析失败：{parseError}
        </div>
      )}
      {deleteError && (
        <div className="mb-4 px-4 py-2.5 text-sm rounded-lg bg-red-50 text-red-600 border border-red-200">
          删除失败：{deleteError}
        </div>
      )}

      <div className="flex gap-2 mb-5">
        <button
          onClick={() => setUploadOpen(true)}
          className="h-9 px-4 text-sm font-medium rounded-lg bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)] transition-colors"
        >
          上传文件
        </button>
        <button
          onClick={() => setGitOpen(true)}
          className="h-9 px-4 text-sm font-medium rounded-lg border border-[var(--color-divider)] bg-white text-[var(--color-text-main)] hover:bg-gray-50 transition-colors"
        >
          接入 Git
        </button>
        <button
          onClick={() => { setEditingNote(null); setNoteOpen(true) }}
          className="h-9 px-4 text-sm font-medium rounded-lg border border-[var(--color-divider)] bg-white text-[var(--color-text-main)] hover:bg-gray-50 transition-colors"
        >
          添加备注
        </button>
      </div>

      <AssetList
        assets={assets}
        onDelete={(id) => setDeleteTarget({ type: 'asset', id })}
        onEditNote={handleEditNote}
      />

      {assets.length > 0 && (
        <button
          onClick={handleParse}
          disabled={parseLoading}
          className="mt-6 w-full h-11 text-sm font-medium rounded-lg bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)] transition-colors disabled:opacity-50"
        >
          {parseLoading ? '解析中...' : '下一步：开始解析'}
        </button>
      )}

      <UploadDialog open={uploadOpen} onClose={() => setUploadOpen(false)} onUpload={handleUpload} />
      <GitRepoDialog open={gitOpen} onClose={() => setGitOpen(false)} onSubmit={handleCreateGit} />
      <NoteDialog
        open={noteOpen}
        onClose={() => { setNoteOpen(false); setEditingNote(null) }}
        onSubmit={editingNote ? handleUpdateNote : handleCreateNote}
        initialNote={editingNote ?? undefined}
      />
      <DeleteConfirm
        open={deleteTarget !== null}
        title={deleteTarget?.type === 'project' ? '删除项目' : '删除资料'}
        message={deleteTarget?.type === 'project'
          ? '删除后该项目下的所有资料和文件将被永久删除，此操作不可撤销。'
          : '删除后该资料将被永久删除，此操作不可撤销。'}
        onConfirm={deleteTarget?.type === 'project' ? handleDeleteProject : handleDeleteAsset}
        onCancel={() => setDeleteTarget(null)}
        loading={deleting}
      />
    </div>
  )

  return (
    <div className="h-screen bg-[var(--color-page-bg)] overflow-y-auto">
      {intakeContent}
    </div>
  )
}
