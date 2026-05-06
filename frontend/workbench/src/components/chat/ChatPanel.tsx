import { useState, useRef, useEffect, useCallback } from 'react'
import { Sparkles, Send, Plus } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { agentApi, undoDraft, redoDraft, type AISession, type ToolCallEntry } from '@/lib/api-client'
import { ToolCallLog } from './ToolCallLog'

interface Message {
  role: 'user' | 'assistant'
  text: string
}

interface Props {
  draftId: number
  onApplyEdits?: () => Promise<void>
  onRestoreHtml?: (html: string) => void
}

export function ChatPanel({ draftId, onApplyEdits, onRestoreHtml }: Props) {
  const [session, setSession] = useState<AISession | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [thinking, setThinking] = useState('')
  const [toolCalls, setToolCalls] = useState<ToolCallEntry[]>([])
  const [editsApplied, setEditsApplied] = useState(false)
  const [undoRedoLoading, setUndoRedoLoading] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false
    setSession(null)
    setMessages([])
    setThinking('')
    setToolCalls([])
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
            const loadedMessages: Message[] = []
            for (const m of history.items) {
              loadedMessages.push({ role: m.role, text: m.content })
            }
            setMessages(loadedMessages)
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

  useEffect(() => {
    scrollRef.current?.scrollIntoView?.({ behavior: 'smooth' })
  }, [messages, toolCalls])

  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || !session || streaming) return

    setInput('')
    setThinking('')
    setToolCalls([])
    setEditsApplied(false)
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
      let gotDone = false

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
              case 'thinking':
                setThinking(prev => prev + event.content)
                break
              case 'tool_call':
                setToolCalls(prev => [...prev, { name: event.name, status: 'running', params: event.params }])
                break
              case 'tool_result':
                setToolCalls(prev => {
                  const updated = [...prev]
                  const last = updated[updated.length - 1]
                  if (last) updated[updated.length - 1] = { ...last, status: event.status }
                  return updated
                })
                break
              case 'done':
                gotDone = true
                if (onApplyEdits) {
                  await onApplyEdits()
                  setEditsApplied(true)
                }
                break
              case 'text':
                currentText += event.content
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
      if (!gotDone) {
        setError('连接中断，AI 回复可能不完整')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '连接失败')
    } finally {
      setStreaming(false)
    }
  }, [input, session, streaming, onApplyEdits])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }, [handleSend])

  const handleNewChat = useCallback(async () => {
    setMessages([])
    setThinking('')
    setToolCalls([])
    setEditsApplied(false)
    setError(null)
    try {
      const s = await agentApi.createSession(draftId)
      setSession(s)
    } catch {
      setError('创建新会话失败')
    }
  }, [draftId])

  return (
    <div className="flex flex-col h-full bg-[var(--color-page-bg)]">
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2.5 border-b border-[var(--color-divider)]">
        <Sparkles size={16} className="text-[var(--color-primary)]" />
        <span className="text-sm font-medium text-[var(--color-text-main)]">AI 助手</span>
        <div className="flex-1" />
        <button
          onClick={handleNewChat}
          disabled={streaming}
          aria-label="新对话"
          className="text-[var(--color-text-secondary)] hover:text-[var(--color-text-main)] disabled:opacity-50 cursor-pointer"
        >
          <Plus size={16} />
        </button>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-3 py-3 space-y-3">
        {/* Thinking collapsible */}
        {thinking && (
          <details className="bg-[var(--color-page-bg)] border border-[var(--color-divider)] rounded-lg text-sm">
            <summary className="px-3 py-1.5 text-xs font-medium text-[var(--color-text-secondary)] cursor-pointer select-none">
              AI 推理过程
            </summary>
            <pre className="px-3 py-2 text-xs text-[var(--color-text-secondary)] whitespace-pre-wrap max-h-40 overflow-y-auto border-t border-[var(--color-divider)]">{thinking}</pre>
          </details>
        )}

        {messages.length === 0 && !streaming && (
          <p className="text-xs text-[var(--color-text-secondary)] text-center mt-8">
            输入你的需求，AI 将帮你优化简历内容
          </p>
        )}

        {messages.map((msg, i) => (
          <div key={i}>
            <div className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-[85%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap ${
                msg.role === 'user'
                  ? 'bg-[var(--color-primary)] text-white'
                  : 'bg-white border border-[var(--color-divider)] text-[var(--color-text-main)]'
              }`}>
                {msg.role === 'user' ? (
                  msg.text
                ) : (
                  <ReactMarkdown>{msg.text}</ReactMarkdown>
                )}
              </div>
            </div>
            {/* Show tool calls after the last assistant message */}
            {msg.role === 'assistant' && i === messages.length - 1 && !streaming && toolCalls.length > 0 && (
              <ToolCallLog calls={toolCalls} />
            )}
            {/* Show undo/redo after the last assistant message when edits were applied */}
            {msg.role === 'assistant' && i === messages.length - 1 && !streaming && editsApplied && (
              <div className="flex gap-1 mt-1">
                <button
                  onClick={async () => {
                    setUndoRedoLoading(true)
                    try {
                      const result = await undoDraft(draftId)
                      onRestoreHtml?.(result.html_content)
                    } finally { setUndoRedoLoading(false) }
                  }}
                  disabled={undoRedoLoading}
                  aria-label="Undo"
                  className="px-2 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded disabled:opacity-50 cursor-pointer"
                >
                  Undo
                </button>
                <button
                  onClick={async () => {
                    setUndoRedoLoading(true)
                    try {
                      const result = await redoDraft(draftId)
                      onRestoreHtml?.(result.html_content)
                    } finally { setUndoRedoLoading(false) }
                  }}
                  disabled={undoRedoLoading}
                  aria-label="Redo"
                  className="px-2 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded disabled:opacity-50 cursor-pointer"
                >
                  Redo
                </button>
              </div>
            )}
          </div>
        ))}

        {/* Active tool indicator */}
        {streaming && toolCalls.some(c => c.status === 'running') && (
          <div className="flex items-center gap-2 px-3 py-2 text-xs text-[var(--color-text-secondary)] bg-[var(--color-page-bg)] border border-[var(--color-divider)] rounded-lg">
            <span className="animate-spin">...</span>
            <span>正在执行工具...</span>
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
