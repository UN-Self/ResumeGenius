import { ImageOff, Loader2, Save, Eye, Pencil, AlertTriangle } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { Button } from '@/components/ui/button'
import type { Asset } from '@/lib/api-client'
import { getAssetVisual } from './fileVisuals'
import { getDisplayTitle } from './AssetList'

interface AssetWorkspaceProps {
  asset: Asset
  value: string
  dirty: boolean
  saving: boolean
  onChange: (value: string) => void
  onSave: () => void
}

function normalizeText(value?: string) {
  return value?.replace(/\r\n/g, '\n').replace(/\r/g, '\n') ?? ''
}

function lineNumbersFor(value: string) {
  const lineCount = Math.max(1, normalizeText(value).split('\n').length)
  return Array.from({ length: lineCount }, (_, index) => index + 1).join('\n')
}

function isImageAsset(asset: Asset) {
  return asset.type === 'resume_image'
}

function getParseStatus(asset: Asset): string | null {
  const parsing = asset.metadata?.parsing as Record<string, unknown> | undefined
  const s = parsing?.status
  if (s === 'parsing' || s === 'success' || s === 'failed') return s
  return null
}

export function AssetWorkspace({
  asset,
  value,
  dirty,
  saving,
  onChange,
  onSave,
}: AssetWorkspaceProps) {
  const lineNumberRef = useRef<HTMLPreElement>(null)
  const [imageError, setImageError] = useState(false)
  const visual = getAssetVisual(asset.type, asset.uri)
  const title = getDisplayTitle(asset, visual.chipLabel)

  const isGitRepo = asset.type === 'git_repo'
  const parseStatus = getParseStatus(asset)
  const hasContent = !!value
  const isParsing = isGitRepo && (parseStatus === 'parsing' || (!hasContent && parseStatus !== 'failed'))
  const isFailed = isGitRepo && parseStatus === 'failed'
  const isReady = isGitRepo && parseStatus === 'success' && hasContent

  const [viewMode, setViewMode] = useState<'edit' | 'preview'>(() =>
    isReady ? 'preview' : 'edit'
  )

  useEffect(() => {
    setImageError(false)
  }, [asset.id])

  useEffect(() => {
    setViewMode(isReady ? 'preview' : 'edit')
  }, [asset.id, isReady])

  if (isImageAsset(asset)) {
    return (
      <div className="asset-workspace">
        <div className="asset-workspace-header">
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-foreground">{title}</p>
            <p className="mt-1 text-xs text-muted-foreground">图片预览</p>
          </div>
        </div>
        <div className="asset-image-stage">
          {imageError ? (
            <div className="asset-image-error">
              <ImageOff className="h-8 w-8" />
              <p className="mt-3 text-sm font-semibold text-foreground">图片文件暂时不可预览</p>
              <p className="mt-2 max-w-sm text-center text-xs leading-5 text-muted-foreground">
                原始图片文件不在当前后端存储目录中。重新上传一次后，后续会保存在持久化目录里。
              </p>
            </div>
          ) : (
            <img
              key={asset.id}
              src={`/api/v1/assets/${asset.id}/file`}
              alt={title}
              className="asset-image-preview"
              onError={() => setImageError(true)}
            />
          )}
        </div>
      </div>
    )
  }

  // Git repo: parsing in progress — show spinner, content will auto-appear on next poll refresh
  if (isParsing) {
    return (
      <div className="asset-workspace">
        <div className="asset-workspace-header">
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-foreground">{title}</p>
            <p className="mt-1 text-xs text-muted-foreground">仓库分析</p>
          </div>
        </div>
        <div className="flex-1 flex items-center justify-center min-h-0">
          <div className="flex flex-col items-center gap-3 text-muted-foreground">
            <Loader2 className="h-8 w-8 animate-spin text-amber-500" />
            <p className="text-sm font-medium">正在解析仓库...</p>
            <p className="text-xs">AI 正在分析代码结构与架构，完成后自动展示</p>
          </div>
        </div>
      </div>
    )
  }

  // Git repo: parse failed — show error state
  if (isFailed) {
    return (
      <div className="asset-workspace">
        <div className="asset-workspace-header">
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-foreground">{title}</p>
            <p className="mt-1 text-xs text-muted-foreground">解析失败</p>
          </div>
        </div>
        <div className="flex-1 flex items-center justify-center min-h-0">
          <div className="flex flex-col items-center gap-3 text-muted-foreground max-w-sm text-center">
            <AlertTriangle className="h-8 w-8 text-red-400" />
            <p className="text-sm font-medium">仓库解析失败</p>
            <p className="text-xs">可能是网络问题或仓库无法访问，请在侧栏点击"重新解析"重试</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="asset-workspace">
      <div className="asset-workspace-header">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-foreground">{title}</p>
          <p className="mt-1 text-xs text-muted-foreground">
            {viewMode === 'preview' ? '分析报告' : '解析文本'}
          </p>
        </div>
        <div className="flex items-center gap-1">
          {isReady && (
            <div className="flex rounded-md border border-input overflow-hidden mr-1">
              <button
                type="button"
                onClick={() => setViewMode('preview')}
                className={`flex items-center gap-1 px-2 py-1 text-xs ${
                  viewMode === 'preview'
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-background text-muted-foreground hover:bg-muted'
                }`}
              >
                <Eye className="h-3 w-3" />
                预览
              </button>
              <button
                type="button"
                onClick={() => setViewMode('edit')}
                className={`flex items-center gap-1 px-2 py-1 text-xs ${
                  viewMode === 'edit'
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-background text-muted-foreground hover:bg-muted'
                }`}
              >
                <Pencil className="h-3 w-3" />
                编辑
              </button>
            </div>
          )}
          <Button
            size="sm"
            type="button"
            disabled={!dirty || saving}
            onClick={onSave}
          >
            <Save className="h-3.5 w-3.5" />
            {saving ? '保存中...' : '保存'}
          </Button>
        </div>
      </div>

      {viewMode === 'preview' ? (
        <div className="asset-markdown-preview">
          <ReactMarkdown>{value}</ReactMarkdown>
        </div>
      ) : (
        <div className="asset-code-editor">
          <pre ref={lineNumberRef} className="asset-code-lines" aria-hidden="true">
            {lineNumbersFor(value)}
          </pre>
          <textarea
            value={value}
            onChange={(event) => onChange(event.target.value.replace(/\r\n/g, '\n').replace(/\r/g, '\n'))}
            onScroll={(event) => {
              if (lineNumberRef.current) {
                lineNumberRef.current.scrollTop = event.currentTarget.scrollTop
              }
            }}
            spellCheck={false}
            placeholder="该素材当前没有解析文本，可以在这里补充给 AI 使用的内容。"
            className="asset-code-textarea"
          />
        </div>
      )}
    </div>
  )
}
