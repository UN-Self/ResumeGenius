import { http } from 'msw'

export const agentHandlers = [
  http.get('/api/v1/ai/sessions', () => {
    return Response.json({ code: 0, data: [] })
  }),

  http.post('/api/v1/ai/sessions', async ({ request }) => {
    const body = (await request.json()) as { draft_id?: number }
    return Response.json({
      code: 0,
      data: {
        id: 1,
        draft_id: body.draft_id ?? 1,
        created_at: new Date().toISOString(),
      },
    })
  }),

  http.get('/api/v1/ai/sessions/:sessionId', ({ params }) => {
    return Response.json({
      code: 0,
      data: { id: Number(params.sessionId), draft_id: 1, created_at: new Date().toISOString() },
    })
  }),

  http.delete('/api/v1/ai/sessions/:sessionId', () => {
    return Response.json({ code: 0, data: null, message: 'ok' })
  }),

  http.post('/api/v1/ai/sessions/:sessionId/chat', () => {
    const encoder = new TextEncoder()
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('data: {"type":"text","content":"好的，我来帮你优化简历。"}\n\n'))
        controller.enqueue(encoder.encode('data: {"type":"text","content":"\\n<!--RESUME_HTML_START-->\\n<html><body><h1>Mock优化简历</h1></body></html>\\n<!--RESUME_HTML_END-->\\n"}\n\n'))
        controller.enqueue(encoder.encode('data: {"type":"done"}\n\n'))
        controller.close()
      },
    })
    return new Response(stream, {
      headers: { 'Content-Type': 'text/event-stream' },
    })
  }),

  http.get('/api/v1/ai/sessions/:sessionId/history', ({ params }) => {
    return Response.json({
      code: 0,
      data: {
        items: [
          { id: 1, session_id: Number(params.sessionId), role: 'user', content: '优化简历', created_at: new Date().toISOString() },
          { id: 2, session_id: Number(params.sessionId), role: 'assistant', content: '好的，我来帮你优化简历。<!--RESUME_HTML_START--><html><body>Mock</body></html><!--RESUME_HTML_END-->', created_at: new Date().toISOString() },
        ],
      },
    })
  }),
]
