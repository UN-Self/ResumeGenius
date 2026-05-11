import { describe, it, expect } from 'vitest'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import { Div } from '@/components/editor/extensions/Div'
import { PresetAttributes } from '@/components/editor/extensions/PresetAttributes'
import { buildSplitTransaction } from '@/components/editor/extensions/smart-split/splitTransaction'

function createEditor(content: string) {
  return new Editor({
    extensions: [StarterKit.configure({ strike: false }), Div, PresetAttributes],
    content,
  })
}

function findAllPositions(doc: any, predicate: (node: any) => boolean): number[] {
  const results: number[] = []
  doc.descendants((node: any, pos: number) => {
    // +1 to enter the node (same semantics as view.posAtDOM(el, 0))
    if (predicate(node)) results.push(pos + 1)
  })
  return results
}

describe('buildSplitTransaction', () => {
  it('splits parent into front + <p> + back with shared data-ss-parent', () => {
    const editor = createEditor(
      '<section class="skills">' +
        '<div class="skill-item"><p>A</p></div>' +
        '<div class="skill-item"><p>B</p></div>' +
        '<div class="skill-item"><p>C</p></div>' +
      '</section>',
    )

    const doc = editor.state.doc
    const positions = findAllPositions(doc, (n: any) => n.attrs.class === 'skill-item')
    expect(positions.length).toBe(3)

    const crossPos = positions[1] // skill-item B
    const tr = buildSplitTransaction(editor.state, crossPos, 'data-ss-parent')
    expect(tr).not.toBeNull()

    const newState = editor.state.apply(tr!)
    // doc: frontSection, <p>, backSection
    expect(newState.doc.childCount).toBe(3)

    const s1 = newState.doc.child(0)!
    const pad = newState.doc.child(1)!
    const s2 = newState.doc.child(2)!

    expect(s1.attrs.class).toBe('skills')
    expect(pad.type.name).toBe('paragraph')
    expect(s2.attrs.class).toBe('skills')
    expect(s1.attrs['data-ss-parent']).toBe(s2.attrs['data-ss-parent'])

    expect(s1.childCount).toBe(1) // A
    expect(s2.childCount).toBe(2) // B, C

    editor.destroy()
  })

  it('returns null for position at document boundary', () => {
    const editor = createEditor('<section><p>A</p></section>')
    const tr = buildSplitTransaction(editor.state, 0, 'data-ss-parent')
    expect(tr).toBeNull()
    editor.destroy()
  })

  it('preserves class and style on split containers', () => {
    const editor = createEditor(
      '<section class="skills" style="color:red">' +
        '<div class="item"><p>A</p></div>' +
        '<div class="item"><p>B</p></div>' +
      '</section>',
    )

    const doc = editor.state.doc
    const positions = findAllPositions(doc, (n: any) => n.attrs.class === 'item')
    const crossPos = positions[1]

    const tr = buildSplitTransaction(editor.state, crossPos, 'data-ss-parent')
    expect(tr).not.toBeNull()

    const newState = editor.state.apply(tr!)
    const s2 = newState.doc.child(2)!
    expect(s2.attrs.class).toBe('skills')
    expect(s2.attrs.style).toBe('color:red')
    editor.destroy()
  })

  it('reuses existing data-ss-parent on re-split', () => {
    const editor = createEditor(
      '<section class="sec" data-ss-parent="existing">' +
        '<div class="item"><p>A</p></div>' +
        '<div class="item"><p>B</p></div>' +
      '</section>',
    )

    const doc = editor.state.doc
    const positions = findAllPositions(doc, (n: any) => n.attrs.class === 'item')
    const crossPos = positions[1]

    const tr = buildSplitTransaction(editor.state, crossPos, 'data-ss-parent')
    expect(tr).not.toBeNull()

    const newState = editor.state.apply(tr!)
    const s1 = newState.doc.child(0)!
    const s2 = newState.doc.child(2)!
    expect(s1.attrs['data-ss-parent']).toBe('existing')
    expect(s2.attrs['data-ss-parent']).toBe('existing')
    editor.destroy()
  })

  it('returns null when crossing element is first child', () => {
    const editor = createEditor(
      '<section class="sec">' +
        '<div class="item"><p>A</p></div>' +
        '<div class="item"><p>B</p></div>' +
      '</section>',
    )

    const doc = editor.state.doc
    const positions = findAllPositions(doc, (n: any) => n.attrs.class === 'item')
    const crossPos = positions[0] // first child — can't split before it

    const tr = buildSplitTransaction(editor.state, crossPos, 'data-ss-parent')
    expect(tr).toBeNull()
    editor.destroy()
  })

  it('returns null when parent has only one child', () => {
    const editor = createEditor('<section><p>Only child</p></section>')
    const doc = editor.state.doc
    const positions = findAllPositions(doc, (n: any) => n.type.name === 'paragraph')
    const crossPos = positions[0]

    const tr = buildSplitTransaction(editor.state, crossPos, 'data-ss-parent')
    expect(tr).toBeNull()
    editor.destroy()
  })

  it('preserves ancestor wrapper and inserts <p> between split halves', () => {
    const editor = createEditor(
      '<div class="resume">' +
        '<section class="skills">' +
          '<div class="skill-item"><p>A</p></div>' +
          '<div class="skill-item"><p>B</p></div>' +
          '<div class="skill-item"><p>C</p></div>' +
        '</section>' +
      '</div>',
    )

    const doc = editor.state.doc
    const positions = findAllPositions(doc, (n: any) => n.attrs.class === 'skill-item')
    const crossPos = positions[1]

    const tr = buildSplitTransaction(editor.state, crossPos, 'data-ss-parent')
    expect(tr).not.toBeNull()

    const newState = editor.state.apply(tr!)

    expect(newState.doc.childCount).toBe(1)
    const resume = newState.doc.firstChild!
    expect(resume.attrs.class).toBe('resume')
    // resume: frontSection, <p>, backSection
    expect(resume.childCount).toBe(3)

    const s1 = resume.child(0)!
    const pad = resume.child(1)!
    const s2 = resume.child(2)!
    expect(s1.attrs.class).toBe('skills')
    expect(pad.type.name).toBe('paragraph')
    expect(s2.attrs.class).toBe('skills')
    expect(s1.attrs['data-ss-parent']).toBe(s2.attrs['data-ss-parent'])
    expect(s1.childCount).toBe(1)
    expect(s2.childCount).toBe(2)

    editor.destroy()
  })
})
