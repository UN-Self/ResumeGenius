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

  it('captures preEditDoc when transaction changes document', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    const mockDoc = { eq: vi.fn().mockReturnValue(true) } as any
    const tr = { getMeta: vi.fn().mockReturnValue({}), docChanged: true }
    const oldState = { doc: mockDoc }

    const result = plugin.spec.state!.apply(
      tr as any,
      { isOwnDispatch: false, preEditDoc: null } as any,
      oldState as any,
      {} as any,
    )

    expect(result.preEditDoc).toBe(mockDoc)
    expect(result.isOwnDispatch).toBe(false)
  })

  it('preserves preEditDoc when transaction does not change document', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    const savedDoc = { eq: vi.fn() } as any
    const tr = { getMeta: vi.fn().mockReturnValue({}), docChanged: false }

    const result = plugin.spec.state!.apply(
      tr as any,
      { isOwnDispatch: false, preEditDoc: savedDoc } as any,
      {} as any,
      {} as any,
    )

    expect(result.preEditDoc).toBe(savedDoc)
    expect(result.isOwnDispatch).toBe(false)
  })

  it('preserves preEditDoc through ownDispatch cycle', () => {
    const plugin = smartSplitPlugin(DEFAULT_OPTIONS)
    const savedDoc = { eq: vi.fn() } as any

    // First: a docChanged tr captures preEditDoc
    const captureTr = { getMeta: vi.fn().mockReturnValue({}), docChanged: true }
    const afterCapture = plugin.spec.state!.apply(
      captureTr as any,
      { isOwnDispatch: false, preEditDoc: null } as any,
      { doc: savedDoc } as any,
      {} as any,
    )
    expect(afterCapture.preEditDoc).toBe(savedDoc)

    // Then: an ownDispatch tr should preserve preEditDoc
    const ownTr = { getMeta: vi.fn().mockReturnValue({ ownDispatch: true }) }
    const afterOwn = plugin.spec.state!.apply(
      ownTr as any,
      afterCapture as any,
      {} as any,
      {} as any,
    )
    expect(afterOwn.isOwnDispatch).toBe(true)
    expect(afterOwn.preEditDoc).toBe(savedDoc)
  })

  it('ownDispatch with docChanged preserves preEditDoc', () => {
    // When programmatic undos carry ownDispatch, preEditDoc must survive
    // prosemirror-history being imported — the module graph should be clean.
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, debug: false })
    expect(plugin).toBeDefined()
    // The revert path must use ownDispatch meta, not undo().
    // We verify the state contract: ownDispatch transactions preserve preEditDoc.
    const savedDoc = { eq: vi.fn() } as any
    const revertTr = { getMeta: vi.fn().mockReturnValue({ ownDispatch: true }), docChanged: true }
    const result = plugin.spec.state!.apply(
      revertTr as any,
      { isOwnDispatch: false, preEditDoc: savedDoc } as any,
      {} as any, {} as any,
    )
    // Revert dispatches with ownDispatch:true → preEditDoc must survive
    expect(result.isOwnDispatch).toBe(true)
    expect(result.preEditDoc).toBe(savedDoc)
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
