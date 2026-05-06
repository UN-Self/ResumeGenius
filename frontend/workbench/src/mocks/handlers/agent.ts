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
        // Text reply
        controller.enqueue(encoder.encode('data: {"type":"text","content":"好的，我来帮你优化简历。"}\n\n'))
        // ReAct: thinking
        controller.enqueue(encoder.encode('data: {"type":"thinking","content":"用户希望优化简历，需要先查看当前草稿内容。"}\n\n'))
        // ReAct: tool_call
        controller.enqueue(encoder.encode('data: {"type":"tool_call","name":"get_draft","params":{"draft_id":1}}\n\n'))
        // ReAct: tool_result
        controller.enqueue(encoder.encode('data: {"type":"tool_result","name":"get_draft","status":"completed"}\n\n'))
        // Edit events (new format: inline diff ops)
        controller.enqueue(encoder.encode('data: {"type":"edit","params":{"ops":[{"old_string":"前端开发工程师","new_string":"高级前端开发工程师","description":"提升职位级别"}]}}\n\n'))
        // Final text
        controller.enqueue(encoder.encode('data: {"type":"text","content":"\\n已完成简历优化，主要调整了职位头衔。"}\n\n'))
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
          { id: 2, session_id: Number(params.sessionId), role: 'assistant',
            content: '好的，我来帮你优化简历。\n已完成简历优化，主要调整了职位头衔。',
            created_at: new Date().toISOString() },
        ],
      },
    })
  }),
]
