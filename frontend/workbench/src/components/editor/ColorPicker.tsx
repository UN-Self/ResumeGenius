import { useState, useEffect } from 'react'
import { Type, Highlighter } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { Popover, PopoverAnchor, PopoverContent } from '@/components/ui/popover'

const PRESET_COLORS = [
  '#000000', '#434343', '#666666', '#999999', '#b7b7b7', '#cccccc',
  '#d9d9d9', '#efefef', '#f3f3f3', '#ffffff',
  '#e06666', '#f6b26b', '#ffd966', '#93c47d', '#76a5af', '#6fa8dc',
  '#8e7cc3', '#c27ba0',
] as const

type ColorTarget = 'font' | 'background' | null

interface ColorPickerProps {
  editor: Editor
}

export function ColorPicker({ editor }: ColorPickerProps) {
  const [open, setOpen] = useState(false)
  const [target, setTarget] = useState<ColorTarget>(null)
  const [fontColor, setFontColor] = useState<string | null>(null)
  const [backgroundColor, setBackgroundColor] = useState<string | null>(null)

  useEffect(() => {
    if (!editor) return

    const updateColors = () => {
      const attrs = editor.getAttributes('textStyle')
      setFontColor(attrs.color || null)
      setBackgroundColor(attrs.backgroundColor || null)
    }

    updateColors()
    editor.on('transaction', updateColors)
    return () => { editor.off('transaction', updateColors) }
  }, [editor])

  const handleColorSelect = (color: string) => {
    if (target === 'font') {
      editor.chain().focus().setColor(color).run()
    } else if (target === 'background') {
      editor.chain().focus().setBackgroundColor(color).run()
    }
    setOpen(false)
  }

  const handleReset = () => {
    if (target === 'font') {
      editor.chain().focus().unsetColor().run()
    } else if (target === 'background') {
      editor.chain().focus().unsetBackgroundColor().run()
    }
    setOpen(false)
  }

  const handleFontClick = () => {
    setTarget('font')
    setOpen(true)
  }

  const handleHighlightClick = () => {
    setTarget('background')
    setOpen(true)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverAnchor>
      <div className="flex items-center gap-1">
        {/* Font color trigger */}
        <button
          type="button"
          onClick={handleFontClick}
          className="relative p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-md text-[#5f6368] hover:bg-[#f8f9fa] transition-colors"
          aria-label="字体颜色"
        >
          <Type size={20} />
          {fontColor && (
            <span className="absolute bottom-1 left-1/2 -translate-x-1/2 w-4 h-1 rounded-full" style={{ backgroundColor: fontColor }} />
          )}
        </button>

        {/* Highlight color trigger */}
        <button
          type="button"
          onClick={handleHighlightClick}
          className="relative p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-md text-[#5f6368] hover:bg-[#f8f9fa] transition-colors"
          aria-label="背景高亮"
        >
          <Highlighter size={20} />
          {backgroundColor && (
            <span className="absolute bottom-1 left-1/2 -translate-x-1/2 w-4 h-1 rounded-full" style={{ backgroundColor: backgroundColor }} />
          )}
        </button>
      </div>
      </PopoverAnchor>

      <PopoverContent side="top" align="start" sideOffset={4} className="w-auto p-3">
        <div className="grid grid-cols-9 gap-1 mb-3">
          {PRESET_COLORS.map((color) => (
            <button
              key={color}
              type="button"
              onClick={() => handleColorSelect(color)}
              className="w-6 h-6 rounded border border-[#dadce0] hover:scale-110 transition-transform"
              style={{ backgroundColor: color }}
              aria-label={`选择颜色 ${color}`}
            />
          ))}
        </div>

        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={handleReset}
            className="text-sm text-[#5f6368] hover:text-[#202124] transition-colors"
          >
            重置
          </button>

          <label className="flex items-center gap-2 text-sm text-[#5f6368] cursor-pointer">
            自定义
            <input
              type="color"
              onChange={(e) => handleColorSelect(e.target.value)}
              className="w-8 h-8 border-0 p-0 cursor-pointer"
            />
          </label>
        </div>
      </PopoverContent>
    </Popover>
  )
}
