import { useCallback, useEffect, useRef, useState } from 'react'
import {
  CheckCircle2,
  LoaderCircle,
  Plus,
  Redo2,
  Send,
  Sparkles,
  Undo2,
  WandSparkles,
} from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { agentApi, redoDraft, undoDraft, type AISession, type ToolCallEntry } from '@/lib/api-client'
import '@/styles/editor.css'
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

interface StreamEvent {
  type: 'thinking' | 'tool_call' | 'tool_result' | 'edit' | 'done' | 'text' | 'error'
  content?: string
  name?: string
  status?: ToolCallEntry['status']
  params?: Record<string, unknown>
  result?: string
  message?: string
}

function updateLatestToolCall(
  calls: ToolCallEntry[],
  name: string | undefined,
  patch: Partial<ToolCallEntry>,
) {
  const updated = [...calls]
  const targetIndex = name
    ? updated.map((call) => call.name).lastIndexOf(name)
    : updated.length - 1

  if (targetIndex >= 0) {
    updated[targetIndex] = { ...updated[targetIndex], ...patch }
  }

  return updated
}

function getFriendlyErrorMessage(message: string) {
  const normalized = message.toLowerCase()
  const isRateLimited =
    normalized.includes('status 429') ||
    normalized.includes('too many requests') ||
    normalized.includes('rate limit') ||
    normalized.includes('"code":"1302"') ||
    normalized.includes('"code": "1302"') ||
    message.includes('速率限制') ||
    message.includes('请求频率')

  if (isRateLimited) {
    return '模型服务触发限流了，请等 30-60 秒后再试；连续点“重试”会更容易触发。'
  }

  const isTimeout =
    normalized.includes('model call timeout') ||
    normalized.includes('context deadline exceeded') ||
    normalized.includes('client.timeout exceeded') ||
    normalized.includes('awaiting headers') ||
    normalized.includes('http 504') ||
    normalized.includes('gateway timeout')

  if (isTimeout) {
    return '模型服务响应超时了。通常是上游排队或网络波动，请稍后再试；这次没有应用任何修改。'
  }

  if (message.includes('max tool-calling iterations exceeded')) {
    return '这轮 AI 一直在查资料但没有生成可应用修改，我已停止本轮。请再发一句明确目标，我会直接把修改写入画布。'
  }
  return message
}

