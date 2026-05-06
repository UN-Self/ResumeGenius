import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'

interface RollbackConfirmDialogProps {
  open: boolean
  rollbacking?: boolean
  onClose: () => void
  onConfirm: () => void
}

export function RollbackConfirmDialog({ open, rollbacking, onClose, onConfirm }: RollbackConfirmDialogProps) {
  return (
    <Modal open={open} onClose={onClose}>
      <ModalHeader>确认回退</ModalHeader>
      <ModalBody>
        <p className="text-sm text-muted-foreground">
          回退将覆盖当前编辑内容，是否继续？
        </p>
      </ModalBody>
      <ModalFooter>
        <Button variant="secondary" size="sm" onClick={onClose} disabled={rollbacking}>
          取消
        </Button>
        <Button variant="danger" size="sm" onClick={onConfirm} disabled={rollbacking}>
          {rollbacking ? '回退中...' : '确认回退'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
