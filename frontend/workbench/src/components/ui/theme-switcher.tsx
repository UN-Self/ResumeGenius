import { Check, ChevronDown, Moon, Palette, SunMedium } from 'lucide-react'
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '@/lib/utils'
import {
  applyThemeChoice,
  getInitialThemeChoiceId,
  getPresetById,
  getSystemPreset,
  hasStoredPreset,
  SYSTEM_THEME_ID,
  THEME_CHOICES,
  THEME_MANUAL_STORAGE_KEY,
  THEME_STORAGE_KEY,
} from '@/lib/theme'

const MENU_WIDTH = 176
const MENU_HEIGHT = 216
const MENU_GAP = 8
const VIEWPORT_PADDING = 12

export function ThemeSwitcher({
  className,
  compact = false,
}: {
  className?: string
  compact?: boolean
}) {
  const [choiceId, setChoiceId] = useState(() => getInitialThemeChoiceId())
  const [systemPresetId, setSystemPresetId] = useState(() => getSystemPreset().id)
  const [open, setOpen] = useState(false)
  const [menuPosition, setMenuPosition] = useState({ left: 0, top: 0 })
  const rootRef = useRef<HTMLDivElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const activeChoice = THEME_CHOICES.find((choice) => choice.id === choiceId) ?? THEME_CHOICES[0]
  const activePreset = choiceId === SYSTEM_THEME_ID ? getPresetById(systemPresetId) : getPresetById(choiceId)

  const updateMenuPosition = useCallback(() => {
    const rect = rootRef.current?.getBoundingClientRect()
    if (!rect) return

    const left = Math.min(
      window.innerWidth - MENU_WIDTH - VIEWPORT_PADDING,
      Math.max(VIEWPORT_PADDING, rect.right - MENU_WIDTH),
    )
    const preferredTop = rect.bottom + MENU_GAP
    const top = preferredTop + MENU_HEIGHT > window.innerHeight - VIEWPORT_PADDING
      ? Math.max(VIEWPORT_PADDING, rect.top - MENU_GAP - MENU_HEIGHT)
      : preferredTop

    setMenuPosition({ left, top })
  }, [])

  useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== THEME_STORAGE_KEY && event.key !== THEME_MANUAL_STORAGE_KEY) return
      setChoiceId(getInitialThemeChoiceId())
      setSystemPresetId(getSystemPreset().id)
    }

    window.addEventListener('storage', handleStorage)
    return () => window.removeEventListener('storage', handleStorage)
  }, [])

  useEffect(() => {
    const media = window.matchMedia?.('(prefers-color-scheme: dark)')
    if (!media) return

    const handleSystemThemeChange = () => {
      if (hasStoredPreset()) return
      setChoiceId(SYSTEM_THEME_ID)
      setSystemPresetId(getSystemPreset().id)
    }

    media.addEventListener('change', handleSystemThemeChange)
    return () => media.removeEventListener('change', handleSystemThemeChange)
  }, [])

  useEffect(() => {
    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node
      if (rootRef.current?.contains(target) || panelRef.current?.contains(target)) return
      setOpen(false)
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }

    document.addEventListener('pointerdown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [])

  useLayoutEffect(() => {
    if (!open) return
    updateMenuPosition()
  }, [open, updateMenuPosition])

  useEffect(() => {
    if (!open) return

    window.addEventListener('resize', updateMenuPosition)
    window.addEventListener('scroll', updateMenuPosition, true)
    return () => {
      window.removeEventListener('resize', updateMenuPosition)
      window.removeEventListener('scroll', updateMenuPosition, true)
    }
  }, [open, updateMenuPosition])

  return (
    <div
      ref={rootRef}
      className={cn(
        'relative z-[120] inline-flex items-center gap-1 rounded-full border border-border bg-card/70 p-1 backdrop-blur-xl',
        className,
      )}
    >
      <div
        className={cn(
          'hidden items-center gap-1 px-2 text-[11px] font-medium text-muted-foreground sm:flex',
          compact && 'sm:hidden',
        )}
      >
        <Palette size={13} />
        <span>主题</span>
      </div>
      <button
        type="button"
        aria-label="选择主题"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        className={cn(
          'inline-flex h-8 items-center justify-between gap-2 rounded-full border border-transparent bg-transparent px-2 text-xs font-medium text-foreground outline-none transition-colors hover:bg-surface-hover focus:border-border-glow',
          compact ? 'min-w-0' : 'min-w-20',
        )}
      >
        <span>{activeChoice.label}</span>
        <ChevronDown size={13} className={cn('transition-transform', open && 'rotate-180')} />
      </button>
      <span className="flex h-8 w-8 items-center justify-center rounded-full bg-surface-hover text-primary">
        {activePreset.mode === 'dark' ? <Moon size={14} /> : <SunMedium size={14} />}
      </span>

      {open && createPortal(
        <div
          ref={panelRef}
          className="fixed z-[9999] w-44 overflow-hidden rounded-2xl border border-border bg-popover/95 p-1.5 text-popover-foreground shadow-[0_24px_70px_rgba(2,8,23,0.32)] backdrop-blur-2xl"
          style={{
            left: menuPosition.left,
            top: menuPosition.top,
          }}
        >
          {THEME_CHOICES.map((choice) => {
            const selected = choice.id === choiceId
            return (
              <button
                key={choice.id}
                type="button"
                onClick={() => {
                  setChoiceId(choice.id)
                  applyThemeChoice(choice.id)
                  setSystemPresetId(getSystemPreset().id)
                  setOpen(false)
                }}
                className={cn(
                  'flex w-full items-center justify-between rounded-xl px-3 py-2 text-left text-xs font-medium transition-colors',
                  selected
                    ? 'bg-primary text-primary-foreground shadow-[0_10px_26px_color-mix(in_srgb,var(--color-primary),transparent_76%)]'
                    : 'text-muted-foreground hover:bg-surface-hover hover:text-foreground',
                )}
              >
                <span>{choice.label}</span>
                {selected && <Check size={14} />}
              </button>
            )
          })}
        </div>,
        document.body,
      )}
    </div>
  )
}
