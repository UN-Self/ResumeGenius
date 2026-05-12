import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { smartSplitPlugin } from '@/components/editor/extensions/smart-split/SmartSplitPlugin'
import { DEFAULT_OPTIONS } from '@/components/editor/extensions/smart-split/types'

describe('smartSplitPlugin', () => {
  it('creates a ProseMirror plugin with key', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    expect(plugin).toBeDefined()
    expect(plugin.key).toBeTruthy()
  })

  it('has a PluginKey named smartSplit', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    expect(plugin.key).toBe('smartSplit$')
  })

  it('returns view spec with update and destroy methods', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    const viewSpec = plugin.spec.view
    expect(viewSpec).toBeDefined()
    expect(typeof viewSpec).toBe('function')
  })

  it('does not crash when view() is invoked', () => {
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, debounce: 0 })
    const mockEditorView = {
      dom: document.createElement('div'),
      state: {} as any,
      dispatch: () => {},
    }

    const viewObj = plugin.spec.view!(mockEditorView as any)
    expect(viewObj).toBeDefined()
    expect(typeof viewObj.update).toBe('function')
    expect(typeof viewObj.destroy).toBe('function')

    viewObj.update(mockEditorView as any)
    viewObj.destroy()
  })

  it('debounces detection calls', () => {
    vi.useFakeTimers()
    const dispatch = vi.fn()
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, debounce: 100 })
    const mockEditorView = {
      dom: document.createElement('div'),
      state: {} as any,
      dispatch,
    }

    const viewObj = plugin.spec.view!(mockEditorView as any)

    // Multiple rapid updates should only trigger one detection
    viewObj.update(mockEditorView as any)
    viewObj.update(mockEditorView as any)
    viewObj.update(mockEditorView as any)

    // Before debounce fires, dispatch should not have been called
    expect(dispatch).not.toHaveBeenCalled()

    // After debounce fires, no breakers means no dispatch
    vi.advanceTimersByTime(150)
    expect(dispatch).not.toHaveBeenCalled()

    viewObj.destroy()
    vi.useRealTimers()
  })

  it('skips detection after own dispatch (cascade prevention)', () => {
    vi.useFakeTimers()
    const dispatch = vi.fn()
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, debounce: 0, debug: true })
    const mockEditorView = {
      dom: document.createElement('div'),
      state: {
        doc: { resolve: vi.fn() },
      } as any,
      dispatch,
    }

    const viewObj = plugin.spec.view!(mockEditorView as any)

    // Simulate own dispatch: apply(tr, value, oldState, newState)
    const ownDispatchTr = { getMeta: vi.fn().mockReturnValue({ ownDispatch: true }) }
    const result = plugin.spec.state!.apply(
      ownDispatchTr as any,
      { isOwnDispatch: false } as any,
      {} as any, {} as any,
    )
    expect(result.isOwnDispatch).toBe(true)

    viewObj.destroy()
    vi.useRealTimers()
  })

  it('clears pending timer on destroy', () => {
    vi.useFakeTimers()
    const dispatch = vi.fn()
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, debounce: 200 })
    const mockEditorView = {
      dom: document.createElement('div'),
      state: {} as any,
      dispatch,
    }

    const viewObj = plugin.spec.view!(mockEditorView as any)
    viewObj.update(mockEditorView as any)
    viewObj.destroy()

    // After destroy, timer should be cleared — advancing time should not cause issues
    vi.advanceTimersByTime(300)
    expect(dispatch).not.toHaveBeenCalled()

    vi.useRealTimers()
  })

  it('maintains isOwnDispatch false on non-own-dispatch doc change', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    const tr = { getMeta: vi.fn().mockReturnValue({}), docChanged: true }

    const result = plugin.spec.state!.apply(
      tr as any,
      { isOwnDispatch: false } as any,
      {} as any,
      {} as any,
    )

    expect(result.isOwnDispatch).toBe(false)
  })

  it('maintains isOwnDispatch false when transaction does not change document', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    const tr = { getMeta: vi.fn().mockReturnValue({}), docChanged: false }

    const result = plugin.spec.state!.apply(
      tr as any,
      { isOwnDispatch: false } as any,
      {} as any,
      {} as any,
    )

    expect(result.isOwnDispatch).toBe(false)
  })

  it('sets isOwnDispatch true when ownDispatch meta is set', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)

    const ownTr = { getMeta: vi.fn().mockReturnValue({ ownDispatch: true }) }
    const result = plugin.spec.state!.apply(
      ownTr as any,
      { isOwnDispatch: false } as any,
      {} as any,
      {} as any,
    )
    expect(result.isOwnDispatch).toBe(true)
  })

  it('reflects current tr ownDispatch (not previous value)', () => {
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, debug: false })
    expect(plugin).toBeDefined()

    const plainTr = { getMeta: vi.fn().mockReturnValue({}), docChanged: true }
    const result = plugin.spec.state!.apply(
      plainTr as any,
      { isOwnDispatch: true } as any,
      {} as any, {} as any,
    )
    expect(result.isOwnDispatch).toBe(false)
  })

  it('accepts insertPageBreaks option', () => {
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, insertPageBreaks: true })
    expect(plugin).toBeDefined()
  })

  it('accepts insertPageBreaks: false option', () => {
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, insertPageBreaks: false })
    expect(plugin).toBeDefined()
  })
})
