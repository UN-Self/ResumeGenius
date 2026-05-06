import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Deletion, Insertion } from '@/components/editor/extensions/ai-diff'

function createEditor(content: string) {
  return new Editor({
    extensions: [
      StarterKit.configure({ strike: false }),
      Deletion,
      Insertion,
    ],
    content,
  })
}

describe('ai-diff extension', () => {
  it('parses <del> as deletion mark', () => {
    const editor = createEditor('<p>hello <del>world</del> there</p>')
    const json = editor.getJSON()
    const para = json.content[0] as any
    const hasDel = para.content?.some(
      (n: any) => n.marks?.some((m: any) => m.type === 'deletion'),
    )
    expect(hasDel).toBe(true)
  })

  it('parses <ins> as insertion mark', () => {
    const editor = createEditor('<p>hello <ins>world</ins> there</p>')
    const json = editor.getJSON()
    const para = json.content[0] as any
    const hasIns = para.content?.some(
      (n: any) => n.marks?.some((m: any) => m.type === 'insertion'),
    )
    expect(hasIns).toBe(true)
  })

  it('serializes deletion back to <del>', () => {
    const editor = createEditor('<p><del>removed</del></p>')
    expect(editor.getHTML()).toContain('<del>')
  })

  it('serializes insertion back to <ins>', () => {
    const editor = createEditor('<p><ins>added</ins></p>')
    expect(editor.getHTML()).toContain('<ins>')
  })
})
