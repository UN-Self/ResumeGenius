import { useState, useEffect } from 'react'
import type { Asset } from '@/lib/api-client'
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

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

  const isEdit = !!initialNote

  return (
    <Modal open={open} onClose={onClose}>
      <ModalHeader>
        {isEdit ? '编辑备注' : '添加备注'}
      </ModalHeader>

      <Input
        value={label}
        onChange={(e) => setLabel(e.target.value)}
        placeholder="标签（可选）"
        className="mt-4"
      />

      <Textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="输入备注内容..."
        rows={4}
        className="mt-3"
      />

      {error && (
        <p className="text-xs text-destructive mt-2">{error}</p>
      )}

      <ModalFooter>
        <Button variant="secondary" onClick={onClose}>
          取消
        </Button>
        <Button onClick={handleSubmit} disabled={!content.trim() || submitting}>
          {submitting ? '保存中...' : isEdit ? '保存' : '添加'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
