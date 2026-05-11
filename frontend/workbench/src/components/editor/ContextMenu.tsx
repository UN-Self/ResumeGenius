import { useRef, useLayoutEffect, useState } from 'react'
import { type Editor } from '@tiptap/react'
import { Undo2, Redo2, Scissors, Copy, ClipboardPaste, MousePointerClick } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { captureCopy, sliceFromJson, getMimeType, getLastCopy } from '@/lib/clipboard'

export interface ContextMenuProps {
  editor: Editor | null
  isOpen: boolean
  x: number
  y: number
  onClose: () => void
}

interface MenuItem {
  label: string
  shortcut: string
  icon: LucideIcon
  disabled: boolean
  action: () => void
}

function Separator() {
  return <div role="separator" className="h-px bg-border my-1 mx-1" />
}

export function ContextMenu({ editor, isOpen, x, y, onClose }: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState({ x, y })
  const [clipboardError, setClipboardError] = useState<string | null>(null)

  // Adjust position to keep menu within viewport boundaries
  useLayoutEffect(() => {
    if (!isOpen) {
      setClipboardError(null)
      return
    }
    const menu = menuRef.current
    if (!menu) return

    const rect = menu.getBoundingClientRect()
    const viewportW = window.innerWidth
    const viewportH = window.innerHeight

    let adjustedX = x
    let adjustedY = y

    if (rect.right > viewportW) {
      adjustedX = viewportW - rect.width - 8
    }
    if (rect.bottom > viewportH) {
      adjustedY = viewportH - rect.height - 8
    }
    if (adjustedX < 0) adjustedX = 4
    if (adjustedY < 0) adjustedY = 4

    if (adjustedX !== position.x || adjustedY !== position.y) {
      setPosition({ x: adjustedX, y: adjustedY })
    }
  }, [isOpen, x, y, position.x, position.y])

  if (!isOpen || !editor) return null

  const { from, to } = editor.state.selection
  const hasSelection = from !== to

  const items: MenuItem[] = [
    {
      label: '撤销',
      shortcut: 'Ctrl+Z',
      icon: Undo2,
      disabled: !editor.can().undo(),
      action: () => {
        editor.chain().focus().undo().run()
        onClose()
      },
    },
    {
      label: '重做',
      shortcut: 'Ctrl+Y',
      icon: Redo2,
      disabled: !editor.can().redo(),
      action: () => {
        editor.chain().focus().redo().run()
        onClose()
      },
    },
    {
      label: '剪切',
      shortcut: 'Ctrl+X',
      icon: Scissors,
      disabled: !hasSelection,
      action: () => {
        const { text, json } = captureCopy(editor.state, from, to)
        const item = new ClipboardItem({
          'text/plain': new Blob([text], { type: 'text/plain' }),
          [getMimeType()]: new Blob([json], { type: getMimeType() }),
        })
        navigator.clipboard.write([item]).then(() => {
          editor.chain().focus().deleteSelection().run()
          onClose()
        }).catch((err) => {
          console.error('Clipboard write failed:', err)
          setClipboardError('剪贴板访问被拒绝')
        })
      },
    },
    {
      label: '复制',
      shortcut: 'Ctrl+C',
      icon: Copy,
      disabled: !hasSelection,
      action: () => {
        const { text, json } = captureCopy(editor.state, from, to)
        const item = new ClipboardItem({
          'text/plain': new Blob([text], { type: 'text/plain' }),
          [getMimeType()]: new Blob([json], { type: getMimeType() }),
        })
        navigator.clipboard.write([item]).then(() => {
          onClose()
        }).catch((err) => {
          console.error('Clipboard write failed:', err)
          setClipboardError('剪贴板访问被拒绝')
        })
      },
    },
    {
      label: '粘贴',
      shortcut: 'Ctrl+V',
      icon: ClipboardPaste,
      disabled: false,
      action: async () => {
        try {
          const internal = getLastCopy()
          if (internal) {
            const currentText = await navigator.clipboard.readText()
            if (currentText === internal.text) {
              const slice = sliceFromJson(editor.state.schema, internal.json)
              editor.view.dispatch(editor.view.state.tr.replaceSelection(slice))
              onClose()
              return
            }
          }
          const text = await navigator.clipboard.readText()
          editor.chain().focus().insertContent(text).run()
          onClose()
        } catch (err) {
          console.error('Clipboard read failed:', err)
          setClipboardError('剪贴板访问被拒绝')
        }
      },
    },
    {
      label: '全选',
      shortcut: 'Ctrl+A',
      icon: MousePointerClick,
      disabled: false,
      action: () => {
        editor.chain().focus().selectAll().run()
        onClose()
      },
    },
  ]

  // Group separators: after index 1 (undo/redo), after index 4 (cut/copy/paste)
  const separatorAfter = new Set([1, 4])

  return (
    <div
      ref={menuRef}
      role="menu"
      className="fixed z-40 min-w-[190px] overflow-hidden rounded-xl border border-border bg-popover/95 py-1 text-popover-foreground shadow-[0_18px_54px_rgba(2,8,23,0.22)] backdrop-blur-xl"
      style={{ left: position.x, top: position.y }}
    >
      {clipboardError && (
        <div className="border-b border-border px-3 py-1.5 text-xs text-destructive">
          {clipboardError}
        </div>
      )}
      {items.map((item, index) => (
        <div key={item.label}>
          <button
            role="menuitem"
            aria-disabled={item.disabled}
            disabled={item.disabled}
            className={`w-full flex items-center gap-3 px-3 min-h-9 text-sm ${
              item.disabled
                ? 'opacity-50 cursor-not-allowed'
                : 'hover:bg-surface-hover hover:text-foreground'
            }`}
            onClick={() => {
              if (!item.disabled) {
                item.action()
              }
            }}
            tabIndex={item.disabled ? -1 : 0}
          >
            <item.icon className="w-4 h-4 text-muted-foreground shrink-0" />
            <span className="flex-1 text-left">{item.label}</span>
            <span className="text-xs text-muted-foreground ml-auto">{item.shortcut}</span>
          </button>
          {separatorAfter.has(index) && <Separator />}
        </div>
      ))}
    </div>
  )
}
