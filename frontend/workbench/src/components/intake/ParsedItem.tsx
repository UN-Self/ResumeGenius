import type { ParsedContent } from '@/lib/api-client'

const TYPE_ICONS: Record<string, string> = {
  resume_pdf: '📄',
  resume_docx: '📝',
  note: '💬',
  git_repo: '💻',
}

interface ParsedItemProps {
  content: ParsedContent
}

export default function ParsedItem({ content }: ParsedItemProps) {
  const icon = TYPE_ICONS[content.type] || '📄'

  return (
    <div className="rounded-lg bg-background p-3">
      <div className="mb-1.5 text-xs font-medium text-primary">
        {icon} {content.label}
      </div>
      <div className="max-h-48 overflow-y-auto text-[13px] leading-relaxed text-muted-foreground whitespace-pre-wrap">
        {content.text}
      </div>
    </div>
  )
}
