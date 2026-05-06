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
        p-2 rounded-lg transition-all duration-150 ease-in-out
        min-w-10 min-h-10 flex items-center justify-center
        ${isActive
          ? 'bg-surface-hover text-primary shadow-[inset_0_0_0_1px_var(--color-border-glow)]'
          : 'text-muted-foreground hover:bg-surface-hover hover:text-foreground'
        }
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
        focus:outline-none focus:ring-2 focus:ring-ring/35
      `}
    >
      {icon}
    </button>
  )
}
