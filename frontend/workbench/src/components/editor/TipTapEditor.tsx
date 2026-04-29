import { EditorContent, type Editor } from '@tiptap/react'
import '@/styles/editor.css'

interface TipTapEditorProps {
  editor: Editor
}

export function TipTapEditor({ editor }: TipTapEditorProps) {
  return <EditorContent editor={editor} />
}
