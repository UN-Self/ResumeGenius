import { useEffect, useMemo, useState } from 'react'
import type { Asset } from '@/lib/api-client'
import { Modal, ModalFooter, ModalHeader } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

interface AssetEditorDialogProps {
  open: boolean
  asset: Asset | null
  onClose: () => void
  onSubmit: (content: string, label: string) => Promise<void>
}

interface DialogCopy {
  title: string
  contentPlaceholder: string
  emptyContentError: string
}

function dialogCopyForAsset(type?: string): DialogCopy {
  switch (type) {
    case 'note':
      return {
        title: '\u7f16\u8f91\u5907\u6ce8',
        contentPlaceholder: '\u8f93\u5165\u5907\u6ce8\u5185\u5bb9...',
        emptyContentError: '\u8bf7\u8f93\u5165\u5907\u6ce8\u5185\u5bb9',
      }
    case 'git_repo':
      return {
        title: '\u7f16\u8f91 Git \u7d20\u6750',
        contentPlaceholder: '\u53ef\u5728\u6b64\u4fee\u6b63\u7ed9 AI \u4f7f\u7528\u7684 Git \u6458\u8981...',
        emptyContentError: '\u8bf7\u8f93\u5165 Git \u7d20\u6750\u6b63\u6587',
      }
    case 'resume_pdf':
    case 'resume_docx':
      return {
        title: '\u7f16\u8f91\u89e3\u6790\u6587\u672c',
        contentPlaceholder: '\u53ef\u5728\u6b64\u4fee\u6b63\u89e3\u6790\u540e\u7684\u7b80\u5386\u6587\u672c...',
        emptyContentError: '\u8bf7\u8f93\u5165\u89e3\u6790\u540e\u7684\u6b63\u6587',
      }
    default:
      return {
        title: '\u7f16\u8f91\u7d20\u6750',
        contentPlaceholder: '\u8f93\u5165\u7ed9 AI \u4f7f\u7528\u7684\u7d20\u6750\u6b63\u6587...',
        emptyContentError: '\u8bf7\u8f93\u5165\u7d20\u6750\u6b63\u6587',
      }
  }
}

export default function AssetEditorDialog({
  open,
  asset,
  onClose,
  onSubmit,
}: AssetEditorDialogProps) {
  const [label, setLabel] = useState('')
  const [content, setContent] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const copy = useMemo(() => dialogCopyForAsset(asset?.type), [asset?.type])

  useEffect(() => {
    if (!open) return
    setLabel(asset?.label ?? '')
    setContent(asset?.content ?? '')
    setError('')
  }, [asset, open])

  const handleSubmit = async () => {
    const trimmedContent = content.trim()
    if (!trimmedContent) {
      setError(copy.emptyContentError)
      return
    }

    try {
      setSubmitting(true)
      setError('')
      await onSubmit(trimmedContent, label.trim())
      onClose()
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : '\u4fdd\u5b58\u5931\u8d25')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose}>
      <ModalHeader>{copy.title}</ModalHeader>

      <Input
        value={label}
        onChange={(event) => setLabel(event.target.value)}
        placeholder="标题（可选）"
        className="mt-4"
      />

      <Textarea
        value={content}
        onChange={(event) => setContent(event.target.value)}
        placeholder={copy.contentPlaceholder}
        rows={8}
        className="mt-3"
      />

      {error && (
        <p className="mt-2 text-xs text-destructive">{error}</p>
      )}

      <ModalFooter>
        <Button variant="secondary" onClick={onClose}>
          {'\u53d6\u6d88'}
        </Button>
        <Button onClick={handleSubmit} disabled={!content.trim() || submitting}>
          {submitting ? '\u4fdd\u5b58\u4e2d...' : '\u4fdd\u5b58'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
