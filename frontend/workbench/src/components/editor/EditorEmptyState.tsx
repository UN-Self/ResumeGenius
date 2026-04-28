import { FileEdit } from 'lucide-react'

export function EditorEmptyState() {
  return (
    <div className="empty-state">
      <FileEdit size={64} className="text-[var(--color-text-disabled)]" />
      <h3 className="text-base font-medium text-[var(--color-text-secondary)]">暂无简历内容</h3>
      <p className="text-xs font-normal text-[var(--color-text-disabled)] max-w-sm">
        开始编辑你的简历，或使用 AI 助手生成初稿
      </p>
    </div>
  )
}
