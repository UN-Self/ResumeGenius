import { useState } from 'react'

interface GitRepoDialogProps {
  open: boolean
  onClose: () => void
  onSubmit: (repoUrl: string) => Promise<void>
}

export default function GitRepoDialog({ open, onClose, onSubmit }: GitRepoDialogProps) {
  const [repoUrl, setRepoUrl] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    const trimmed = repoUrl.trim()
    if (!trimmed) {
      setError('请输入 Git 仓库地址')
      return
    }
    try {
      setSubmitting(true)
      setError('')
      await onSubmit(trimmed)
      setRepoUrl('')
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleClose = () => {
    setRepoUrl('')
    setError('')
    onClose()
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={handleClose} />
      <div className="relative bg-card rounded-lg border border-border shadow-lg p-6 w-full max-w-md mx-4">
        <h3 className="text-base font-serif font-semibold text-foreground">接入 Git 仓库</h3>
        <p className="text-xs text-muted-foreground mt-1">输入 GitHub / GitLab 仓库的 HTTPS 地址</p>

        <input
          value={repoUrl}
          onChange={(e) => setRepoUrl(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
          placeholder="https://github.com/user/repo"
          className="mt-4 w-full h-10 px-4 text-sm rounded-lg border border-border bg-card text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent transition-shadow"
        />

        {error && (
          <p className="text-xs text-destructive mt-2">{error}</p>
        )}

        <div className="flex justify-end gap-2 mt-5">
          <button
            onClick={handleClose}
            className="px-4 py-2 text-sm rounded-lg border border-border text-foreground hover:bg-accent transition-colors"
          >
            取消
          </button>
          <button
            onClick={handleSubmit}
            disabled={!repoUrl.trim() || submitting}
            className="px-4 py-2 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            {submitting ? '接入中...' : '接入'}
          </button>
        </div>
      </div>
    </div>
  )
}
