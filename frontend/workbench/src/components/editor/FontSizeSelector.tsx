import { useState, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'

const SIZES = ['10pt', '12pt', '14pt', '16pt', '18pt', '24pt'] as const
type SizeType = (typeof SIZES)[number]

interface FontSizeSelectorProps {
  editor: Editor
}

export function FontSizeSelector({ editor }: FontSizeSelectorProps) {
  const [currentSize, setCurrentSize] = useState<string>('12')
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    if (!editor) return

    const updateFontSize = () => {
      const fontSize = editor.getAttributes('textStyle').fontSize
      if (fontSize) {
        // Extract numeric value from fontSize (e.g., "14pt" -> "14")
        const match = fontSize.match(/(\d+)/)
        setCurrentSize(match ? match[1] : '12')
      } else {
        setCurrentSize('12')
      }
    }

    updateFontSize()
    editor.on('transaction', updateFontSize)
    return () => { editor.off('transaction', updateFontSize) }
  }, [editor])

  const handleSizeSelect = (size: SizeType) => {
    editor.chain().focus().setFontSize(size).run()
    setIsOpen(false)
  }

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="flex items-center gap-1 px-2 min-h-[44px] rounded-md text-sm text-muted-foreground hover:bg-surface-hover transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
        >
        <span>{currentSize}</span>
        <ChevronDown size={16} />
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" className="w-24 p-1">
        <div className="flex flex-col">
          {SIZES.map((size) => {
            const sizeNumeric = size.match(/(\d+)/)?.[1] || ''
            const isActive = sizeNumeric === currentSize
            return (
              <button
                key={size}
                onClick={() => handleSizeSelect(size)}
                className={`px-3 py-2 text-sm text-left rounded-md transition-colors ${
                  isActive
                    ? 'bg-primary-50 text-primary'
                    : 'text-muted-foreground hover:bg-surface-hover'
                }`}
              >
                {size}
              </button>
            )
          })}
        </div>
      </PopoverContent>
    </Popover>
  )
}
