import { useState, useEffect } from 'react'
import { Bold, Italic, Underline, List, ListOrdered } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { ToolbarButton } from './ToolbarButton'
import { FontSelector } from './FontSelector'
import { FontSizeSelector } from './FontSizeSelector'
import { ColorPicker } from './ColorPicker'
import { LineHeightSelector } from './LineHeightSelector'
import { AlignSelector } from './AlignSelector'

interface ActiveStates {
  isBold: boolean
  isItalic: boolean
  isUnderline: boolean
  isBulletList: boolean
  isOrderedList: boolean
}

function getActiveStates(editor: Editor): ActiveStates {
  return {
    isBold: editor.isActive('bold'),
    isItalic: editor.isActive('italic'),
    isUnderline: editor.isActive('underline'),
    isBulletList: editor.isActive('bulletList'),
    isOrderedList: editor.isActive('orderedList'),
  }
}

interface BubbleToolbarProps {
  editor: Editor
}

export function BubbleToolbar({ editor }: BubbleToolbarProps) {
  const [activeStates, setActiveStates] = useState<ActiveStates>(() =>
    getActiveStates(editor)
  )

  useEffect(() => {
    const update = () => setActiveStates(getActiveStates(editor))
    editor.on('transaction', update)
    return () => { editor.off('transaction', update) }
  }, [editor])

  return (
    <div className="flex items-center gap-2 bg-white border border-border rounded-lg shadow-sm px-1 py-0.5">
      {/* Font & Size group */}
      <div role="group" aria-label="字体和字号" className="flex items-center gap-0.5">
        <FontSelector editor={editor} />
        <FontSizeSelector editor={editor} />
      </div>

      {/* Text Format + Color group */}
      <div role="group" aria-label="文本格式" className="flex items-center gap-0.5">
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleBold().run()}
          isActive={activeStates.isBold}
          icon={<Bold size={16} />}
          label="粗体 (Ctrl+B)"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleItalic().run()}
          isActive={activeStates.isItalic}
          icon={<Italic size={16} />}
          label="斜体 (Ctrl+I)"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleUnderline().run()}
          isActive={activeStates.isUnderline}
          icon={<Underline size={16} />}
          label="下划线 (Ctrl+U)"
        />
        <ColorPicker editor={editor} />
      </div>

      {/* List group */}
      <div role="group" aria-label="列表格式" className="flex items-center gap-0.5">
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          isActive={activeStates.isBulletList}
          icon={<List size={16} />}
          label="无序列表 (Ctrl+Shift+8)"
        />
        <ToolbarButton
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
          isActive={activeStates.isOrderedList}
          icon={<ListOrdered size={16} />}
          label="有序列表 (Ctrl+Shift+7)"
        />
      </div>

      {/* Alignment + Line Height group */}
      <div role="group" aria-label="行距和对齐" className="flex items-center gap-0.5">
        <AlignSelector editor={editor} />
        <LineHeightSelector editor={editor} />
      </div>
    </div>
  )
}
