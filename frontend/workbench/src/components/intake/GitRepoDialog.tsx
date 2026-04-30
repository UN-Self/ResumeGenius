import { useState } from 'react'
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

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

  return (
    <Modal open={open} onClose={handleClose}>
      <ModalHeader>接入 Git 仓库</ModalHeader>
      <p className="text-xs text-muted-foreground mt-1">输入 GitHub / GitLab 仓库的 HTTPS 地址</p>

      <Input
        value={repoUrl}
        onChange={(e) => setRepoUrl(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
        placeholder="https://github.com/user/repo"
        className="mt-4"
      />

      {error && (
        <p className="text-xs text-destructive mt-2">{error}</p>
      )}

      <ModalFooter>
        <Button variant="secondary" onClick={handleClose}>
          取消
        </Button>
        <Button onClick={handleSubmit} disabled={!repoUrl.trim() || submitting}>
          {submitting ? '接入中...' : '接入'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
