import { describe, it, expect } from 'vitest'
import { appendBreakBefore, removeBreakBefore, resolveToBlockPos, promotePastList } from '@/components/editor/extensions/smart-split/styleUtils'
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

describe('promotePastList', () => {
  const schema = new Schema({
    nodes: {
      doc: { content: 'block+' },
      paragraph: {
        content: 'inline*',
        group: 'block',
        attrs: { style: { default: null } },
        toDOM: () => ['p', 0],
      },
      bulletList: {
        content: 'listItem+',
        group: 'block',
        attrs: { style: { default: null } },
        toDOM: () => ['ul', 0],
      },
      orderedList: {
        content: 'listItem+',
        group: 'block',
        attrs: { style: { default: null } },
        toDOM: () => ['ol', 0],
      },
      listItem: {
        content: 'paragraph+',
        attrs: { style: { default: null } },
        toDOM: () => ['li', 0],
      },
      text: { inline: true, group: 'inline' },
    },
  })

  function makeListDoc(listType: 'bulletList' | 'orderedList', texts: string[]): PmNode {
    return schema.nodeFromJSON({
      type: 'doc',
      content: [{
        type: listType,
        content: texts.map(text => ({
          type: 'listItem',
          content: [{
            type: 'paragraph',
            content: [{ type: 'text', text }],
          }],
        })),
      }],
    })
  }

  it('returns original position for paragraph not inside a list', () => {
    const doc = schema.nodeFromJSON({
      type: 'doc',
      content: [{
        type: 'paragraph',
        content: [{ type: 'text', text: 'plain' }],
      }],
    })
    // For a flat paragraph at doc start, resolveToBlockPos returns 0
    // promotePastList stays at 0 (no list ancestor)
    expect(promotePastList(doc, 0)).toBe(0)
  })

  it('promotes paragraph inside bulletList to listItem start', () => {
    const doc = makeListDoc('bulletList', ['first', 'second'])
    // 0: bulletList
    //   1: listItem
    //     2: paragraph
    //       3-7: text "first" (5 chars)
    //   ...
    //   10: listItem
    //     11: paragraph
    //       12-17: text "second" (6 chars)

    // resolveToBlockPos for second paragraph → 11
    // promotePastList promotes to enclosing listItem's before() = 10
    const result = promotePastList(doc, 11)
    expect(result).toBe(10)
    expect(doc.nodeAt(result)!.type.name).toBe('listItem')
  })

  it('promotes paragraph inside orderedList to listItem start', () => {
    const doc = makeListDoc('orderedList', ['one'])
    // 0: orderedList
    //   1: listItem
    //     2: paragraph
    //       3-5: text "one" (3 chars)
    const result = promotePastList(doc, 2)
    expect(result).toBe(1)
    expect(doc.nodeAt(result)!.type.name).toBe('listItem')
  })
})
