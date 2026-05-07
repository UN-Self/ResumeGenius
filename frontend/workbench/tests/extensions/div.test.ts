import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Div } from '@/components/editor/extensions/Div'
import { PresetAttributes } from '@/components/editor/extensions/PresetAttributes'

function createEditor(content = '') {
  return new Editor({
    extensions: [StarterKit.configure({ strike: false }), Div, PresetAttributes],
    content,
  })
}

describe('Div extension', () => {
  it('preserves div with class attribute', () => {
    const editor = createEditor('<div class="resume">Hello</div>')
    const html = editor.getHTML()
    expect(html).toContain('<div')
    expect(html).toContain('class="resume"')
    editor.destroy()
  })

  it('preserves div with style attribute', () => {
    const editor = createEditor('<div style="color: red">Hello</div>')
    const html = editor.getHTML()
    expect(html).toContain('<div')
    expect(html).toContain('style="color: red;"')
    editor.destroy()
  })

  it('preserves nested divs', () => {
    const editor = createEditor(
      '<div class="outer"><div class="inner">Text</div></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="outer"')
    expect(html).toContain('class="inner"')
    editor.destroy()
  })

  it('wraps bare text in paragraph inside div', () => {
    const editor = createEditor('<div class="box">Plain text</div>')
    const html = editor.getHTML()
    expect(html).toContain('<p>Plain text</p>')
    editor.destroy()
  })

  it('preserves section tag as section (not div)', () => {
    const editor = createEditor('<section class="section"><p>Content</p></section>')
    const html = editor.getHTML()
    expect(html).toContain('<section')
    expect(html).toContain('class="section"')
    editor.destroy()
  })

  it('preserves header tag as header', () => {
    const editor = createEditor('<header class="profile"><p>Name</p></header>')
    const html = editor.getHTML()
    expect(html).toContain('<header')
    expect(html).toContain('class="profile"')
    editor.destroy()
  })

  it('handles div containing a list', () => {
    const editor = createEditor(
      '<div class="item"><ul><li>One</li><li>Two</li></ul></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="item"')
    expect(html).toContain('<li><p>One</p></li>')
    expect(html).toContain('<li><p>Two</p></li>')
    editor.destroy()
  })

  it('handles div containing headings', () => {
    const editor = createEditor(
      '<div class="section"><h2>Title</h2></div>',
    )
    const html = editor.getHTML()
    expect(html).toContain('class="section"')
    expect(html).toContain('<h2>Title</h2>')
    editor.destroy()
  })
})
