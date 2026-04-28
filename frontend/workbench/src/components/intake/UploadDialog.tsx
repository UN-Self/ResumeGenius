import { useState, useRef, useCallback } from 'react'

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

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={handleClose} />
      <div className="relative bg-card rounded-lg border border-border shadow-lg p-6 w-full max-w-md mx-4">
        <h3 className="text-base font-serif font-semibold text-foreground">上传文件</h3>
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

        <div className="flex justify-end gap-2 mt-5">
          <button
            onClick={handleClose}
            className="px-4 py-2 text-sm rounded-lg border border-border text-foreground hover:bg-accent transition-colors"
          >
            取消
          </button>
          <button
            onClick={handleSubmit}
            disabled={!file || uploading}
            className="px-4 py-2 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            {uploading ? '上传中...' : '上传'}
          </button>
        </div>
      </div>
    </div>
  )
}
