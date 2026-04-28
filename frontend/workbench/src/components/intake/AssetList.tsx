import type { Asset } from '@/lib/api-client'

type AssetItem = Asset & { label?: string; content?: string; uri?: string }

interface AssetListProps {
  assets: AssetItem[]
  onDelete: (id: number) => void
  onEditNote: (asset: AssetItem) => void
}

const typeLabels: Record<string, string> = {
  resume_pdf: 'PDF 简历',
  resume_docx: 'DOCX 简历',
  resume_image: '图片',
  git_repo: 'Git 仓库',
  note: '备注',
}

const typeIcons: Record<string, string> = {
  resume_pdf: '📄',
  resume_docx: '📝',
  resume_image: '🖼',
  git_repo: '🔀',
  note: '💬',
}

export default function AssetList({ assets, onDelete, onEditNote }: AssetListProps) {
  if (assets.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <p className="text-sm">还没有添加任何资料</p>
        <p className="text-xs mt-1">点击上方按钮上传文件或添加备注</p>
      </div>
    )
  }

  return (
    <div className="divide-y divide-border rounded-lg border border-border bg-card">
      {assets.map((asset) => (
        <div key={asset.id} className="flex items-center justify-between px-5 py-3.5">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="text-sm">{typeIcons[asset.type] ?? '📎'}</span>
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                {typeLabels[asset.type] ?? asset.type}
              </span>
            </div>
            {asset.label && (
              <p className="text-sm text-foreground mt-0.5 truncate">{asset.label}</p>
            )}
            {asset.content && !asset.label && (
              <p className="text-sm text-foreground mt-0.5 truncate">{asset.content}</p>
            )}
            {asset.uri && asset.type === 'git_repo' && (
              <p className="text-xs text-muted-foreground mt-0.5 truncate">{asset.uri}</p>
            )}
            {asset.uri && asset.type.startsWith('resume_') && (
              <p className="text-xs text-muted-foreground mt-0.5 truncate">{asset.uri.split('/').pop()}</p>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0 ml-3">
            {asset.type === 'note' && (
              <button
                onClick={() => onEditNote(asset)}
                className="text-xs text-muted-foreground hover:text-foreground px-2 py-1 rounded hover:bg-accent transition-colors"
              >
                编辑
              </button>
            )}
            <button
              onClick={() => onDelete(asset.id)}
              className="text-xs text-muted-foreground hover:text-destructive px-2 py-1 rounded hover:bg-destructive/10 transition-colors"
            >
              删除
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}
