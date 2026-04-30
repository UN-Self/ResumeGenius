import { FileText } from 'lucide-react'
import type { ReactNode } from 'react'
import type { ExportStatus } from '@/hooks/useExport'

interface ActionBarProps {
  projectName: string
  saveIndicator?: ReactNode
  draftId: string | null
  exportStatus: ExportStatus
  onExport: () => void
}

const EXPORT_LABEL: Record<ExportStatus, string> = {
  idle: '导出 PDF',
  exporting: '导出中...',
  completed: '导出 PDF',
  failed: '导出失败',
}

export function ActionBar({
  projectName,
  saveIndicator,
  draftId,
  exportStatus,
  onExport,
}: ActionBarProps) {
  return (
    <div className="action-bar">
      {/* Logo */}
      <div className="flex items-center gap-2">
        <FileText size={24} className="text-[var(--color-primary)]" />
      </div>

      {/* Project Name */}
      <div className="h-6 w-px bg-[var(--color-divider)]" />
      <span className="text-base font-medium text-[var(--color-text-main)]">{projectName}</span>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Save Status */}
      <div className="flex items-center gap-2">
        {saveIndicator}
      </div>

      {/* Version History Button */}
      <button
        type="button"
        className="px-3 py-1.5 text-sm font-medium text-[var(--color-text-main)] hover:bg-[var(--color-primary-bg)] rounded-md transition-colors cursor-pointer"
      >
        版本历史
      </button>

      {/* Export Button */}
      <button
        type="button"
        disabled={!draftId || exportStatus === 'exporting'}
        onClick={onExport}
        className="px-3 py-1.5 text-sm font-medium text-[var(--color-text-main)] bg-[var(--color-page-bg)] border border-[var(--color-divider)] rounded-md disabled:cursor-not-allowed disabled:text-[var(--color-text-disabled)] hover:bg-[var(--color-primary-bg)] transition-colors cursor-pointer"
      >
        {EXPORT_LABEL[exportStatus]}
      </button>
    </div>
  )
}
