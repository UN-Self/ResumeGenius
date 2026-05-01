import '@testing-library/jest-dom'
import { beforeAll, afterEach, afterAll } from 'vitest'
import { setupServer } from 'msw/node'
import { draftHandlers } from '@/mocks/handlers/drafts'
import { projectHandlers } from '@/mocks/handlers/projects'
import { agentHandlers } from '@/mocks/handlers/agent'

// Setup MSW server for tests
export const server = setupServer(...projectHandlers, ...draftHandlers, ...agentHandlers)

// Start server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))

// Reset handlers after each test
afterEach(() => server.resetHandlers())

// Close server after all tests
afterAll(() => server.close())
