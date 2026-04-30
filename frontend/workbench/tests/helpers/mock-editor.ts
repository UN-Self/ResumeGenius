import { vi } from 'vitest'

export interface MockEditorOptions {
  /** Method names available via editor.chain().focus() — each returns { run: runMock } */
  chainCommands?: string[]
  /** Return value for getAttributes() (default: () => ({})) */
  getAttributes?: () => Record<string, any>
  /** isActive implementation (default: () => false) */
  isActive?: (name: string, attrs?: Record<string, unknown>) => boolean
}

/**
 * Creates a reusable mock TipTap editor for component tests.
 *
 * All 5 test files that previously had inline createMockEditor() now share this.
 * Test helpers (runMock, listeners, simulateTransaction) are attached to the editor object.
 */
export function createMockEditor(options: MockEditorOptions = {}) {
  const runMock = vi.fn()
  const listeners = new Map<string, Set<() => void>>()

  const focusMock = () => {
    const commands: Record<string, () => { run: typeof runMock }> = {}
    for (const cmd of options.chainCommands ?? []) {
      commands[cmd] = () => ({ run: runMock })
    }
    return commands
  }

  const editor = {
    chain: () => ({ focus: focusMock }),
    getAttributes: vi.fn(options.getAttributes ?? (() => ({}))),
    isActive: options.isActive ?? (() => false),
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn((event: string, cb: () => void) => {
      listeners.get(event)?.delete(cb)
    }),
  }

  // Attach test helpers for easy access
  ;(editor as any).runMock = runMock
  ;(editor as any).listeners = listeners
  ;(editor as any).simulateTransaction = () => {
    const callbacks = listeners.get('transaction')
    if (callbacks) {
      callbacks.forEach((cb) => cb())
    }
  }

  return editor as any
}
