import { http } from 'msw'
import { sampleDraft } from '../fixtures'

export const draftHandlers = [
  // GET /api/v1/drafts/:draftId
  http.get('/api/v1/drafts/:draftId', async ({ params }) => {
    const { draftId } = params

    // Simulate 200ms delay
    await new Promise((resolve) => setTimeout(resolve, 200))

    // Return sample draft
    return Response.json({
      code: 0,
      data: {
        ...sampleDraft,
        id: Number(draftId),
      },
      message: 'ok',
    })
  }),

  // PUT /api/v1/drafts/:draftId
  http.put('/api/v1/drafts/:draftId', async ({ params, request }) => {
    const { draftId } = params
    // Parse body but don't use it (mock server)
    await request.json()

    // Simulate 100ms delay
    await new Promise((resolve) => setTimeout(resolve, 100))

    // Return updated draft (contract: only id and updated_at)
    return Response.json({
      code: 0,
      data: {
        id: Number(draftId),
        updated_at: new Date().toISOString(),
      },
      message: 'ok',
    })
  }),
]
