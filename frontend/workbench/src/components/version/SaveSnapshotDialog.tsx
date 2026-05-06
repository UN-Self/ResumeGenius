import { useState } from 'react'
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface SaveSnapshotDialogProps {
  open: boolean
  saving?: boolean
  onClose: () => void
  onConfirm: (label: string) => void
}

export function SaveSnapshotDialog({ open, saving, onClose, onConfirm }: SaveSnapshotDialogProps) {
  const [label, setLabel] = useState('')

  const handleConfirm = () => {
    onConfirm(label)
    setLabel('')
  }

  const handleClose = () => {
    if (saving) return
    setLabel('')
    onClose()
  }

  return (
    <Modal open={open} onClose={handleClose}>
      <ModalHeader>保存快照</ModalHeader>
      <ModalBody>
        <Input
          placeholder="可选，如「校招版」「精简版」"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !saving) handleConfirm()
          }}
        />
      </ModalBody>
      <ModalFooter>
        <Button variant="secondary" size="sm" onClick={handleClose} disabled={saving}>
          取消
        </Button>
        <Button size="sm" onClick={handleConfirm} disabled={saving}>
          {saving ? '保存中...' : '确认'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
