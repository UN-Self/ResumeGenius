import { setupWorker } from 'msw/browser'
import { draftHandlers } from './handlers/drafts'

// Export the worker instance for the main app to start
export const worker = setupWorker(...draftHandlers)

// Start the worker in development mode when needed
export async function startMockWorker() {
  if (import.meta.env.DEV && import.meta.env.VITE_USE_MOCK === 'true') {
    await worker.start({
      onUnhandledRequest: 'bypass',
    })
    console.log('[MSW] Mock worker started')
  }
}
