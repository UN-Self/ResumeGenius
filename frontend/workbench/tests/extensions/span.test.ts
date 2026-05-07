import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { Span } from '@/components/editor/extensions/Span'
import { Div } from '@/components/editor/extensions/Div'

function createEditor(content = '') {
  return new Editor({
    extensions: [StarterKit.configure({ strike: false }), TextStyleKit, Span, Div],
    content,
  })
}

describe('Span mark', () => {
  it('preserves span with class attribute', () => {
    const editor = createEditor('<p><span class="tag">TypeScript</span></p>')
    const html = editor.getHTML()
    expect(html).toContain('class="tag"')
    editor.destroy()
  })

  it('does not match span with style (left for TextStyle)', () => {
    const editor = createEditor('<p><span style="color: red">Red</span></p>')
    const html = editor.getHTML()
    // TextStyle handles this — should render as <span style="...">
    expect(html).toContain('style="color: red;"')
    editor.destroy()
  })

  it('leaves span with both class and style to TextStyle', () => {
    const editor = createEditor('<p><span class="highlight" style="color: red">Text</span></p>')
    const html = editor.getHTML()
    // TextStyle has higher priority and should handle this span
    expect(html).toContain('style="color: red;"')
    editor.destroy()
  })

  it('does not create span for bare span without class or style', () => {
    const editor = createEditor('<p><span>Bare</span></p>')
    const html = editor.getHTML()
    // No class and no style → neither Span nor TextStyle matches → plain text
    expect(html).not.toContain('<span')
    expect(html).toContain('Bare')
    editor.destroy()
  })

  it('preserves multiple spans with different classes', () => {
    const editor = createEditor(
      '<p><span class="tag">Go</span> and <span class="tag">React</span></p>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="tag"')
    editor.destroy()
  })

  it('preserves span class inside a div container', () => {
    const editor = createEditor(
      '<div><p><span class="label">Name:</span> Alice</p></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="label"')
    expect(html).toContain('Alice')
    editor.destroy()
  })
})
