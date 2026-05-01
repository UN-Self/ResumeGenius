import type { Asset } from '@/lib/api-client'
import { getAssetVisual } from './fileVisuals'

type AssetItem = Asset & { label?: string; content?: string; uri?: string }

interface AssetListProps {
  assets: AssetItem[]
  onDelete: (id: number) => void
  onEditNote: (asset: AssetItem) => void
}

export default function AssetList({ assets, onDelete, onEditNote }: AssetListProps) {
  if (assets.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        <p className="text-sm">{'\u8fd8\u6ca1\u6709\u6dfb\u52a0\u4efb\u4f55\u8d44\u6599'}</p>
        <p className="mt-1 text-xs">{'\u70b9\u51fb\u4e0a\u65b9\u6309\u94ae\u4e0a\u4f20\u6587\u4ef6\u6216\u6dfb\u52a0\u5907\u6ce8'}</p>
      </div>
    )
  }

  return (
    <div className="divide-y divide-border rounded-lg border border-border bg-card">
      {assets.map((asset) => {
        const visual = getAssetVisual(asset.type)
        const Icon = visual.icon

        return (
          <div key={asset.id} className="flex items-start justify-between gap-3 px-5 py-3.5">
            <div className="flex min-w-0 flex-1 gap-3">
              <div className={`mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border ${visual.iconWrapperClassName}`}>
                <Icon className={`h-5 w-5 ${visual.iconClassName}`} />
              </div>

              <div className="min-w-0 flex-1">
                <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold ${visual.chipClassName}`}>
                  {visual.typeLabel}
                </span>

                {asset.label && (
                  <p className="mt-1 text-sm text-foreground truncate">{asset.label}</p>
                )}

                {asset.content && !asset.label && (
                  <p className="mt-1 text-sm text-foreground truncate">{asset.content}</p>
                )}

                {asset.uri && asset.type === 'git_repo' && (
                  <p className="mt-1 text-xs text-muted-foreground truncate">{asset.uri}</p>
                )}

                {asset.uri && asset.type.startsWith('resume_') && (
                  <p className="mt-1 text-xs text-muted-foreground truncate">{asset.uri.split('/').pop()}</p>
                )}
              </div>
            </div>

            <div className="flex shrink-0 items-center gap-1">
              {asset.type === 'note' && (
                <button
                  onClick={() => onEditNote(asset)}
                  className="rounded px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-primary-50 hover:text-foreground"
                >
                  {'\u7f16\u8f91'}
                </button>
              )}

              <button
                onClick={() => onDelete(asset.id)}
                className="rounded px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
              >
                {'\u5220\u9664'}
              </button>
            </div>
          </div>
        )
      })}
    </div>
  )
}
