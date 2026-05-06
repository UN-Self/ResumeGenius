import type { Version } from '@/lib/api-client'
import { Button } from '@/components/ui/button'

interface VersionPreviewBannerProps {
  version: Version
  onRollback: () => void
  onClose: () => void
}

export function VersionPreviewBanner({ version, onRollback, onClose }: VersionPreviewBannerProps) {
  return (
    <div className="flex items-center gap-3 bg-blue-50 border-b border-blue-200 px-4 py-2 text-sm text-blue-800">
      <span>
        正在预览: <strong>{version.label}</strong>
      </span>
      <div className="flex-1" />
      <Button variant="secondary" size="sm" onClick={onRollback}>
        回退到此版本
      </Button>
      <Button variant="ghost" size="sm" onClick={onClose}>
        关闭预览
      </Button>
    </div>
  )
}
