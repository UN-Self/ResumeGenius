import { ImageOff, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
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

  useEffect(() => {
    setImageError(false)
  }, [asset.id])

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

  return (
    <div className="asset-workspace">
      <div className="asset-workspace-header">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-foreground">{title}</p>
          <p className="mt-1 text-xs text-muted-foreground">解析文本</p>
        </div>
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
    </div>
  )
}
