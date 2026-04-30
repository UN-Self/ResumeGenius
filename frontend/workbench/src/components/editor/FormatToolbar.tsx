import { useState, useEffect } from 'react'
import { Bold, Italic, Underline, List, ListOrdered, AlignLeft, AlignCenter, AlignRight, AlignJustify } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { ToolbarButton } from './ToolbarButton'
import { ToolbarSeparator } from './ToolbarSeparator'
import { FontSelector } from './FontSelector'
import { FontSizeSelector } from './FontSizeSelector'
import { ColorPicker } from './ColorPicker'
import { LineHeightSelector } from './LineHeightSelector'

interface ActiveStates {
  isBold: boolean
  isItalic: boolean
  isUnderline: boolean
  isBulletList: boolean
  isOrderedList: boolean
  isAlignLeft: boolean
  isAlignCenter: boolean
  isAlignRight: boolean
  isAlignJustify: boolean
}

function getActiveStates(editor: Editor): ActiveStates {
  return {
    isBold: editor.isActive('bold'),
    isItalic: editor.isActive('italic'),
    isUnderline: editor.isActive('underline'),
    isBulletList: editor.isActive('bulletList'),
    isOrderedList: editor.isActive('orderedList'),
    isAlignLeft: editor.isActive({ textAlign: 'left' }),
    isAlignCenter: editor.isActive({ textAlign: 'center' }),
    isAlignRight: editor.isActive({ textAlign: 'right' }),
    isAlignJustify: editor.isActive({ textAlign: 'justify' }),
  }
}

interface FormatToolbarProps {
  editor: Editor | null
}

export function FormatToolbar({ editor }: FormatToolbarProps) {
  const [activeStates, setActiveStates] = useState<ActiveStates | null>(null)

  useEffect(() => {
    if (!editor) return

    const update = () => setActiveStates(getActiveStates(editor))
    update()
    editor.on('transaction', update)
    return () => { editor.off('transaction', update) }
  }, [editor])

  if (!editor || !activeStates) return null

  return (
    <div className="flex items-center gap-1">
      {/* Font & Size group */}
      <div role="group" aria-label="字体和字号" className="flex items-center gap-1">
        <FontSelector editor={editor} />
        <FontSizeSelector editor={editor} />
      </div>

      <ToolbarSeparator />

      {/* Text Format + Color group */}
      <div role="group" aria-label="文本格式" className="flex items-center gap-1">
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleBold().run()}
          isActive={activeStates.isBold}
          icon={<Bold size={20} />}
          label="粗体 (Ctrl+B)"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleItalic().run()}
          isActive={activeStates.isItalic}
          icon={<Italic size={20} />}
          label="斜体 (Ctrl+I)"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleUnderline().run()}
          isActive={activeStates.isUnderline}
          icon={<Underline size={20} />}
          label="下划线 (Ctrl+U)"
        />
        <ColorPicker editor={editor} />
      </div>

      <ToolbarSeparator />

      {/* List group */}
      <div role="group" aria-label="列表格式" className="flex items-center gap-1">
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          isActive={activeStates.isBulletList}
          icon={<List size={20} />}
          label="无序列表 (Ctrl+Shift+8)"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
          isActive={activeStates.isOrderedList}
          icon={<ListOrdered size={20} />}
          label="有序列表 (Ctrl+Shift+7)"
        />
      </div>

      <ToolbarSeparator />

      {/* Line Height + Alignment group */}
      <div role="group" aria-label="行距和对齐" className="flex items-center gap-1">
        <LineHeightSelector editor={editor} />
        <ToolbarButton
          onClick={() => editor.chain().focus().setTextAlign('left').run()}
          isActive={activeStates.isAlignLeft}
          icon={<AlignLeft size={20} />}
          label="左对齐"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().setTextAlign('center').run()}
          isActive={activeStates.isAlignCenter}
          icon={<AlignCenter size={20} />}
          label="居中"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().setTextAlign('right').run()}
          isActive={activeStates.isAlignRight}
          icon={<AlignRight size={20} />}
          label="右对齐"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().setTextAlign('justify').run()}
          isActive={activeStates.isAlignJustify}
          icon={<AlignJustify size={20} />}
          label="两端对齐"
        />
      </div>
    </div>
  )
}
