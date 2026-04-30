import { useState, useEffect } from 'react'
import { type Editor } from '@tiptap/react'
import { ChevronDown } from 'lucide-react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'

const FONTS = [
  { label: '默认字体', value: '' },
  { label: '宋体', value: 'SimSun, serif' },
  { label: '黑体', value: 'SimHei, sans-serif' },
  { label: '楷体', value: 'KaiTi, serif' },
  { label: '仿宋', value: 'FangSong, serif' },
  { label: 'Times New Roman', value: '"Times New Roman", serif' },
  { label: 'Arial', value: 'Arial, sans-serif' },
  { label: 'Georgia', value: 'Georgia, serif' },
] as const

interface FontSelectorProps {
  editor: Editor
}

export function FontSelector({ editor }: FontSelectorProps) {
  const [currentFont, setCurrentFont] = useState<string | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    const updateFont = () => {
      const fontFamily = editor.getAttributes('textStyle').fontFamily
      setCurrentFont(fontFamily || null)
    }

    updateFont()

    editor.on('transaction', updateFont)

    return () => {
      editor.off('transaction', updateFont)
    }
  }, [editor])

  const currentFontLabel = currentFont
    ? FONTS.find((f) => f.value === currentFont)?.label || '字体'
    : '字体'

  const handleFontSelect = (value: string) => {
    if (value === '') {
      editor.chain().focus().unsetFontFamily().run()
    } else {
      editor.chain().focus().setFontFamily(value).run()
    }
    setIsOpen(false)
  }

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="flex items-center gap-1 px-2 min-h-[44px] rounded-md text-sm text-muted-foreground hover:bg-surface-hover transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
        >
          <span>{currentFontLabel}</span>
          <ChevronDown className="w-4 h-4" />
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" className="w-48 p-1">
        <div className="flex flex-col">
          {FONTS.map((font) => (
            <button
              key={font.value}
              type="button"
              onClick={() => handleFontSelect(font.value)}
              className="px-3 py-2 text-sm text-left hover:bg-surface-hover rounded-md transition-colors"
              style={{ fontFamily: font.value || undefined }}
            >
              {font.label}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
