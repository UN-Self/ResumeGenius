import { useState } from 'react'
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'

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

  const handleConfirm = () => {
    if (!confirming) {
      setConfirming(true)
      return
    }
    onConfirm()
  }

  return (
    <Modal open={open} onClose={onCancel} maxWidth="max-w-sm">
      <ModalHeader>{title}</ModalHeader>
      <ModalBody>
        <p className="text-sm text-muted-foreground">{message}</p>
      </ModalBody>

      {!confirming ? (
        <ModalFooter>
          <Button variant="secondary" onClick={onCancel}>
            取消
          </Button>
          <Button variant="danger" onClick={handleConfirm}>
            删除
          </Button>
        </ModalFooter>
      ) : (
        <div className="mt-5">
          <p className="text-xs text-destructive font-medium">
            确定要删除吗？此操作不可撤销。再次点击确认删除。
          </p>
          <div className="flex justify-end gap-2 mt-3">
            <Button variant="secondary" onClick={() => { setConfirming(false); onCancel() }}>
              取消
            </Button>
            <Button variant="danger" onClick={handleConfirm} disabled={loading}>
              {loading ? '删除中...' : '确认删除'}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}
