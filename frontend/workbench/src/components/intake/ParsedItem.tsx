import type { ParsedContent } from '@/lib/api-client'
import { getAssetBadgeText, getAssetVisual, getDisplayAssetTitle } from './fileVisuals'

interface ParsedItemProps {
  content: ParsedContent
}

function getBodyText(content: ParsedContent) {
  const trimmedText = content.text.trim()
  const trimmedLabel = content.label.trim()

  if (content.type === 'note' && trimmedLabel && trimmedText.startsWith(trimmedLabel)) {
    return trimmedText.slice(trimmedLabel.length).replace(/^\s+/, '')
  }

  return trimmedText
}

export default function ParsedItem({ content }: ParsedItemProps) {
  const visual = getAssetVisual(content.type, content.label)
  const Icon = visual.icon
  const title = getDisplayAssetTitle(content.type, content.label) || content.label || visual.typeLabel
  const badgeText = getAssetBadgeText(content.type, content.label)
  const bodyText = getBodyText(content)

  return (
    <div className="rounded-2xl border border-border bg-card/80 p-3.5 shadow-sm">
      <div className="flex items-start gap-3">
        <div className={`mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border ${visual.iconWrapperClassName}`}>
          <Icon className={`h-5 w-5 ${visual.iconClassName}`} />
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="min-w-0 break-all text-sm font-semibold text-foreground">{title}</p>
            <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold tracking-wide ${visual.chipClassName}`}>
              {badgeText}
            </span>
          </div>
        </div>
      </div>

      <div className="mt-3 max-h-48 overflow-y-auto rounded-xl bg-background px-3 py-2.5 text-[13px] leading-relaxed text-muted-foreground whitespace-pre-wrap">
        {bodyText}
      </div>
    </div>
  )
}
