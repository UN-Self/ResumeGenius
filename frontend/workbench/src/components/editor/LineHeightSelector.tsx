import { useState, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'

const LINE_HEIGHTS = ['1.0', '1.15', '1.5', '1.75', '2.0', '2.5'] as const

interface LineHeightSelectorProps {
  editor: Editor
}

export function LineHeightSelector({ editor }: LineHeightSelectorProps) {
  const [currentLineHeight, setCurrentLineHeight] = useState<string | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    if (!editor) return

    const updateLineHeight = () => {
      const attributes = editor.getAttributes('textStyle')
      setCurrentLineHeight(attributes.lineHeight || null)
    }

    updateLineHeight()
    editor.on('transaction', updateLineHeight)

    return () => {
      editor.off('transaction', updateLineHeight)
    }
  }, [editor])

  const handleLineHeightChange = (value: string) => {
    editor.chain().focus().setLineHeight(value).run()
    setIsOpen(false)
  }

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="行高"
          className="flex items-center gap-1 px-2 min-h-[44px] rounded-md text-sm text-muted-foreground hover:bg-surface-hover transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
        >
        <span>{currentLineHeight || '—'}</span>
        <ChevronDown size={16} />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-24 p-1" side="top">
        <div className="flex flex-col">
          {LINE_HEIGHTS.map((height) => (
            <button
              key={height}
              onClick={() => handleLineHeightChange(height)}
              className={`px-2 py-1 text-sm rounded hover:bg-surface-hover transition-colors ${
                currentLineHeight === height
                  ? 'bg-primary-50 text-primary'
                  : 'text-muted-foreground'
              }`}
            >
              {height}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
