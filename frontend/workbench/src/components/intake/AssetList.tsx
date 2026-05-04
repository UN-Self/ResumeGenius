import { PencilLine, RefreshCcw, Trash2 } from 'lucide-react'
import type { Asset } from '@/lib/api-client'
import { getAssetVisual, getDisplayAssetTitle, getDisplayFileName } from './fileVisuals'

type AssetItem = Asset & { label?: string; content?: string; uri?: string }

interface AssetListProps {
  assets: AssetItem[]
  onDelete: (id: number) => void
  onEditAsset?: (asset: AssetItem) => void
  canEditAsset?: (asset: AssetItem) => boolean
  onReparseAsset?: (asset: AssetItem) => void
  canReparseAsset?: (asset: AssetItem) => boolean
  reparseLoadingAssetId?: number | null
}

interface AssetActionButtonProps {
  label: string
  onClick: () => void
  icon: typeof RefreshCcw
  danger?: boolean
  disabled?: boolean
}

function AssetActionButton({
  label,
  onClick,
  icon: Icon,
  danger = false,
  disabled = false,
}: AssetActionButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={[
        'inline-flex h-7 items-center gap-1 rounded-full border px-2.5 text-[11px] font-medium shadow-sm transition-colors',
        'disabled:pointer-events-none disabled:opacity-50',
        danger
          ? 'border-red-200 bg-white text-muted-foreground hover:bg-red-50 hover:text-red-600'
          : 'border-border bg-white text-muted-foreground hover:border-primary-200 hover:bg-primary-50 hover:text-foreground',
      ].join(' ')}
    >
      <Icon className="h-3.5 w-3.5" />
      <span>{label}</span>
    </button>
  )
}

function getDisplayTitle(asset: AssetItem, fallbackLabel: string) {
  const originalFilename = getOriginalFilenameFromMetadata(asset)
  if (isFileAsset(asset.type) && originalFilename) {
    return getDisplayFileName(originalFilename) || originalFilename
  }

  if (asset.label?.trim() && !isGenericAssetLabel(asset, asset.label.trim())) {
    return getDisplayAssetTitle(asset.type, asset.label) || asset.label.trim()
  }

  if (asset.uri && asset.type.startsWith('resume_')) {
    const fileName = asset.uri.split('/').pop() ?? asset.uri
    return getDisplayFileName(fileName) || fallbackLabel
  }

  if (asset.uri && asset.type === 'git_repo') {
    const normalized = asset.uri.replace(/\/+$/, '')
    return normalized.split('/').pop()?.replace(/\.git$/i, '') || normalized
  }

  if (asset.type === 'note') {
    return '未命名备注'
  }

  return fallbackLabel
}

function isFileAsset(assetType: string) {
  return assetType === 'resume_pdf' || assetType === 'resume_docx' || assetType === 'resume_image'
}

function isGenericAssetLabel(asset: AssetItem, label: string) {
  const visual = getAssetVisual(asset.type, asset.uri)
  const normalized = label.trim().toLowerCase()
  return normalized === visual.chipLabel.toLowerCase() || normalized === visual.typeLabel.toLowerCase()
}

function getOriginalFilenameFromMetadata(asset: AssetItem) {
  if (!asset.metadata || typeof asset.metadata !== 'object') {
    return ''
  }

  const parsing = (asset.metadata as Record<string, unknown>).parsing
  if (!parsing || typeof parsing !== 'object') {
    return ''
  }

  const originalFilename = (parsing as Record<string, unknown>).original_filename
  return typeof originalFilename === 'string' ? originalFilename.trim() : ''
}

function getContentPreview(asset: AssetItem) {
  const raw = asset.content?.replace(/\r\n/g, '\n').replace(/\r/g, '\n').trim() ?? ''
  if (!raw) {
    return ''
  }

  if (asset.type === 'note' && asset.label?.trim() && raw.startsWith(asset.label.trim())) {
    return raw.slice(asset.label.trim().length).replace(/^\s+/, '').replace(/\n{3,}/g, '\n\n')
  }

  return raw.replace(/\n{3,}/g, '\n\n')
}

export default function AssetList({
  assets,
  onDelete,
  onEditAsset,
  canEditAsset,
  onReparseAsset,
  canReparseAsset,
  reparseLoadingAssetId,
}: AssetListProps) {
  if (assets.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        <p className="text-sm">还没有添加任何资料</p>
        <p className="mt-1 text-xs">点击上方按钮上传文件、接入 Git 或添加备注</p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {assets.map((asset) => {
        const visual = getAssetVisual(asset.type, asset.uri)
        const Icon = visual.icon
        const title = getDisplayTitle(asset, visual.chipLabel)
        const contentPreview = getContentPreview(asset)
        const editable = onEditAsset !== undefined && (canEditAsset ? canEditAsset(asset) : asset.type === 'note')
        const reparsable = onReparseAsset !== undefined && (canReparseAsset ? canReparseAsset(asset) : false)
        const showGitSource = asset.type === 'git_repo' && asset.uri

        return (
          <div
            key={asset.id}
            className="rounded-2xl border border-border bg-card/90 p-3.5 shadow-sm transition-colors hover:border-primary-200"
          >
            <div className="flex items-start gap-3">
              <div
                className={`mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border ${visual.iconWrapperClassName}`}
              >
                <Icon className={`h-5 w-5 ${visual.iconClassName}`} />
              </div>

              <div className="min-w-0 flex-1">
                <div className="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
                  <div className="min-w-0 flex-1">
                    <div className="flex min-h-8 flex-wrap items-center gap-2">
                      <p className="min-w-0 break-all text-sm font-semibold text-foreground">{title}</p>
                      <span
                        className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold tracking-wide ${visual.chipClassName}`}
                      >
                        {visual.chipLabel}
                      </span>
                    </div>

                    {showGitSource && (
                      <p className="mt-1 text-xs text-muted-foreground break-all">{asset.uri}</p>
                    )}
                  </div>

                  <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                    {reparsable && (
                      <AssetActionButton
                        label={reparseLoadingAssetId === asset.id ? '解析中...' : '重新解析'}
                        onClick={() => onReparseAsset?.(asset)}
                        icon={RefreshCcw}
                        disabled={reparseLoadingAssetId === asset.id}
                      />
                    )}

                    {editable && (
                      <AssetActionButton
                        label="编辑"
                        onClick={() => onEditAsset?.(asset)}
                        icon={PencilLine}
                      />
                    )}

                    <AssetActionButton
                      label="删除"
                      onClick={() => onDelete(asset.id)}
                      icon={Trash2}
                      danger
                    />
                  </div>
                </div>

                {contentPreview ? (
                  <div className="mt-3 max-h-36 overflow-y-auto rounded-xl border border-border/80 bg-background px-3 py-2.5 text-[13px] leading-relaxed text-muted-foreground whitespace-pre-wrap break-words">
                    {contentPreview}
                  </div>
                ) : (
                  <div className="mt-3 rounded-xl border border-dashed border-border/80 bg-background/60 px-3 py-2.5 text-xs text-muted-foreground">
                    该素材当前没有可展示的正文内容。
                  </div>
                )}
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
