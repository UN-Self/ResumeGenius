import { Sparkles } from 'lucide-react'

export function AiPanelPlaceholder() {
  return (
    <div className="ai-panel">
      <Sparkles size={48} className="text-[var(--color-text-disabled)] mb-4" />
      <h2 className="text-xl font-semibold text-[var(--color-text-main)] mb-2">AI 助手</h2>
      <p className="text-sm font-normal text-[var(--color-text-secondary)] mb-4">即将推出</p>
      <p className="text-xs font-normal text-[var(--color-text-disabled)] max-w-xs">
        AI 助手将帮助您优化简历内容，提供智能建议，并自动生成简历初稿
      </p>
    </div>
  )
}
