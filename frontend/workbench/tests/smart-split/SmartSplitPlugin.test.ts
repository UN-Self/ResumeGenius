import { describe, it, expect } from 'vitest'
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

  it('runs detection on update without requiring focus', () => {
    const plugin = smartSplitPlugin({ ...DEFAULT_OPTIONS, debounce: 0 })
    const mockEditorView = {
      dom: document.createElement('div'),
      state: {} as any,
      dispatch: () => {},
    }

    const viewObj = plugin.spec.view!(mockEditorView as any)
    // Should not throw even without focus
    expect(() => viewObj.update(mockEditorView as any)).not.toThrow()
    viewObj.destroy()
  })
})
