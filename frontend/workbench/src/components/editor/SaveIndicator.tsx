import { Loader2, Check, AlertCircle } from 'lucide-react'
import type { SaveStatus } from '@/hooks/useAutoSave'

interface SaveIndicatorProps {
  status: SaveStatus
  lastSavedAt: Date | null
  onRetry?: () => void
}

export function SaveIndicator({ status, lastSavedAt, onRetry }: SaveIndicatorProps) {
  if (status === 'idle') return null

  const timeStr = lastSavedAt
    ? lastSavedAt.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    : ''

  return (
    <div aria-live="polite" className="flex items-center gap-1 text-xs">
      {status === 'saving' && (
        <>
          <Loader2 size={14} className="animate-spin text-muted-foreground" />
          <span className="text-muted-foreground">保存中...</span>
        </>
      )}
      {status === 'saved' && (
        <>
          <Check size={14} className="text-primary" />
          <span className="text-primary">已保存 {timeStr}</span>
        </>
      )}
      {status === 'error' && (
        <button
          onClick={onRetry}
          className="flex items-center gap-1 text-destructive hover:underline cursor-pointer"
        >
          <AlertCircle size={14} />
          <span>保存失败，点击重试</span>
        </button>
      )}
    </div>
  )
}
