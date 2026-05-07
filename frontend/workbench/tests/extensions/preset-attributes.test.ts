import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { PresetAttributes } from '@/components/editor/extensions/PresetAttributes'

function createEditor(content = '') {
  return new Editor({
    extensions: [StarterKit.configure({ strike: false }), PresetAttributes],
    content,
  })
}

describe('PresetAttributes', () => {
  it('preserves class attribute on paragraph', () => {
    const editor = createEditor('<p class="intro">Hello</p>')
    expect(editor.getHTML()).toContain('class="intro"')
    editor.destroy()
  })

  it('preserves class attribute on heading', () => {
    const editor = createEditor('<h2 class="section-title">Title</h2>')
    expect(editor.getHTML()).toContain('class="section-title"')
    editor.destroy()
  })

  it('preserves class attribute on list item', () => {
    const editor = createEditor('<ul><li class="item">Item</li></ul>')
    expect(editor.getHTML()).toContain('class="item"')
    editor.destroy()
  })

  it('preserves style attribute on paragraph', () => {
    const editor = createEditor('<p style="color: red">Red text</p>')
    expect(editor.getHTML()).toContain('style="color: red;"')
    editor.destroy()
  })

  it('does not add attributes when none present', () => {
    const editor = createEditor('<p>Plain text</p>')
    expect(editor.getHTML()).toBe('<p>Plain text</p>')
    editor.destroy()
  })

  it('preserves both class and style on the same element', () => {
    const editor = createEditor('<p class="highlight" style="font-size: 14px">Text</p>')
    expect(editor.getHTML()).toContain('class="highlight"')
    expect(editor.getHTML()).toContain('style="font-size: 14px;"')
    editor.destroy()
  })
})
