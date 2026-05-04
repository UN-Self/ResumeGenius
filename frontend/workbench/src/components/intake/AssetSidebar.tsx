import { useMemo, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { intakeApi, parsingApi, type Asset } from '@/lib/api-client'
import AssetList from './AssetList'
import AssetEditorDialog from './AssetEditorDialog'
import DeleteConfirm from './DeleteConfirm'
import GitRepoDialog from './GitRepoDialog'
import NoteDialog from './NoteDialog'
import UploadDialog from './UploadDialog'

interface AssetSidebarProps {
  projectId: number
  assets: Asset[]
  onReload: () => Promise<void>
}

interface ParsingMetadata {
  updated_by_user?: boolean
  derived?: boolean
  last_parsed_at?: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function getParsingMetadata(asset: Asset): ParsingMetadata | null {
  if (!isRecord(asset.metadata)) {
    return null
  }
  const parsing = asset.metadata.parsing
  if (!isRecord(parsing)) {
    return null
  }
  return parsing as ParsingMetadata
}

function isDerivedImageAsset(asset: Asset) {
  return asset.type === 'resume_image' && getParsingMetadata(asset)?.derived === true
}

function canEditAsset(asset: Asset) {
  return asset.type !== 'resume_image'
}

function canReparseAsset(asset: Asset) {
  return asset.type === 'resume_pdf' || asset.type === 'resume_docx' || asset.type === 'git_repo'
}

function hasUserEditedContent(asset: Asset) {
  return getParsingMetadata(asset)?.updated_by_user === true
}

function formatParsedAt(value?: string) {
  if (!value) return ''

  const parsedDate = new Date(value)
  if (Number.isNaN(parsedDate.getTime())) {
    return ''
  }

  return parsedDate.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

export default function AssetSidebar({ projectId, assets, onReload }: AssetSidebarProps) {
  const [error, setError] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [gitOpen, setGitOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [editingAsset, setEditingAsset] = useState<Asset | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Asset | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [reparseLoadingAssetId, setReparseLoadingAssetId] = useState<number | null>(null)

  const visibleAssets = useMemo(
    () => assets.filter((asset) => !isDerivedImageAsset(asset)),
    [assets]
  )

  const reparsableAssets = useMemo(
    () => visibleAssets.filter(canReparseAsset),
    [visibleAssets]
  )

  const dirtyReparseAssets = useMemo(
    () => reparsableAssets.filter(hasUserEditedContent),
    [reparsableAssets]
  )

  const refreshAssets = async () => {
    await onReload()
  }

  const handleUpload = async (file: File) => {
    setError('')
    try {
      await intakeApi.uploadFile(projectId, file)
      await parsingApi.parseProject(projectId)
    } finally {
      await refreshAssets()
    }
  }

  const handleCreateGit = async (repoUrl: string) => {
    setError('')
    try {
      await intakeApi.createGitRepo(projectId, repoUrl)
      await parsingApi.parseProject(projectId)
    } finally {
      await refreshAssets()
    }
  }

  const handleCreateNote = async (content: string, label: string) => {
    setError('')
    await intakeApi.createNote(projectId, content, label)
    await refreshAssets()
  }

  const handleUpdateAsset = async (content: string, label: string) => {
    if (!editingAsset) return
    setError('')
    await intakeApi.updateAsset(editingAsset.id, { content, label })
    setEditingAsset(null)
    await refreshAssets()
  }

  const handleDeleteAsset = async () => {
    if (!deleteTarget) return
    try {
      setDeleting(true)
      setError('')
      await intakeApi.deleteAsset(deleteTarget.id)
      await refreshAssets()
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '删除失败')
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  const handleReparseAsset = async (asset: Asset) => {
    const dirtyCount = dirtyReparseAssets.length
    if (dirtyCount > 0) {
      const confirmed = window.confirm(
        dirtyCount === 1
          ? '重新解析会覆盖当前已手动修改的素材正文，是否继续？'
          : `重新解析会刷新项目中所有可解析素材，并覆盖 ${dirtyCount} 项已手动修改的正文，是否继续？`
      )
      if (!confirmed) {
        return
      }
    }

    try {
      setReparseLoadingAssetId(asset.id)
      setError('')
      await parsingApi.parseProject(projectId)
      await refreshAssets()
    } catch (reparseError) {
      setError(reparseError instanceof Error ? reparseError.message : '重新解析失败')
    } finally {
      setReparseLoadingAssetId(null)
    }
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="flex flex-wrap gap-2">
        <Button size="sm" onClick={() => setUploadOpen(true)}>
          上传文件
        </Button>
        <Button size="sm" variant="secondary" onClick={() => setGitOpen(true)}>
          接入 Git
        </Button>
        <Button size="sm" variant="secondary" onClick={() => setNoteOpen(true)}>
          添加备注
        </Button>
      </div>

      {dirtyReparseAssets.length > 0 && (
        <Alert className="mt-3">
          {dirtyReparseAssets.length === 1
            ? '有 1 项素材已手动修改，重新解析会覆盖当前正文。'
            : `有 ${dirtyReparseAssets.length} 项素材已手动修改，重新解析会覆盖这些正文。`}
        </Alert>
      )}

      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}

      <div className="mt-4">
        <AssetList
          assets={visibleAssets}
          onDelete={(id) => {
            const target = visibleAssets.find((asset) => asset.id === id) ?? null
            setDeleteTarget(target)
          }}
          onEditAsset={(asset) => setEditingAsset(asset)}
          canEditAsset={canEditAsset}
          onReparseAsset={handleReparseAsset}
          canReparseAsset={canReparseAsset}
          reparseLoadingAssetId={reparseLoadingAssetId}
          getAssetStatusMeta={(asset) => {
            if (hasUserEditedContent(asset)) {
              return {
                text: '已手动修改，重新解析将覆盖当前正文',
                tone: 'warning',
              }
            }

            const formatted = formatParsedAt(getParsingMetadata(asset)?.last_parsed_at)
            if (formatted) {
              return {
                text: `最近解析：${formatted}`,
                tone: 'muted',
              }
            }

            return null
          }}
        />
      </div>

      <UploadDialog
        open={uploadOpen}
        onClose={() => setUploadOpen(false)}
        onUpload={handleUpload}
      />
      <GitRepoDialog
        open={gitOpen}
        onClose={() => setGitOpen(false)}
        onSubmit={handleCreateGit}
      />
      <NoteDialog
        open={noteOpen}
        onClose={() => setNoteOpen(false)}
        onSubmit={handleCreateNote}
      />
      <AssetEditorDialog
        open={editingAsset !== null}
        asset={editingAsset}
        onClose={() => setEditingAsset(null)}
        onSubmit={handleUpdateAsset}
      />
      <DeleteConfirm
        open={deleteTarget !== null}
        title="删除素材"
        message="删除后该素材将被永久删除，此操作不可撤销。"
        onConfirm={handleDeleteAsset}
        onCancel={() => setDeleteTarget(null)}
        loading={deleting}
      />
    </div>
  )
}