function ThinkingBubble({
  toolCalls,
  elapsedSeconds,
}: {
  toolCalls: ToolCallEntry[]
  elapsedSeconds: number
}) {
  const runningTool = [...toolCalls].reverse().find((call) => call.status === 'running')
  const label = runningTool
    ? runningTool.name === 'apply_edits'
      ? '正在同步到画布'
      : runningTool.name === 'search_design_skill'
        ? '正在查找设计参考'
        : '正在处理资料'
    : '正在构思简历方案'

  return (
    <div className="ai-thinking-card" role="status" aria-live="polite">
      <div className="ai-thinking-main">
        <span className="ai-thinking-mark">
          <Sparkles className="h-3.5 w-3.5" />
        </span>
        <div className="min-w-0">
          <div className="ai-thinking-title">{label}</div>
          <div className="ai-thinking-subtitle">AI 正在把内容、结构和视觉细节对齐</div>
          <div className="ai-thinking-subtitle">已等待 {elapsedSeconds}s</div>
        </div>
      </div>
      {elapsedSeconds >= 20 && (
        <div className="ai-wait-note">
          模型首个响应偏慢，通常是上游排队、限流或网络抖动。
        </div>
      )}
      <div className="ai-thinking-flow" aria-hidden="true">
        <span />
        <span />
        <span />
      </div>
    </div>
  )
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
  const [elapsedSeconds, setElapsedSeconds] = useState(0)
  const [planeCrashing, setPlaneCrashing] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const abortControllerRef = useRef<AbortController | null>(null)
  const cancelRequestedRef = useRef(false)
  const crashTimerRef = useRef<number | null>(null)

  useEffect(() => {
    let cancelled = false
    /* eslint-disable react-hooks/set-state-in-effect -- Chat state should reset immediately when switching drafts. */
    setSession(null)
    setMessages([])
    setThinking('')
    setToolCalls([])
    setError(null)
    setEditsApplied(false)
    /* eslint-enable react-hooks/set-state-in-effect */

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
  }, [messages, toolCalls, streaming, editsApplied])

  useEffect(() => {
    if (!streaming) {
      setElapsedSeconds(0)
      return
    }

    const startedAt = Date.now()
    const timer = window.setInterval(() => {
      setElapsedSeconds(Math.max(0, Math.floor((Date.now() - startedAt) / 1000)))
    }, 1000)

    return () => window.clearInterval(timer)
  }, [streaming])

  useEffect(() => {
    return () => {
      if (crashTimerRef.current !== null) {
        window.clearTimeout(crashTimerRef.current)
      }
    }
  }, [])

  const restoreHtml = useCallback(async (direction: 'undo' | 'redo') => {
    setUndoRedoLoading(true)
    try {
      const result = direction === 'undo'
        ? await undoDraft(draftId)
        : await redoDraft(draftId)
      onRestoreHtml?.(result.html_content)
    } catch (restoreError) {
      setError(restoreError instanceof Error ? restoreError.message : '恢复画布失败')
    } finally {
      setUndoRedoLoading(false)
    }
  }, [draftId, onRestoreHtml])

  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || !session || streaming) return

    setInput('')
    setThinking('已发送请求，等待模型服务响应...\n')
    setToolCalls([])
    setEditsApplied(false)
    setError(null)
    setPlaneCrashing(false)
    cancelRequestedRef.current = false
    setMessages((prev) => [...prev, { role: 'user', text }])
    setStreaming(true)

    try {
      const abortController = new AbortController()
      abortControllerRef.current = abortController
      const resp = await fetch(`/api/v1/ai/sessions/${session.id}/chat`, {
        method: 'POST',
        credentials: 'include',
        signal: abortController.signal,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: text }),
      })

      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)

      const reader = resp.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let currentText = ''
      let gotDone = false
      let receivedEdit = false

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
            const event = JSON.parse(trimmed.slice(6)) as StreamEvent
            switch (event.type) {
              case 'thinking':
                setThinking((prev) => prev + (event.content ?? ''))
                break
              case 'tool_call':
                if (event.name) {
                  setToolCalls((prev) => [
                    ...prev,
                    { name: event.name!, status: 'running', params: event.params },
                  ])
                }
                break
              case 'tool_result':
                setToolCalls((prev) => {
                  const patch: Partial<ToolCallEntry> = { status: event.status ?? 'completed' }
                  if (event.result !== undefined) patch.result = event.result
                  return updateLatestToolCall(prev, event.name, patch)
                })
                break
              case 'edit':
                receivedEdit = true
                setToolCalls((prev) => updateLatestToolCall(prev, event.name ?? 'apply_edits', {
                  status: 'completed',
                  result: event.result,
                }))
                break
              case 'done':
                gotDone = true
                setToolCalls((prev) => prev.map((call) => (
                  call.status === 'running' ? { ...call, status: 'completed' } : call
                )))
                if (receivedEdit && onApplyEdits) {
                  try {
                    await onApplyEdits()
                    setEditsApplied(true)
                  } catch (applyError) {
                    setError(applyError instanceof Error ? applyError.message : '应用到画布失败')
                  }
                }
                break
              case 'text':
                currentText += event.content ?? ''
                setMessages((prev) => {
                  const last = prev[prev.length - 1]
                  if (last?.role === 'assistant') {
                    return [...prev.slice(0, -1), { ...last, text: currentText }]
                  }
                  return [...prev, { role: 'assistant', text: currentText }]
                })
                break
              case 'error':
                setError(getFriendlyErrorMessage(event.message || 'AI 响应出错'))
                break
            }
          } catch {
            // Skip unparseable SSE lines.
          }
        }
      }
      if (!gotDone) {
        setError(cancelRequestedRef.current ? '已停止本轮请求。' : '连接中断，AI 回复可能不完整')
      }
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') {
        setError('已停止本轮请求。')
      } else {
        setError(getFriendlyErrorMessage(err instanceof Error ? err.message : '连接失败'))
      }
    } finally {
      abortControllerRef.current = null
      cancelRequestedRef.current = false
      setStreaming(false)
    }
  }, [input, onApplyEdits, session, streaming])

  const handleCancelStreaming = useCallback(() => {
    const controller = abortControllerRef.current
    if (!controller) return

    setPlaneCrashing(true)
    cancelRequestedRef.current = true
    controller.abort()

    if (crashTimerRef.current !== null) {
      window.clearTimeout(crashTimerRef.current)
    }
    crashTimerRef.current = window.setTimeout(() => {
      setPlaneCrashing(false)
      crashTimerRef.current = null
    }, 760)
  }, [])

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

  const waitingForAssistant = streaming && messages[messages.length - 1]?.role !== 'assistant'
  const canSend = Boolean(input.trim() && session && !streaming)

  return (
    <div className="ai-chat-panel">
      <div className="ai-chat-header">
        <div className="ai-chat-title">
          <span className="ai-chat-title-icon">
            <WandSparkles className="h-4 w-4" />
          </span>
          <span>AI 助手</span>
        </div>
        <button
          type="button"
          onClick={handleNewChat}
          disabled={streaming}
          aria-label="新对话"
          className="ai-chat-icon-button"
        >
          <Plus size={16} />
        </button>
      </div>

      <div className="ai-chat-scroll">
        {messages.length === 0 && !streaming && (
          <div className="ai-empty-chat">
            <Sparkles className="h-5 w-5" />
            <p>告诉我你想优化的方向，我会把修改直接同步到画布。</p>
          </div>
        )}

        {messages.map((msg, i) => (
          <div key={i} className={`ai-message-row ${msg.role === 'user' ? 'is-user' : 'is-assistant'}`}>
            <div data-message-role={msg.role} className="ai-message-bubble">
              {msg.role === 'user' ? (
                msg.text
              ) : (
                <div className="ai-markdown">
                  <ReactMarkdown>{msg.text}</ReactMarkdown>
                </div>
              )}
            </div>
          </div>
        ))}

        {waitingForAssistant && (
          <ThinkingBubble
            toolCalls={toolCalls}
            elapsedSeconds={elapsedSeconds}
          />
        )}

        {thinking && (
          <details className="ai-reasoning-panel" open={streaming || undefined}>
            <summary>推理过程</summary>
            <pre>{thinking}</pre>
          </details>
        )}

        {toolCalls.length > 0 && (
          <ToolCallLog calls={toolCalls} compact={streaming} />
        )}

        {!streaming && editsApplied && (
          <div className="ai-apply-card">
            <div className="ai-apply-status">
              <span className="ai-apply-icon">
                <CheckCircle2 className="h-4 w-4" />
              </span>
              <div>
                <div className="ai-apply-title">已应用到画布</div>
                <div className="ai-apply-subtitle">样式和内容已刷新，可以继续追问或回退。</div>
              </div>
            </div>
            <div className="ai-apply-actions">
              <button
                type="button"
                onClick={() => void restoreHtml('undo')}
                disabled={undoRedoLoading}
                aria-label="撤销本次修改"
                className="ai-history-button"
              >
                {undoRedoLoading ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Undo2 className="h-3.5 w-3.5" />}
                <span>撤销</span>
              </button>
              <button
                type="button"
                onClick={() => void restoreHtml('redo')}
                disabled={undoRedoLoading}
                aria-label="重做修改"
                className="ai-history-button"
              >
                {undoRedoLoading ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Redo2 className="h-3.5 w-3.5" />}
                <span>重做</span>
              </button>
            </div>
          </div>
        )}

        {error && (
          <div className="ai-error-card">
            {error}
          </div>
        )}

        <div ref={scrollRef} />
      </div>

      <div className="ai-composer">
        <div className="ai-composer-shell">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入你的需求..."
            disabled={streaming || !session}
            rows={2}
            className="ai-composer-input"
          />
          <button
            type="button"
            onClick={streaming ? handleCancelStreaming : handleSend}
            disabled={!streaming && !canSend}
            className={`ai-send-button ${streaming ? 'is-streaming' : ''} ${planeCrashing ? 'is-crashing' : ''}`}
            aria-label={streaming ? '停止对话' : '发送'}
            data-tooltip={streaming ? '停止对话' : undefined}
          >
            <span className="send-trail trail-one" aria-hidden="true" />
            <span className="send-trail trail-two" aria-hidden="true" />
            <Send className="send-plane h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
