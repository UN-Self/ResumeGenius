import '@testing-library/jest-dom/vitest'

// jsdom does not implement ResizeObserver
Object.defineProperty(global, 'ResizeObserver', {
  value: class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
})
