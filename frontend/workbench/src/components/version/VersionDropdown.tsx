import { useState } from 'react'
import { History, Plus, RefreshCw } from 'lucide-react'
import type { Version } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover'

interface VersionDropdownProps {
  versions: Version[]
  loading: boolean
  error?: string | null
  onPreview: (version: Version) => void
  onSaveSnapshot: () => void
  onRetry?: () => void
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  if (isNaN(then)) return '日期未知'
  const diffSec = Math.floor((now - then) / 1000)

  if (diffSec < 60) return '刚刚'
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} 分钟前`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)} 小时前`
  return `${Math.floor(diffSec / 86400)} 天前`
}

export function VersionDropdown({
  versions,
  loading,
  error,
  onPreview,
  onSaveSnapshot,
  onRetry,
}: VersionDropdownProps) {
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="secondary" size="sm" type="button">
          <History size={14} className="mr-1" />
          版本历史
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 p-0">
        <div className="max-h-80 overflow-y-auto">
          {loading ? (
            <div className="px-4 py-6 text-center text-sm text-muted-foreground">
              加载中...
            </div>
          ) : error ? (
            <div className="px-4 py-6 text-center">
              <p className="text-sm text-destructive mb-2">{error}</p>
              {onRetry && (
                <Button variant="secondary" size="sm" onClick={onRetry}>
                  <RefreshCw size={14} className="mr-1" />
                  重试
                </Button>
              )}
            </div>
          ) : versions.length === 0 ? (
            <div className="px-4 py-6 text-center text-sm text-muted-foreground">
              暂无版本记录
            </div>
          ) : (
            versions.map((v) => (
              <button
                key={v.id}
                type="button"
                className="flex w-full items-center gap-2 px-4 py-2.5 text-left hover:bg-accent transition-colors cursor-pointer"
                onClick={() => {
                  onPreview(v)
                  setOpen(false)
                }}
              >
                <span className="flex-1 text-sm truncate">{v.label}</span>
                <span className="text-xs text-muted-foreground whitespace-nowrap">
                  {formatRelativeTime(v.created_at)}
                </span>
              </button>
            ))
          )}
        </div>
        <div className="border-t border-border p-2">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start"
            onClick={() => {
              onSaveSnapshot()
              setOpen(false)
            }}
          >
            <Plus size={14} className="mr-1" />
            保存快照
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
