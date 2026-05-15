import { useState } from 'react'
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import SSHKeySelector from './SSHKeySelector'

interface GitRepoDialogProps {
  open: boolean
  onClose: () => void
  onSubmit: (repoUrls: string[], keyId?: number) => Promise<void>
}

export default function GitRepoDialog({ open, onClose, onSubmit }: GitRepoDialogProps) {
  const [repoUrlsText, setRepoUrlsText] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [selectedKeyId, setSelectedKeyId] = useState<number | null>(null)

  const handleSubmit = async () => {
    const urls = repoUrlsText
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line !== '')

    if (urls.length === 0) {
      setError('请输入至少一个 Git 仓库 HTTPS 地址')
      return
    }
    try {
      setSubmitting(true)
      setError('')
      await onSubmit(urls, selectedKeyId ?? undefined)
      setRepoUrlsText('')
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleClose = () => {
    setRepoUrlsText('')
    setError('')
    setSelectedKeyId(null)
    onClose()
  }

  return (
    <Modal open={open} onClose={handleClose}>
      <ModalHeader>接入 Git 仓库</ModalHeader>
      <p className="text-xs text-muted-foreground mt-1">
        输入 GitHub / GitLab 仓库的 HTTPS 地址，每行一个
      </p>

      <textarea
        value={repoUrlsText}
        onChange={(e) => setRepoUrlsText(e.target.value)}
        placeholder={'https://github.com/user/repo1\nhttps://github.com/user/repo2'}
        className="mt-4 w-full rounded-md border border-input bg-background px-3 py-2 text-sm min-h-[80px] resize-y"
        rows={4}
      />

      {error && (
        <p className="text-xs text-destructive mt-2">{error}</p>
      )}

      <div className="mt-4">
        <SSHKeySelector value={selectedKeyId} onChange={setSelectedKeyId} />
      </div>

      <ModalFooter>
        <Button variant="secondary" onClick={handleClose}>
          取消
        </Button>
        <Button onClick={handleSubmit} disabled={!repoUrlsText.trim() || submitting}>
          {submitting ? '接入中...' : '接入'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
