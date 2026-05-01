import { useCallback, useRef, useState } from 'react'
import { Upload } from 'lucide-react'
import { Modal, ModalFooter, ModalHeader } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import { formatFileSize, getDisplayFileName, getExt, getUploadFileVisual } from './fileVisuals'

const MAX_SIZE = 20 * 1024 * 1024
const ALLOWED = ['.pdf', '.docx', '.png', '.jpg', '.jpeg']

interface UploadDialogProps {
  open: boolean
  onClose: () => void
  onUpload: (file: File) => Promise<void>
}

export default function UploadDialog({ open, onClose, onUpload }: UploadDialogProps) {
  const [dragging, setDragging] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const validate = useCallback((nextFile: File) => {
    if (!ALLOWED.includes(getExt(nextFile.name))) {
      return '\u4e0d\u652f\u6301\u7684\u6587\u4ef6\u683c\u5f0f\uff0c\u8bf7\u4e0a\u4f20 PDF\u3001DOCX\u3001PNG \u6216 JPG \u6587\u4ef6'
    }

    if (nextFile.size > MAX_SIZE) {
      return '\u6587\u4ef6\u5927\u5c0f\u8d85\u8fc7 20MB \u9650\u5236'
    }

    return ''
  }, [])

  const handleFile = useCallback((nextFile: File) => {
    const nextError = validate(nextFile)
    if (nextError) {
      setFile(null)
      setError(nextError)
      return
    }

    setError('')
    setFile(nextFile)
  }, [validate])

  const handleDrop = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    setDragging(false)

    const nextFile = event.dataTransfer.files[0]
    if (nextFile) {
      handleFile(nextFile)
    }
  }, [handleFile])

  const handleDragOver = (event: React.DragEvent) => {
    event.preventDefault()
    setDragging(true)
  }

  const handleDragLeave = () => setDragging(false)

  const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0]
    if (nextFile) {
      handleFile(nextFile)
    }
  }

  const openPicker = () => {
    if (!inputRef.current) return

    inputRef.current.value = ''
    inputRef.current.click()
  }

  const handleSubmit = async () => {
    if (!file) return

    try {
      setUploading(true)
      setError('')
      await onUpload(file)
      setFile(null)
      onClose()
    } catch (eventError) {
      setError(eventError instanceof Error ? eventError.message : '\u4e0a\u4f20\u5931\u8d25')
    } finally {
      setUploading(false)
    }
  }

  const handleClose = () => {
    setDragging(false)
    setFile(null)
    setError('')
    onClose()
  }

  const selectedVisual = file ? getUploadFileVisual(file.name) : null
  const SelectedIcon = selectedVisual?.icon

  return (
    <Modal open={open} onClose={handleClose}>
      <ModalHeader>{'\u4e0a\u4f20\u6587\u4ef6'}</ModalHeader>
      <p className="mt-1 text-xs text-muted-foreground">{'\u652f\u6301 PDF\u3001DOCX\u3001PNG\u3001JPG\uff0c\u6700\u5927 20MB'}</p>

      <div
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onClick={openPicker}
        className={`mt-4 cursor-pointer rounded-2xl border-2 border-dashed p-6 transition-colors ${
          dragging
            ? 'border-primary bg-primary-50'
            : file
              ? 'border-primary/60 bg-primary-50/70'
              : 'border-border hover:border-primary/50 hover:bg-primary-50/40'
        }`}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".pdf,.docx,.png,.jpg,.jpeg"
          aria-label="Upload file"
          className="hidden"
          onChange={handleInputChange}
        />

        {file && selectedVisual && SelectedIcon ? (
          <div className="flex items-center gap-4 text-left">
            <div className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl border ${selectedVisual.iconWrapperClassName}`}>
              <SelectedIcon className={`h-7 w-7 ${selectedVisual.iconClassName}`} />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <p className="min-w-0 break-all text-sm font-semibold text-foreground">{getDisplayFileName(file.name)}</p>
                <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold tracking-wide ${selectedVisual.chipClassName}`}>
                  {selectedVisual.chipLabel}
                </span>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">{formatFileSize(file.size)}</p>
              <p className="mt-1 text-xs text-muted-foreground">{'\u70b9\u51fb\u6b64\u533a\u57df\u53ef\u91cd\u65b0\u9009\u62e9\u6587\u4ef6'}</p>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-3 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-primary-200 bg-primary-50">
              <Upload className="h-6 w-6 text-primary-700" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{'\u62d6\u62fd\u6587\u4ef6\u5230\u6b64\u5904\uff0c\u6216\u70b9\u51fb\u9009\u62e9'}</p>
              <p className="mt-1 text-xs text-muted-foreground">{'\u9009\u62e9\u540e\u4f1a\u6309\u6587\u4ef6\u7c7b\u578b\u5c55\u793a\u5bf9\u5e94\u56fe\u6807'}</p>
            </div>
          </div>
        )}
      </div>

      {error && (
        <p className="mt-2 text-xs text-destructive">{error}</p>
      )}

      <ModalFooter>
        <Button variant="secondary" onClick={handleClose}>
          {'\u53d6\u6d88'}
        </Button>
        <Button onClick={handleSubmit} disabled={!file || uploading}>
          {uploading ? '\u4e0a\u4f20\u4e2d...' : '\u4e0a\u4f20'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
