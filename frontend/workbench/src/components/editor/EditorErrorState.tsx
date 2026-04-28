import { AlertTriangle } from 'lucide-react'

interface EditorErrorStateProps {
  message?: string
  detail?: string
  onRetry?: () => void
}

export function EditorErrorState({
  message = '加载失败',
  detail,
  onRetry
}: EditorErrorStateProps) {
  return (
    <div className="flex flex-col items-center justify-center h-full min-h-[400px] text-center p-8">
      <AlertTriangle size={48} className="text-[#c5221f] mb-4" />
      <h3 className="text-base font-medium text-[#c5221f] mb-2">{message}</h3>
      {detail && (
        <p className="text-xs text-[#5f6368] mb-2">{detail}</p>
      )}
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-4 px-4 py-2 text-sm font-medium rounded-md border border-[#1a73e8] text-[#1a73e8] hover:bg-[#e8f0fe] transition-colors cursor-pointer"
        >
          重新加载
        </button>
      )}
    </div>
  )
}