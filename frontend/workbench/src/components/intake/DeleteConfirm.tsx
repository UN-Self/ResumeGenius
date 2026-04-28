import { useState } from 'react'

interface DeleteConfirmProps {
  title: string
  message: string
  open: boolean
  onConfirm: () => void
  onCancel: () => void
  loading?: boolean
}

export default function DeleteConfirm({ title, message, open, onConfirm, onCancel, loading }: DeleteConfirmProps) {
  const [confirming, setConfirming] = useState(false)

  if (!open) return null

  const handleConfirm = () => {
    if (!confirming) {
      setConfirming(true)
      return
    }
    onConfirm()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={onCancel} />
      <div className="relative bg-card rounded-lg border border-border shadow-lg p-6 w-full max-w-sm mx-4">
        <h3 className="text-base font-serif font-semibold text-foreground">{title}</h3>
        <p className="text-sm text-muted-foreground mt-2">{message}</p>

        {!confirming ? (
          <div className="flex justify-end gap-2 mt-5">
            <button
              onClick={onCancel}
              className="px-4 py-2 text-sm rounded-lg border border-border text-foreground hover:bg-accent transition-colors"
            >
              取消
            </button>
            <button
              onClick={handleConfirm}
              className="px-4 py-2 text-sm rounded-lg bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors"
            >
              删除
            </button>
          </div>
        ) : (
          <div className="mt-5">
            <p className="text-xs text-destructive font-medium">
              确定要删除吗？此操作不可撤销。再次点击确认删除。
            </p>
            <div className="flex justify-end gap-2 mt-3">
              <button
                onClick={() => { setConfirming(false); onCancel() }}
                className="px-4 py-2 text-sm rounded-lg border border-border text-foreground hover:bg-accent transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleConfirm}
                disabled={loading}
                className="px-4 py-2 text-sm rounded-lg bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors disabled:opacity-50"
              >
                {loading ? '删除中...' : '确认删除'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
