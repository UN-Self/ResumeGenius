import { type Editor } from '@tiptap/react'
import { Undo2, Redo2, Scissors, Copy, ClipboardPaste, CheckSquare } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

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
        const plainText = editor.view.state.doc.textBetween(from, to, '\n')
        navigator.clipboard.writeText(plainText)
        editor.chain().focus().deleteSelection().run()
        onClose()
      },
    },
    {
      label: '复制',
      shortcut: 'Ctrl+C',
      icon: Copy,
      disabled: !hasSelection,
      action: () => {
        const plainText = editor.view.state.doc.textBetween(from, to, '\n')
        navigator.clipboard.writeText(plainText)
        onClose()
      },
    },
    {
      label: '粘贴',
      shortcut: 'Ctrl+V',
      icon: ClipboardPaste,
      disabled: false,
      action: () => {
        navigator.clipboard.readText().then((text) => {
          editor.chain().focus().insertContent(text).run()
        })
        onClose()
      },
    },
    {
      label: '全选',
      shortcut: 'Ctrl+A',
      icon: CheckSquare,
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
      role="menu"
      className="fixed z-20 min-w-[180px] bg-white border border-border rounded-lg shadow-sm py-1"
      style={{ left: x, top: y }}
    >
      {items.map((item, index) => (
        <div key={item.label}>
          <button
            role="menuitem"
            aria-disabled={item.disabled}
            disabled={item.disabled}
            className={`w-full flex items-center gap-3 px-3 min-h-9 text-sm ${
              item.disabled
                ? 'opacity-50 cursor-not-allowed'
                : 'hover:bg-surface-hover'
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
