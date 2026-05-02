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
        // Text reply before HTML
        controller.enqueue(encoder.encode('data: {"type":"text","content":"好的，我来帮你优化简历。"}\n\n'))
        // ReAct: thinking
        controller.enqueue(encoder.encode('data: {"type":"thinking","content":"用户希望优化简历，需要先查看当前草稿内容。"}\n\n'))
        // ReAct: tool_call
        controller.enqueue(encoder.encode('data: {"type":"tool_call","name":"get_draft","params":{"draft_id":1}}\n\n'))
        // ReAct: tool_result
        controller.enqueue(encoder.encode('data: {"type":"tool_result","name":"get_draft","status":"completed"}\n\n'))
        // HTML chunk 1: START marker + first part (no END — tests cross-chunk parsing)
        controller.enqueue(encoder.encode('data: {"type":"text","content":"\\n<!--RESUME_HTML_START-->\\n<html><body><h1>Mock"}\n\n'))
        // HTML chunk 2: middle part (still no END)
        controller.enqueue(encoder.encode('data: {"type":"text","content":"优化简历</h1><p>工作经验：Mock公司前端开发</p>"}\n\n'))
        // HTML chunk 3: last part + END marker
        controller.enqueue(encoder.encode('data: {"type":"text","content":"</body></html>\\n<!--RESUME_HTML_END-->\\n"}\n\n'))
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
            content: '好的，我来帮你优化简历。<!--RESUME_HTML_START--><html><body>Mock优化简历</body></html><!--RESUME_HTML_END-->',
            thinking: '用户希望优化简历，需要先查看当前草稿内容。',
            tool_call: { id: 1, tool_name: 'get_draft', params: { draft_id: 1 }, result: {}, status: 'completed' },
            created_at: new Date().toISOString() },
        ],
      },
    })
  }),
]
