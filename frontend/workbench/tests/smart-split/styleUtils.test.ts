import { describe, it, expect } from 'vitest'
import { appendBreakBefore, removeBreakBefore, resolveToBlockPos } from '@/components/editor/extensions/smart-split/styleUtils'
import { Schema } from '@tiptap/pm/model'
import type { Node as PmNode } from '@tiptap/pm/model'

describe('appendBreakBefore', () => {
  it('appends break-before: page to empty style', () => {
    expect(appendBreakBefore('')).toBe('break-before: page')
  })

  it('appends to existing style', () => {
    expect(appendBreakBefore('color: red')).toBe('color: red; break-before: page')
  })

  it('does not duplicate if already present', () => {
    expect(appendBreakBefore('color: red; break-before: page')).toBe('color: red; break-before: page')
  })

  it('handles style with trailing semicolon', () => {
    expect(appendBreakBefore('color: red; ')).toBe('color: red; break-before: page')
  })
})

describe('removeBreakBefore', () => {
  it('removes break-before from style', () => {
    expect(removeBreakBefore('color: red; break-before: page')).toBe('color: red')
  })

  it('returns null when style becomes empty', () => {
    expect(removeBreakBefore('break-before: page')).toBeNull()
  })

  it('returns null for empty string', () => {
    expect(removeBreakBefore('')).toBeNull()
  })

  it('preserves other properties', () => {
    expect(removeBreakBefore('color: red; font-size: 14px; break-before: page')).toBe('color: red; font-size: 14px')
  })
})

describe('resolveToBlockPos', () => {
  const schema = new Schema({
    nodes: {
      doc: { content: 'block+' },
      paragraph: {
        content: 'inline*',
        group: 'block',
        attrs: { style: { default: null } },
        toDOM: () => ['p', 0],
      },
      text: { inline: true, group: 'inline' },
    },
  })

  function makeDoc(content: string[]): PmNode {
    return schema.nodeFromJSON({
      type: 'doc',
      content: content.map(text => ({
        type: 'paragraph',
        attrs: { style: null },
        content: [{ type: 'text', text }],
      })),
    })
  }

  it('resolves text position inside paragraph to paragraph node position', () => {
    const doc = makeDoc(['first', 'second'])
    // Position mapping for this doc:
    // 0: paragraph 1 (nodeSize=2+5=7)
    //   1-5: text "first"
    // 7: paragraph 2 (nodeSize=2+6=8)
    //   8-13: text "second"

    // posAtDOM(p2, 0) would return 8 (inside p2, at text "second")
    // We want to resolve to 7 (p2's own position)
    const innerPos = 8 // simulating posAtDOM(p2, 0)
    const resolved = resolveToBlockPos(doc, innerPos)
    expect(resolved).toBe(7)
    expect(doc.nodeAt(resolved)!.type.name).toBe('paragraph')
  })

  it('resolves position inside first paragraph', () => {
    const doc = makeDoc(['hello', 'world'])
    // 0: paragraph 1 (nodeSize=2+5=7)
    //   1-5: text "hello"
    // 7: paragraph 2 (nodeSize=2+5=7)
    //   8-12: text "world"

    const innerPos = 1 // posAtDOM(p1, 0)
    const resolved = resolveToBlockPos(doc, innerPos)
    expect(resolved).toBe(0)
    expect(doc.nodeAt(resolved)!.type.name).toBe('paragraph')
  })
})
