import { describe, it, expect } from 'vitest'
import { appendBreakBefore, removeBreakBefore } from '@/components/editor/extensions/smart-split/styleUtils'

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
