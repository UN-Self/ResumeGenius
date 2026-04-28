import { useState, useEffect } from 'react'
import type { Asset } from '@/lib/api-client'

interface NoteDialogProps {
  open: boolean
  onClose: () => void
  onSubmit: (content: string, label: string) => Promise<void>
  initialNote?: Asset
}

export default function NoteDialog({ open, onClose, onSubmit, initialNote }: NoteDialogProps) {
  const [label, setLabel] = useState('')
  const [content, setContent] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open) {
      setLabel(initialNote?.label ?? '')
      setContent(initialNote?.content ?? '')
      setError('')
    }
  }, [open, initialNote])

  const handleSubmit = async () => {
    const trimmedContent = content.trim()
    if (!trimmedContent) {
      setError('请输入备注内容')
      return
    }
    try {
      setSubmitting(true)
      setError('')
      await onSubmit(trimmedContent, label.trim())
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  if (!open) return null

  const isEdit = !!initialNote

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-card rounded-lg border border-border shadow-lg p-6 w-full max-w-md mx-4">
        <h3 className="text-base font-serif font-semibold text-foreground">
          {isEdit ? '编辑备注' : '添加备注'}
        </h3>

        <input
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          placeholder="标签（可选）"
          className="mt-4 w-full h-10 px-4 text-sm rounded-lg border border-border bg-card text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent transition-shadow"
        />

        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="输入备注内容..."
          rows={4}
          className="mt-3 w-full px-4 py-3 text-sm rounded-lg border border-border bg-card text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent transition-shadow resize-none"
        />

        {error && (
          <p className="text-xs text-destructive mt-2">{error}</p>
        )}

        <div className="flex justify-end gap-2 mt-5">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm rounded-lg border border-border text-foreground hover:bg-accent transition-colors"
          >
            取消
          </button>
          <button
            onClick={handleSubmit}
            disabled={!content.trim() || submitting}
            className="px-4 py-2 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            {submitting ? '保存中...' : isEdit ? '保存' : '添加'}
          </button>
        </div>
      </div>
    </div>
  )
}
