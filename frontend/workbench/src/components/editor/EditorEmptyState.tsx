import { FileEdit, Plus } from 'lucide-react'

interface EditorEmptyStateProps {
  onCreateDraft?: () => void
  loading?: boolean
}

export function EditorEmptyState({ onCreateDraft, loading }: EditorEmptyStateProps) {
  return (
    <div className="empty-state">
      <FileEdit size={64} className="text-[var(--color-text-disabled)]" />
      <h3 className="text-base font-medium text-[var(--color-text-secondary)]">暂无简历内容</h3>
      <p className="text-xs font-normal text-[var(--color-text-disabled)] max-w-sm">
        开始编辑你的简历，或使用 AI 助手生成初稿
      </p>
      {onCreateDraft && (
        <button
          className="mt-4 inline-flex items-center gap-2 rounded-md bg-[var(--color-primary)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-primary-hover)] disabled:opacity-50"
          onClick={onCreateDraft}
          disabled={loading}
        >
          <Plus size={16} />
          新建草稿
        </button>
      )}
    </div>
  )
}
