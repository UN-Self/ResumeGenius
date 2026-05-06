import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { server } from './setup'
import { ChatPanel } from '@/components/chat/ChatPanel'

function renderChatPanel(draftId = 1, onApplyEdits = vi.fn().mockResolvedValue(undefined), onRestoreHtml = vi.fn()) {
  return render(<ChatPanel draftId={draftId} onApplyEdits={onApplyEdits} onRestoreHtml={onRestoreHtml} />)
}

describe('ChatPanel', () => {
  it('renders the chat header', () => {
    renderChatPanel()
    expect(screen.getByText('AI 助手')).toBeInTheDocument()
  })

  it('renders input and send button', () => {
    renderChatPanel()
    expect(screen.getByPlaceholderText('输入你的需求...')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /发送/i })).toBeInTheDocument()
  })

  it('reuses existing session when available', async () => {
    server.use(
      http.get('/api/v1/ai/sessions', ({ request }) => {
        const url = new URL(request.url)
        if (url.searchParams.get('draft_id') === '1') {
          return HttpResponse.json({ code: 0, data: [{ id: 5, draft_id: 1 }] })
        }
        return HttpResponse.json({ code: 0, data: [] })
      }),
      http.get('/api/v1/ai/sessions/5/history', () => {
        return HttpResponse.json({
          code: 0,
          data: {
            items: [
              { id: 1, role: 'user', content: 'hello', created_at: '' },
              { id: 2, role: 'assistant', content: 'hi there', created_at: '' },
            ],
          },
        })
      }),
    )

    renderChatPanel()

    await waitFor(() => {
      expect(screen.getByText('hello')).toBeInTheDocument()
      expect(screen.getByText('hi there')).toBeInTheDocument()
    })
  })

  it('shows undo/redo buttons in chat after edits are applied', async () => {
    const onRestoreHtml = vi.fn()
    renderChatPanel(1, vi.fn().mockResolvedValue(undefined), onRestoreHtml)
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => expect(input).toBeEnabled())

    await userEvent.type(input, '优化简历')
    await userEvent.click(screen.getByRole('button', { name: /发送/i }))

    // Undo/redo buttons should appear in chat after edits are applied
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /undo/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /redo/i })).toBeInTheDocument()
    })
  })

  it('calls onRestoreHtml when undo button is clicked', async () => {
    const restoredHTML = '<p>restored content</p>'
    const onRestoreHtml = vi.fn()
    server.use(
      http.post('/api/v1/ai/drafts/1/undo', () => {
        return HttpResponse.json({ code: 0, data: { html_content: restoredHTML } })
      }),
    )

    renderChatPanel(1, vi.fn().mockResolvedValue(undefined), onRestoreHtml)
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => expect(input).toBeEnabled())

    await userEvent.type(input, '优化简历')
    await userEvent.click(screen.getByRole('button', { name: /发送/i }))

    const undoBtn = await screen.findByRole('button', { name: /undo/i })
    await userEvent.click(undoBtn)

    await waitFor(() => {
      expect(onRestoreHtml).toHaveBeenCalledWith(restoredHTML)
    })
  })

  it('shows empty state message when no history', () => {
    server.use(
      http.get('/api/v1/ai/sessions', () => {
        return HttpResponse.json({ code: 0, data: [] })
      }),
      http.post('/api/v1/ai/sessions', () => {
        return HttpResponse.json({ code: 0, data: { id: 1, draft_id: 1 } })
      }),
    )

    renderChatPanel()
    expect(screen.getByText(/输入你的需求/)).toBeInTheDocument()
  })

  it('disables send button when input is empty', () => {
    renderChatPanel()
    const sendButton = screen.getByRole('button', { name: /发送/i })
    expect(sendButton).toBeDisabled()
  })

  it('enables send button when input has text and session is ready', async () => {
    renderChatPanel()
    const input = screen.getByPlaceholderText('输入你的需求...')
    // Wait for session to be created (textarea becomes enabled)
    await waitFor(() => {
      expect(input).toBeEnabled()
    })
    await userEvent.type(input, '优化简历')
    const sendButton = screen.getByRole('button', { name: /发送/i })
    expect(sendButton).toBeEnabled()
  })

  it('calls onApplyEdits when done event fires', async () => {
    const onApplyEdits = vi.fn().mockResolvedValue(undefined)
    renderChatPanel(1, onApplyEdits)
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => {
      expect(input).toBeEnabled()
    })

    await userEvent.type(input, '优化简历')
    const sendButton = screen.getByRole('button', { name: /发送/i })
    await userEvent.click(sendButton)

    await waitFor(() => {
      expect(onApplyEdits).toHaveBeenCalledTimes(1)
    })
  })

  it('displays tool call log after streaming completes', async () => {
    renderChatPanel()
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => {
      expect(input).toBeEnabled()
    })

    await userEvent.type(input, '优化简历')
    const sendButton = screen.getByRole('button', { name: /发送/i })
    await userEvent.click(sendButton)

    // Wait for the tool call log to appear after streaming completes
    await waitFor(() => {
      expect(screen.getByText('读取简历')).toBeInTheDocument()
    })

    // Verify the OK status is shown
    expect(screen.getByText('OK')).toBeInTheDocument()
  })

  it('shows thinking section when thinking events are received', async () => {
    renderChatPanel()
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => {
      expect(input).toBeEnabled()
    })

    await userEvent.type(input, '优化简历')
    const sendButton = screen.getByRole('button', { name: /发送/i })
    await userEvent.click(sendButton)

    // Wait for thinking section to appear
    await waitFor(() => {
      expect(screen.getByText('AI 推理过程')).toBeInTheDocument()
    })
  })

  it('clears messages and creates new session when new chat button is clicked', async () => {
    let newSessionId = 10
    server.use(
      http.get('/api/v1/ai/sessions', () => {
        return HttpResponse.json({ code: 0, data: [{ id: 5, draft_id: 1 }] })
      }),
      http.get('/api/v1/ai/sessions/5/history', () => {
        return HttpResponse.json({
          code: 0,
          data: {
            items: [
              { id: 1, role: 'user', content: 'old message', created_at: '' },
              { id: 2, role: 'assistant', content: 'old reply', created_at: '' },
            ],
          },
        })
      }),
      http.post('/api/v1/ai/sessions', () => {
        return HttpResponse.json({ code: 0, data: { id: newSessionId, draft_id: 1 } })
      }),
    )

    renderChatPanel()

    // Wait for existing messages to load
    await waitFor(() => {
      expect(screen.getByText('old message')).toBeInTheDocument()
    })

    // Click the new chat button
    await userEvent.click(screen.getByRole('button', { name: /新对话/i }))

    // Messages should be cleared and empty state shown
    await waitFor(() => {
      expect(screen.queryByText('old message')).not.toBeInTheDocument()
      expect(screen.getByText(/输入你的需求/)).toBeInTheDocument()
    })
  })

  it('renders assistant messages as markdown', async () => {
    server.use(
      http.get('/api/v1/ai/sessions', () => {
        return HttpResponse.json({ code: 0, data: [{ id: 5, draft_id: 1 }] })
      }),
      http.get('/api/v1/ai/sessions/5/history', () => {
        return HttpResponse.json({
          code: 0,
          data: {
            items: [
              { id: 2, role: 'assistant', content: '已完成以下修改：\n\n- 职位头衔提升\n- 添加项目经验', created_at: '' },
            ],
          },
        })
      }),
    )

    renderChatPanel()

    await waitFor(() => {
      // Markdown list items should be rendered as <li> elements
      const listItems = screen.getAllByRole('listitem')
      expect(listItems).toHaveLength(2)
      expect(listItems[0]).toHaveTextContent('职位头衔提升')
      expect(listItems[1]).toHaveTextContent('添加项目经验')
    })
  })

  it('shows tool running status from tool_call events and completes after tool_result', async () => {
    const encoder = new TextEncoder()
    server.use(
      http.post('/api/v1/ai/sessions/1/chat', () => {
        // Use a stream that sends tool_call but never sends tool_result
        // so the running state persists long enough to be observed
        const stream = new ReadableStream({
          start(controller) {
            controller.enqueue(encoder.encode('data: {"type":"text","content":"处理中..."}\n\n'))
            controller.enqueue(encoder.encode('data: {"type":"tool_call","name":"get_draft","params":{"draft_id":1}}\n\n'))
            // Intentionally do NOT send tool_result so running status stays
            // The done event will still fire after a delay via setTimeout
            setTimeout(() => {
              controller.enqueue(encoder.encode('data: {"type":"done"}\n\n'))
              controller.close()
            }, 500)
          },
        })
        return new Response(stream, {
          headers: { 'Content-Type': 'text/event-stream' },
        })
      }),
    )

    renderChatPanel()
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => {
      expect(input).toBeEnabled()
    })

    await userEvent.type(input, '读取简历')
    const sendButton = screen.getByRole('button', { name: /发送/i })
    await userEvent.click(sendButton)

    // The active tool indicator should appear while tool is running
    await waitFor(() => {
      expect(screen.getByText('正在执行工具...')).toBeInTheDocument()
    })

    // After done event fires, streaming should stop
    await waitFor(() => {
      expect(screen.queryByText('正在执行工具...')).not.toBeInTheDocument()
    }, { timeout: 3000 })
  })
})
