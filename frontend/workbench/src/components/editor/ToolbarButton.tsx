import { type ReactNode } from 'react'

interface ToolbarButtonProps {
  onClick: () => void
  isActive: boolean
  icon: ReactNode
  label: string
  disabled?: boolean
}

export function ToolbarButton({ onClick, isActive, icon, label, disabled }: ToolbarButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      aria-pressed={isActive}
      className={`
        p-2 rounded-md transition-all duration-150 ease-in-out
        min-w-[44px] min-h-[44px] flex items-center justify-center
        ${isActive
          ? 'bg-[#e8f0fe] text-[#1a73e8] hover:bg-[#d2e3fc]'
          : 'text-[#5f6368] hover:bg-[#f8f9fa]'
        }
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
        focus:outline-none focus:ring-2 focus:ring-[#1a73e8]
      `}
    >
      {icon}
    </button>
  )
}