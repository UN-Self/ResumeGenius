import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  intakeApi, parsingApi,
  ApiError, type Asset,
} from '@/lib/api-client'
import AssetList from '@/components/intake/AssetList'
import DeleteConfirm from '@/components/intake/DeleteConfirm'
import UploadDialog from '@/components/intake/UploadDialog'
import GitRepoDialog from '@/components/intake/GitRepoDialog'
import NoteDialog from '@/components/intake/NoteDialog'
import { useProjectData } from '@/hooks/useProjectData'
import { Button } from '@/components/ui/button'
import { Alert } from '@/components/ui/alert'
import { FullPageState } from '@/components/ui/full-page-state'

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

  // --- Primary action handler ---
  const handleParse = async () => {
    if (project?.current_draft_id) {
      navigate(`/projects/${pid}/edit`)
      return
    }

    try {
      setParseLoading(true)
      setParseError('')
      await parsingApi.generateProject(pid)
      navigate(`/projects/${pid}/edit`)
    } catch (err) {
      setParseError(err instanceof ApiError ? err.message : '生成初稿失败')
    } finally {
      setParseLoading(false)
    }
  }

  // --- Loading / error states ---
  if (loading) {
    return <FullPageState variant="loading" />
  }

  if (!project) {
    return <FullPageState variant="error" message={error || '项目不存在'} />
  }

  // --- Intake content (rendered inside left panel) ---
  const intakeContent = (
    <div className="max-w-2xl mx-auto px-6 py-10">
      <div className="flex items-center justify-between mb-6">
        <div className="min-w-0 flex-1">
          <button
            onClick={() => navigate('/')}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors mb-1"
          >
            &larr; 返回项目列表
          </button>
          <h1 className="font-serif text-xl font-semibold text-foreground truncate">
            {project.title}
          </h1>
        </div>
        <Button
          variant="danger"
          size="sm"
          className="shrink-0 ml-4"
          onClick={() => setDeleteTarget({ type: 'project', id: pid })}
        >
          删除项目
        </Button>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}
      {parseError && (
        <Alert className="mb-4">生成初稿失败：{parseError}</Alert>
      )}
      {deleteError && (
        <Alert className="mb-4">删除失败：{deleteError}</Alert>
      )}

      <div className="flex gap-2 mb-5">
        <Button
          size="md"
          className="h-9"
          onClick={() => setUploadOpen(true)}
        >
          上传文件
        </Button>
        <Button
          variant="secondary"
          size="md"
          className="h-9"
          onClick={() => setGitOpen(true)}
        >
          接入 Git
        </Button>
        <Button
          variant="secondary"
          size="md"
          className="h-9"
          onClick={() => { setEditingNote(null); setNoteOpen(true) }}
        >
          添加备注
        </Button>
      </div>

      <AssetList
        assets={assets}
        onDelete={(id) => setDeleteTarget({ type: 'asset', id })}
        onEditAsset={handleEditNote}
        canEditAsset={(asset) => asset.type === 'note'}
      />

      {(assets.length > 0 || project.current_draft_id) && (
        <Button
          size="lg"
          className="mt-6 w-full h-11"
          onClick={handleParse}
          disabled={parseLoading}
        >
          {parseLoading
            ? '生成中...'
            : project.current_draft_id
              ? '进入编辑页'
              : '下一步：生成初稿'}
        </Button>
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
    <div className="h-screen bg-background overflow-y-auto">
      {intakeContent}
    </div>
  )
}
