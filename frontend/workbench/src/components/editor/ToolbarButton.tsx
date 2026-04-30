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
          ? 'bg-[var(--color-primary-bg)] text-[var(--color-primary)] hover:bg-[var(--color-primary-hover)]'
          : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-page-bg)]'
        }
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
        focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]
      `}
    >
      {icon}
    </button>
  )
}
