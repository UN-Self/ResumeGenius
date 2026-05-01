import { useState, useRef, useEffect, useCallback } from 'react'
import { Sparkles, Send } from 'lucide-react'
import { agentApi, type AISession } from '@/lib/api-client'
import { HtmlPreview } from './HtmlPreview'

interface Message {
  role: 'user' | 'assistant'
  text: string
}

interface Props {
  draftId: number
  onApplyHTML?: (html: string) => void
}

export function ChatPanel({ draftId, onApplyHTML }: Props) {
  const [session, setSession] = useState<AISession | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [htmlPreview, setHtmlPreview] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Reuse existing session or create one on mount / draft change
  useEffect(() => {
    let cancelled = false
    setSession(null)
    setMessages([])
    setHtmlPreview(null)
    setError(null)

    const init = async () => {
      try {
        const sessions = await agentApi.listSessions(draftId)
        if (cancelled) return

        let s: AISession
        if (sessions && sessions.length > 0) {
          s = sessions[0]
          setSession(s)
          const history = await agentApi.getHistory(s.id)
          if (!cancelled) {
            setMessages(history.items.map((m) => ({ role: m.role, text: m.content })))
          }
        } else {
          s = await agentApi.createSession(draftId)
          if (!cancelled) setSession(s)
        }
      } catch {
        if (!cancelled) setError('创建会话失败')
      }
    }

    init()
    return () => { cancelled = true }
  }, [draftId])

  // Auto-scroll
  useEffect(() => {
    scrollRef.current?.scrollIntoView?.({ behavior: 'smooth' })
  }, [messages, htmlPreview])

  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || !session || streaming) return

    setInput('')
    setError(null)
    setMessages((prev) => [...prev, { role: 'user', text }])
    setStreaming(true)

    try {
      const resp = await fetch(`/api/v1/ai/sessions/${session.id}/chat`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: text }),
      })

      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)

      const reader = resp.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let currentText = ''
      let currentHTML = ''

      const HTML_START = '<!--RESUME_HTML_START-->'
      const HTML_END = '<!--RESUME_HTML_END-->'
      let inHTML = false

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed.startsWith('data: ')) continue

          try {
            const event = JSON.parse(trimmed.slice(6))
            switch (event.type) {
              case 'text':
                currentText += event.content
                // Detect and extract HTML markers from the text stream
                if (!inHTML) {
                  const startIdx = currentText.indexOf(HTML_START)
                  if (startIdx !== -1) {
                    inHTML = true
                    currentHTML = ''
                    const afterStart = currentText.substring(startIdx + HTML_START.length)
                    const endIdx = afterStart.indexOf(HTML_END)
                    if (endIdx !== -1) {
                      currentHTML = afterStart.substring(0, endIdx)
                      inHTML = false
                      setHtmlPreview(currentHTML)
                    } else {
                      currentHTML = afterStart
                    }
                  }
                } else {
                  const endIdx = currentHTML.indexOf(HTML_END)
                  if (endIdx !== -1) {
                    currentHTML = currentHTML.substring(0, endIdx)
                    inHTML = false
                    setHtmlPreview(currentHTML)
                  }
                }
                setMessages((prev) => {
                  const last = prev[prev.length - 1]
                  if (last?.role === 'assistant') {
                    return [...prev.slice(0, -1), { ...last, text: currentText }]
                  }
                  return [...prev, { role: 'assistant', text: currentText }]
                })
                break
              case 'error':
                setError(event.message || 'AI 响应出错')
                break
            }
          } catch {
            // Skip unparseable lines
          }
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '连接失败')
    } finally {
      setStreaming(false)
    }
  }, [input, session, streaming])

  const handleApply = useCallback(() => {
    if (htmlPreview && onApplyHTML) {
      onApplyHTML(htmlPreview)
      setHtmlPreview(null)
    }
  }, [htmlPreview, onApplyHTML])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }, [handleSend])

  return (
    <div className="flex flex-col h-full bg-[var(--color-page-bg)]">
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2.5 border-b border-[var(--color-divider)]">
        <Sparkles size={16} className="text-[var(--color-primary)]" />
        <span className="text-sm font-medium text-[var(--color-text-main)]">AI 助手</span>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-3 py-3 space-y-3">
        {messages.length === 0 && !streaming && (
          <p className="text-xs text-[var(--color-text-secondary)] text-center mt-8">
            输入你的需求，AI 将帮你优化简历内容
          </p>
        )}

        {messages.map((msg, i) => (
          <div key={i} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[85%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap ${
              msg.role === 'user'
                ? 'bg-[var(--color-primary)] text-white'
                : 'bg-white border border-[var(--color-divider)] text-[var(--color-text-main)]'
            }`}>
              {msg.text}
            </div>
          </div>
        ))}

        {/* HTML Preview */}
        {htmlPreview && (
          <div className="border border-green-300 rounded-lg overflow-hidden bg-white">
            <div className="flex items-center justify-between px-3 py-1.5 bg-green-50 border-b border-green-200">
              <span className="text-xs font-medium text-green-700">HTML 预览</span>
              <button
                onClick={handleApply}
                disabled={streaming}
                className="text-xs bg-green-600 text-white px-3 py-1 rounded hover:bg-green-700 disabled:opacity-50 transition-colors cursor-pointer"
              >
                应用到简历
              </button>
            </div>
            <HtmlPreview html={htmlPreview} />
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="text-xs text-red-500 text-center bg-red-50 rounded px-3 py-2">
            {error}
          </div>
        )}

        <div ref={scrollRef} />
      </div>

      {/* Input */}
      <div className="border-t border-[var(--color-divider)] p-3">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入你的需求..."
            disabled={streaming || !session}
            rows={2}
            className="flex-1 border border-[var(--color-divider)] rounded-md px-3 py-2 text-sm resize-none focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)] disabled:bg-gray-50 disabled:text-[var(--color-text-disabled)]"
          />
          <button
            onClick={handleSend}
            disabled={streaming || !input.trim() || !session}
            className="self-end bg-[var(--color-primary)] text-white p-2 rounded-md hover:opacity-90 disabled:opacity-50 transition-opacity cursor-pointer"
            aria-label="发送"
          >
            <Send size={16} />
          </button>
        </div>
      </div>
    </div>
  )
}
