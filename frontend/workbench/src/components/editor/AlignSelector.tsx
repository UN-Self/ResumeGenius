import { useState, useEffect } from 'react'
import { AlignLeft, AlignCenter, AlignRight, AlignJustify, ChevronDown } from 'lucide-react'
import { type Editor } from '@tiptap/react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { DropdownTrigger, DropdownItem } from '@/components/ui/dropdown'

const ALIGNMENTS = [
  { label: '左对齐', value: 'left', Icon: AlignLeft },
  { label: '居中', value: 'center', Icon: AlignCenter },
  { label: '右对齐', value: 'right', Icon: AlignRight },
  { label: '两端对齐', value: 'justify', Icon: AlignJustify },
] as const

const ICON_MAP: Record<string, typeof AlignLeft> = {
  left: AlignLeft,
  center: AlignCenter,
  right: AlignRight,
  justify: AlignJustify,
}

interface AlignSelectorProps {
  editor: Editor
}

export function AlignSelector({ editor }: AlignSelectorProps) {
  const [currentAlign, setCurrentAlign] = useState<string | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    if (!editor) return

    const updateAlign = () => {
      for (const { value } of ALIGNMENTS) {
        if (editor.isActive('textAlign', { textAlign: value })) {
          setCurrentAlign(value)
          return
        }
      }
      setCurrentAlign(null)
    }

    updateAlign()
    editor.on('transaction', updateAlign)
    return () => { editor.off('transaction', updateAlign) }
  }, [editor])

  const CurrentIcon = currentAlign ? ICON_MAP[currentAlign] ?? AlignLeft : AlignLeft

  const handleAlignSelect = (value: string) => {
    editor.chain().focus().setTextAlign(value).run()
    setIsOpen(false)
  }

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <DropdownTrigger aria-label="对齐方式">
          <CurrentIcon size={16} />
          <ChevronDown size={14} />
        </DropdownTrigger>
      </PopoverTrigger>
      <PopoverContent side="top" className="w-32 p-1">
        <div className="flex flex-col">
          {ALIGNMENTS.map((align) => (
            <DropdownItem
              key={align.value}
              active={currentAlign === align.value}
              onClick={() => handleAlignSelect(align.value)}
            >
              {align.label}
            </DropdownItem>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
