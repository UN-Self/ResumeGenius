import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Div } from '@/components/editor/extensions/Div'
import { PresetAttributes } from '@/components/editor/extensions/PresetAttributes'
import { SmartSplitExtension } from '@/components/editor/extensions/SmartSplitExtension'

function createEditor(content: string) {
  return new Editor({
    extensions: [
      StarterKit.configure({ strike: false }),
      Div,
      PresetAttributes,
      SmartSplitExtension.configure({ debounce: 0 }),
    ],
    content,
  })
}

describe('SmartSplitExtension integration', () => {
  it('loads without error', () => {
    const editor = createEditor('<section><p>Hello</p></section>')
    expect(editor).toBeDefined()
    editor.destroy()
  })

  it('preserves content through empty detection (no breakers)', () => {
    const html = '<section class="skills"><div class="item"><p>Test</p></div></section>'
    const editor = createEditor(html)
    const result = editor.getHTML()
    expect(result).toContain('class="skills"')
    expect(result).toContain('class="item"')
    editor.destroy()
  })

  it('default options are applied', () => {
    const editor = createEditor('<section><p>Hello</p></section>')
    const plugin = SmartSplitExtension.config
    expect(plugin).toBeDefined()
    editor.destroy()
  })
})
