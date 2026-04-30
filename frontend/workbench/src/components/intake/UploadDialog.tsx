import { useState, useRef, useCallback } from 'react'
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'

const MAX_SIZE = 20 * 1024 * 1024
const ALLOWED = ['.pdf', '.docx', '.png', '.jpg', '.jpeg']

interface UploadDialogProps {
  open: boolean
  onClose: () => void
  onUpload: (file: File) => Promise<void>
}

function getExt(name: string) {
  return name.substring(name.lastIndexOf('.')).toLowerCase()
}

export default function UploadDialog({ open, onClose, onUpload }: UploadDialogProps) {
  const [dragging, setDragging] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const validate = useCallback((f: File) => {
    if (!ALLOWED.includes(getExt(f.name))) {
      return '不支持的文件格式，请上传 PDF、DOCX、PNG 或 JPG 文件'
    }
    if (f.size > MAX_SIZE) {
      return '文件大小超过 20MB 限制'
    }
    return ''
  }, [])

  const handleFile = useCallback((f: File) => {
    const err = validate(f)
    if (err) {
      setError(err)
      return
    }
    setError('')
    setFile(f)
  }, [validate])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragging(false)
    const f = e.dataTransfer.files[0]
    if (f) handleFile(f)
  }, [handleFile])

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setDragging(true)
  }

  const handleDragLeave = () => setDragging(false)

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (f) handleFile(f)
  }

  const handleSubmit = async () => {
    if (!file) return
    try {
      setUploading(true)
      setError('')
      await onUpload(file)
      setFile(null)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '上传失败')
    } finally {
      setUploading(false)
    }
  }

  const handleClose = () => {
    setFile(null)
    setError('')
    onClose()
  }

  return (
    <Modal open={open} onClose={handleClose}>
      <ModalHeader>上传文件</ModalHeader>
      <p className="text-xs text-muted-foreground mt-1">支持 PDF、DOCX、PNG、JPG，最大 20MB</p>

      <div
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onClick={() => inputRef.current?.click()}
        className={`mt-4 border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
          dragging
            ? 'border-primary bg-accent'
            : file
              ? 'border-primary/50 bg-accent/50'
              : 'border-border hover:border-primary/50'
        }`}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".pdf,.docx,.png,.jpg,.jpeg"
          className="hidden"
          onChange={handleInputChange}
        />
        {file ? (
          <div>
            <p className="text-sm font-medium text-foreground">{file.name}</p>
            <p className="text-xs text-muted-foreground mt-1">
              {(file.size / 1024).toFixed(1)} KB
            </p>
          </div>
        ) : (
          <div>
            <p className="text-sm text-muted-foreground">拖拽文件到此处，或点击选择</p>
          </div>
        )}
      </div>

      {error && (
        <p className="text-xs text-destructive mt-2">{error}</p>
      )}

      <ModalFooter>
        <Button variant="secondary" onClick={handleClose}>
          取消
        </Button>
        <Button onClick={handleSubmit} disabled={!file || uploading}>
          {uploading ? '上传中...' : '上传'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
