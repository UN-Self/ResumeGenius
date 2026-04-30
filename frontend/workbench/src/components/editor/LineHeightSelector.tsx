import { useState, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { DropdownTrigger, DropdownItem } from '@/components/ui/dropdown'

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
        <DropdownTrigger aria-label="行距">
          <span>{currentLineHeight || '—'}</span>
          <ChevronDown size={16} />
        </DropdownTrigger>
      </PopoverTrigger>
      <PopoverContent className="w-24 p-1" side="top">
        <div className="flex flex-col">
          {LINE_HEIGHTS.map((height) => (
            <DropdownItem
              key={height}
              active={currentLineHeight === height}
              onClick={() => handleLineHeightChange(height)}
              className="px-2 py-1"
            >
              {height}
            </DropdownItem>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
