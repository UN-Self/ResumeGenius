import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { server } from './setup'
import { ChatPanel } from '@/components/chat/ChatPanel'

function renderChatPanel(draftId = 1, onApplyHTML = vi.fn()) {
  return render(<ChatPanel draftId={draftId} onApplyHTML={onApplyHTML} />)
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

  it('extracts HTML from RESUME_HTML markers in text events and shows preview', async () => {
    renderChatPanel()
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => {
      expect(input).toBeEnabled()
    })

    await userEvent.type(input, '优化简历')
    const sendButton = screen.getByRole('button', { name: /发送/i })
    await userEvent.click(sendButton)

    // Wait for HTML preview to appear (HTML提取成功)
    await waitFor(() => {
      expect(screen.getByText('HTML 预览')).toBeInTheDocument()
    })
    expect(screen.getByText('应用到简历')).toBeInTheDocument()
    // Verify HtmlPreview iframe is rendered with extracted HTML
    const iframe = screen.getByTitle('AI 生成的简历 HTML 预览')
    expect(iframe).toBeInTheDocument()
    expect(iframe.tagName).toBe('IFRAME')
  })

  it('calls onApplyHTML when apply button is clicked', async () => {
    const onApplyHTML = vi.fn()
    renderChatPanel(1, onApplyHTML)
    const input = screen.getByPlaceholderText('输入你的需求...')
    await waitFor(() => {
      expect(input).toBeEnabled()
    })

    await userEvent.type(input, '优化简历')
    const sendButton = screen.getByRole('button', { name: /发送/i })
    await userEvent.click(sendButton)

    // Wait for apply button to appear
    await waitFor(() => {
      expect(screen.getByText('应用到简历')).toBeInTheDocument()
    })

    await userEvent.click(screen.getByText('应用到简历'))

    // Verify onApplyHTML was called with extracted HTML
    expect(onApplyHTML).toHaveBeenCalledTimes(1)
    const appliedHTML = onApplyHTML.mock.calls[0][0] as string
    expect(appliedHTML).toContain('<html>')
    expect(appliedHTML).toContain('Mock优化简历')
    expect(appliedHTML).not.toContain('<!--RESUME_HTML_START-->')
    expect(appliedHTML).not.toContain('<!--RESUME_HTML_END-->')
  })
})
