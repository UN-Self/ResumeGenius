import { useEffect, useMemo, useRef, useState } from 'react'
import { FolderUp, FolderPlus } from 'lucide-react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Modal, ModalBody, ModalFooter, ModalHeader } from '@/components/ui/modal'
import { intakeApi, parsingApi, type Asset } from '@/lib/api-client'
import AssetList from './AssetList'
import DeleteConfirm from './DeleteConfirm'
import GitRepoDialog from './GitRepoDialog'
import NoteDialog from './NoteDialog'
import UploadDialog from './UploadDialog'
import { getExt } from './fileVisuals'

const MAX_FOLDER_DEPTH = 7
const FOLDER_UPLOAD_ALLOWED_EXTENSIONS = new Set(['.pdf', '.docx', '.png', '.jpg', '.jpeg'])

interface AssetSidebarProps {
  projectId: number
  assets: Asset[]
  onReload: () => Promise<void>
  onOpenAsset?: (asset: Asset) => void
  selectedAssetId?: number | null
}

interface ParsingMetadata {
  updated_by_user?: boolean
  derived?: boolean
  last_parsed_at?: string
  source_deleted?: boolean
  original_filename?: string
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

function canReparseAsset(asset: Asset) {
  if (asset.type === 'git_repo') {
    return true
  }

  if (asset.type === 'resume_pdf' || asset.type === 'resume_docx') {
    return getParsingMetadata(asset)?.source_deleted !== true
  }

  return false
}

function hasUserEditedContent(asset: Asset) {
  return getParsingMetadata(asset)?.updated_by_user === true
}

function getAssetFolderId(asset: Asset) {
  const raw = asset.metadata?.folder_id
  return typeof raw === 'number' ? raw : null
}

function getFolderDepth(folderId: number | null, folders: Asset[]) {
  if (folderId === null) {
    return 0
  }

  const foldersById = new Map(folders.map((folder) => [folder.id, folder]))
  const visited = new Set<number>()
  let currentId: number | null = folderId
  let depth = 0

  while (currentId !== null) {
    if (visited.has(currentId)) {
      return depth
    }
    visited.add(currentId)

    const folder = foldersById.get(currentId)
    if (!folder) {
      return depth
    }

    depth += 1
    currentId = getAssetFolderId(folder)
  }

  return depth
}

function getDirectoryFilePath(file: File) {
  return (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name
}

function sanitizeFolderName(value: string) {
  return value.replace(/[\\/:*?"<>|]+/g, ' ').replace(/\s+/g, ' ').trim()
}

export default function AssetSidebar({
  projectId,
  assets,
  onReload,
  onOpenAsset,
  selectedAssetId,
}: AssetSidebarProps) {
  const [error, setError] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [gitOpen, setGitOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [folderOpen, setFolderOpen] = useState(false)
  const [folderName, setFolderName] = useState('')
  const [selectedFolderId, setSelectedFolderId] = useState<number | null>(null)
  const [creatingFolder, setCreatingFolder] = useState(false)
  const [uploadingFolder, setUploadingFolder] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Asset | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [reparseLoadingAssetId, setReparseLoadingAssetId] = useState<number | null>(null)
  const folderUploadInputRef = useRef<HTMLInputElement>(null)

  const visibleAssets = useMemo(
    () =>
      [...assets]
        .filter((asset) => !isDerivedImageAsset(asset))
        .sort((left, right) => {
          const leftTime = new Date(left.created_at).getTime()
          const rightTime = new Date(right.created_at).getTime()
          return rightTime - leftTime
        }),
    [assets]
  )

  const folders = useMemo(
    () => visibleAssets.filter((asset) => asset.type === 'folder'),
    [visibleAssets]
  )

  useEffect(() => {
    if (selectedFolderId === null) return
    if (folders.some((folder) => folder.id === selectedFolderId)) return
    setSelectedFolderId(null)
  }, [folders, selectedFolderId])

  const reparsableAssets = useMemo(
    () => visibleAssets.filter(canReparseAsset),
    [visibleAssets]
  )

  const dirtyReparseAssets = useMemo(
    () => reparsableAssets.filter(hasUserEditedContent),
    [reparsableAssets]
  )

  const selectedFolderDepth = useMemo(
    () => getFolderDepth(selectedFolderId, folders),
    [folders, selectedFolderId]
  )
  const folderDepthLimitReached = selectedFolderDepth >= MAX_FOLDER_DEPTH

  const refreshAssets = async () => {
    await onReload()
  }

  const handleUpload = async (file: File, replaceAssetId?: number, folderId?: number | null) => {
    setError('')
    try {
      await intakeApi.uploadFile(projectId, file, replaceAssetId, folderId)
      await parsingApi.parseProject(projectId)
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : '上传或解析失败')
    } finally {
      await refreshAssets()
    }
  }

  const handleCreateGit = async (repoUrl: string) => {
    setError('')
    try {
      await intakeApi.createGitRepo(projectId, repoUrl)
      await parsingApi.parseProject(projectId)
    } catch (createGitError) {
      setError(createGitError instanceof Error ? createGitError.message : '创建 Git 素材失败')
    } finally {
      await refreshAssets()
    }
  }

  const handleCreateNote = async (content: string, label: string) => {
    setError('')
    try {
      await intakeApi.createNote(projectId, content, label)
    } catch (createNoteError) {
      setError(createNoteError instanceof Error ? createNoteError.message : '创建备注失败')
    } finally {
      await refreshAssets()
    }
  }

  const handleRenameAsset = async (asset: Asset, label: string) => {
    setError('')
    try {
      await intakeApi.updateAsset(asset.id, { label })
    } catch (renameError) {
      setError(renameError instanceof Error ? renameError.message : '重命名素材失败')
    } finally {
      await refreshAssets()
    }
  }

  const handleCreateFolder = async () => {
    const trimmedName = folderName.replace(/\s+/g, ' ').trim()
    if (!trimmedName) return
    if (folderDepthLimitReached) {
      setError(`文件夹最多支持 ${MAX_FOLDER_DEPTH} 层`)
      return
    }

    setCreatingFolder(true)
    setError('')
    try {
      const folder = await intakeApi.createFolder(projectId, trimmedName, selectedFolderId)
      setSelectedFolderId(folder.id)
      setFolderName('')
      setFolderOpen(false)
    } catch (createFolderError) {
      setError(createFolderError instanceof Error ? createFolderError.message : '创建文件夹失败')
    } finally {
      setCreatingFolder(false)
      await refreshAssets()
    }
  }

  const handleUploadFolderInput = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFiles = Array.from(event.target.files ?? [])
    event.currentTarget.value = ''
    if (selectedFiles.length === 0 || folderDepthLimitReached) return

    const supportedFiles = selectedFiles.filter((file) => FOLDER_UPLOAD_ALLOWED_EXTENSIONS.has(getExt(file.name)))
    const ignoredUnsupportedCount = selectedFiles.length - supportedFiles.length
    if (supportedFiles.length === 0) {
      setError('该文件夹里没有可上传的 PDF、DOCX、PNG 或 JPG 文件')
      return
    }

    const firstPath = getDirectoryFilePath(supportedFiles[0]).split(/[\\/]+/).filter(Boolean)
    const rootFolderName = sanitizeFolderName(firstPath.length > 1 ? firstPath[0] : '上传文件夹')
    if (!rootFolderName) {
      setError('无法识别文件夹名称')
      return
    }

    setUploadingFolder(true)
    setError('')
    let skippedDepthCount = 0
    let failedCount = 0
    let uploadedCount = 0

    try {
      const rootFolder = await intakeApi.createFolder(projectId, rootFolderName, selectedFolderId)
      const folderCache = new Map<string, number>([['', rootFolder.id]])
      const maxChildFolderDepth = MAX_FOLDER_DEPTH - (selectedFolderDepth + 1)

      for (const file of supportedFiles) {
        const segments = getDirectoryFilePath(file).split(/[\\/]+/).filter(Boolean)
        const relativeSegments = segments.length > 1 ? segments.slice(1) : segments
        const folderSegments = relativeSegments.slice(0, -1).map(sanitizeFolderName).filter(Boolean)

        if (folderSegments.length > maxChildFolderDepth) {
          skippedDepthCount += 1
          continue
        }

        let parentFolderId = rootFolder.id
        let folderPath = ''
        try {
          for (const folderName of folderSegments) {
            folderPath = folderPath ? `${folderPath}/${folderName}` : folderName
            const cachedFolderId = folderCache.get(folderPath)
            if (cachedFolderId !== undefined) {
              parentFolderId = cachedFolderId
              continue
            }

            const folder = await intakeApi.createFolder(projectId, folderName, parentFolderId)
            folderCache.set(folderPath, folder.id)
            parentFolderId = folder.id
          }

          await intakeApi.uploadFile(projectId, file, undefined, parentFolderId)
          uploadedCount += 1
        } catch {
          failedCount += 1
        }
      }

      if (uploadedCount > 0) {
        await parsingApi.parseProject(projectId)
      }

      const notices = []
      if (ignoredUnsupportedCount > 0) notices.push(`已忽略 ${ignoredUnsupportedCount} 个不支持的文件`)
      if (skippedDepthCount > 0) notices.push(`已忽略 ${skippedDepthCount} 个超过 ${MAX_FOLDER_DEPTH} 层的文件`)
      if (failedCount > 0) notices.push(`${failedCount} 个文件上传失败`)
      setSelectedFolderId(rootFolder.id)
      setError(notices.join('，'))
    } catch (uploadFolderError) {
      setError(uploadFolderError instanceof Error ? uploadFolderError.message : '上传文件夹失败')
    } finally {
      setUploadingFolder(false)
      await refreshAssets()
    }
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
        <Button
          size="sm"
          variant="secondary"
          onClick={() => folderUploadInputRef.current?.click()}
          disabled={uploadingFolder || folderDepthLimitReached}
        >
          <FolderUp className="h-3.5 w-3.5" />
          {uploadingFolder ? '上传中...' : '上传文件夹'}
        </Button>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => setFolderOpen(true)}
          disabled={creatingFolder || folderDepthLimitReached}
        >
          <FolderPlus className="h-3.5 w-3.5" />
          {creatingFolder ? '创建中...' : '新建文件夹'}
        </Button>
        <input
          ref={folderUploadInputRef}
          type="file"
          className="hidden"
          multiple
          onChange={handleUploadFolderInput}
          {...{ webkitdirectory: '', directory: '' }}
        />
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
          onOpenAsset={onOpenAsset}
          selectedAssetId={selectedAssetId}
          onSelectFolder={setSelectedFolderId}
          selectedFolderId={selectedFolderId}
          onDelete={(id) => {
            const target = visibleAssets.find((asset) => asset.id === id) ?? null
            setDeleteTarget(target)
          }}
          onRenameAsset={handleRenameAsset}
          onReparseAsset={handleReparseAsset}
          canReparseAsset={canReparseAsset}
          reparseLoadingAssetId={reparseLoadingAssetId}
        />
      </div>

      <UploadDialog
        open={uploadOpen}
        onClose={() => setUploadOpen(false)}
        onUpload={handleUpload}
        existingAssets={visibleAssets}
        folders={folders}
        defaultFolderId={selectedFolderId}
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
      <Modal open={folderOpen} onClose={() => setFolderOpen(false)} maxWidth="max-w-sm">
        <ModalHeader>新建文件夹</ModalHeader>
        <ModalBody>
          <p className="mb-3 text-sm text-muted-foreground">
            {folderDepthLimitReached
              ? `文件夹最多支持 ${MAX_FOLDER_DEPTH} 层，请先选择更上层目录。`
              : selectedFolderId === null
                ? '将在根目录下创建文件夹。'
                : '将在当前选中文件夹下创建子文件夹。'}
          </p>
          <Input
            value={folderName}
            onChange={(event) => setFolderName(event.target.value.replace(/[\r\n]+/g, ' '))}
            placeholder="例如：原始简历 / 作品集 / 证书"
            autoFocus
          />
        </ModalBody>
        <ModalFooter>
          <Button variant="secondary" onClick={() => setFolderOpen(false)}>
            取消
          </Button>
          <Button
            onClick={() => void handleCreateFolder()}
            disabled={!folderName.trim() || creatingFolder || folderDepthLimitReached}
          >
            {creatingFolder ? '创建中...' : '创建'}
          </Button>
        </ModalFooter>
      </Modal>
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
