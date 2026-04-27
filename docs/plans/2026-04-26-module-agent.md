# 模块 agent — AI 对话助手 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 SSE 流式 AI 对话，AI 能返回带 HTML 标记的简历修改建议，前端可一键应用。

**Architecture:** SessionService 管理会话和消息持久化，ChatService 调用 AI API 并通过 SSE 流式推送。AI 响应中使用 `<!--RESUME_HTML_START-->` 和 `<!--RESUME_HTML_END-->` 分隔 HTML。

**Tech Stack:** Gin SSE / net/http streaming / OpenAI-compatible API / React EventSource

**Depends on:** Phase 0 共享基石完成、模块 workbench 的 TipTap 编辑器

**契约文档:** `docs/modules/agent/contract.md`

---

### Task 1: 后端 — SessionService 会话管理

**Files:**
- Create: `backend/internal/modules/agent/session_test.go`
- Create: `backend/internal/modules/agent/session.go`

**Step 1: 写失败测试**

```go
// session_test.go
package agent

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&models.Project{}, &models.Draft{}, &models.AISession{}, &models.AIMessage{})
	return db
}

func TestCreateSession(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html>test</html>"})

	svc := NewSessionService(db)
	session, err := svc.Create(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID == 0 {
		t.Error("expected non-zero session id")
	}
	if session.DraftID != 1 {
		t.Errorf("expected draft_id 1, got %d", session.DraftID)
	}
}

func TestSaveMessage(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html>test</html>"})
	svc := NewSessionService(db)
	session, _ := svc.Create(1)

	err := svc.SaveMessage(session.ID, "user", "帮我优化简历")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg models.AIMessage
	db.Where("session_id = ?", session.ID).First(&msg)
	if msg.Content != "帮我优化简历" {
		t.Errorf("unexpected content: %s", msg.Content)
	}
}

func TestGetHistory(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{Title: "test", Status: "active"})
	db.Create(&models.Draft{ProjectID: 1, HTMLContent: "<html>test</html>"})
	svc := NewSessionService(db)
	session, _ := svc.Create(1)
	svc.SaveMessage(session.ID, "user", "hello")
	svc.SaveMessage(session.ID, "assistant", "hi")

	messages, err := svc.GetHistory(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}
```

**Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/modules/agent/... -v
# Expected: FAIL
```

**Step 3: 实现 session.go**

```go
// session.go
package agent

import (
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type SessionService struct {
	db *gorm.DB
}

func NewSessionService(db *gorm.DB) *SessionService {
	return &SessionService{db: db}
}

func (s *SessionService) Create(draftID uint) (*models.AISession, error) {
	session := models.AISession{DraftID: draftID}
	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *SessionService) SaveMessage(sessionID uint, role, content string) error {
	msg := models.AIMessage{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}
	return s.db.Create(&msg).Error
}

func (s *SessionService) GetHistory(sessionID uint) ([]models.AIMessage, error) {
	var messages []models.AIMessage
	err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}
```

**Step 4: 运行测试确认通过**

```bash
cd backend && go test ./internal/modules/agent/... -v -run TestCreateSession
cd backend && go test ./internal/modules/agent/... -v -run TestSaveMessage
cd backend && go test ./internal/modules/agent/... -v -run TestGetHistory
# Expected: PASS all
```

**Step 5: Commit**

```bash
git add backend/internal/modules/agent/
git commit -m "feat(module-c): implement session management with tests"
```

---

### Task 2: 后端 — SSE 流式对话

**Files:**
- Create: `backend/internal/modules/agent/chat.go`
- Create: `backend/internal/modules/agent/handler.go`
- Modify: `backend/internal/modules/agent/routes.go`

**Step 1: 实现 ChatService**

```go
// chat.go
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

type ChatService struct {
	db     *gorm.DB
	apiURL string
	apiKey string
}

func NewChatService(db *gorm.DB) *ChatService {
	return &ChatService{
		db:     db,
		apiURL: os.Getenv("AI_API_URL"),
		apiKey: os.Getenv("AI_API_KEY"),
	}
}

// StreamResponse 流式调用 AI API，通过 callbacks 推送 SSE 事件
func (s *ChatService) StreamResponse(sessionID uint, userMessage string, sendEvent func(event string)) error {
	// 保存用户消息
	s.db.Create(&models.AIMessage{SessionID: sessionID, Role: "user", Content: userMessage})

	// 获取历史消息作为上下文
	history, _ := s.GetHistory(sessionID)

	var apiMessages []map[string]string
	apiMessages = append(apiMessages, map[string]string{
		"role": "system",
		"content": `你是简历优化助手。根据用户的要求修改简历内容。
当需要修改简历时，在回复中用 <!--RESUME_HTML_START--> 和 <!--RESUME_HTML_END--> 包裹完整的简历 HTML。
格式说明：
1. 先用文字说明你的修改思路
2. 然后输出 <!--RESUME_HTML_START-->
3. 然后输出完整的简历 HTML
4. 最后输出 <!--RESUME_HTML_END-->`,
	})

	for _, msg := range history {
		apiMessages = append(apiMessages, map[string]string{"role": msg.Role, "content": msg.Content})
	}

	if os.Getenv("USE_MOCK") == "true" {
		return s.mockStream(sendEvent)
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":       "default",
		"messages":    apiMessages,
		"temperature": 0.7,
		"stream":      true,
	})

	req, _ := http.NewRequest("POST", s.apiURL, strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("模型调用失败: %w", err)
	}
	defer resp.Body.Close()

	return s.processSSEStream(resp.Body, sessionID, sendEvent)
}

func (s *ChatService) processSSEStream(body io.Reader, sessionID uint, sendEvent func(string)) error {
	scanner := bufio.NewScanner(body)
	fullContent := strings.Builder{}
	inHTML := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}
		fullContent.WriteString(content)

		// 检测 HTML 标记
		if strings.Contains(fullContent.String(), "<!--RESUME_HTML_START-->") && !inHTML {
			inHTML = true
			sendEvent(`{"type": "html_start"}`)
			continue
		}
		if strings.Contains(fullContent.String(), "<!--RESUME_HTML_END-->") && inHTML {
			inHTML = false
			sendEvent(`{"type": "html_end"}`)
			continue
		}

		if inHTML {
			event, _ := json.Marshal(map[string]string{"type": "html_chunk", "content": content})
			sendEvent(string(event))
		} else {
			event, _ := json.Marshal(map[string]string{"type": "text", "content": content})
			sendEvent(string(event))
		}
	}

	// 保存 AI 回复
	s.db.Create(&models.AIMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   fullContent.String(),
	})

	sendEvent(`{"type": "done"}`)
	return scanner.Err()
}

func (s *ChatService) mockStream(sendEvent func(string)) error {
	sendEvent(`{"type": "text", "content": "好的"}`)
	sendEvent(`{"type": "text", "content": "，我来帮你优化简历。\n\n"}`)
	sendEvent(`{"type": "html_start"}`)
	sendEvent(`{"type": "html_chunk", "content": "<!DOCTYPE html><html><body><h1>Mock Modified Resume</h1></body></html>"}`)
	sendEvent(`{"type": "html_end"}`)
	sendEvent(`{"type": "done"}`)

	s.db.Create(&models.AIMessage{
		SessionID: 1,
		Role:      "assistant",
		Content:   "好的，我来帮你优化简历。\n\n<!--RESUME_HTML_START--><!DOCTYPE html><html><body><h1>Mock Modified Resume</h1></body></html><!--RESUME_HTML_END-->",
	})
	return nil
}
```

**Step 2: 实现 handler.go**

```go
// handler.go
package agent

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

type Handler struct {
	sessionSvc *SessionService
	chatSvc    *ChatService
}

func NewHandler(sessionSvc *SessionService, chatSvc *ChatService) *Handler {
	return &Handler{sessionSvc: sessionSvc, chatSvc: chatSvc}
}

func (h *Handler) CreateSession(c *gin.Context) {
	var req struct {
		DraftID uint `json:"draft_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "draft_id is required")
		return
	}
	session, err := h.sessionSvc.Create(req.DraftID)
	if err != nil {
		response.Error(c, 3004, "草稿不存在")
		return
	}
	response.Success(c, session)
}

func (h *Handler) Chat(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "message is required")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Error(c, 50000, "streaming not supported")
		return
	}

	sendEvent := func(event string) {
		fmt.Fprintf(c.Writer, "data: %s\n\n", event)
		flusher.Flush()
	}

	if err := h.chatSvc.StreamResponse(parseUint(sessionID), req.Message, sendEvent); err != nil {
		sendEvent(fmt.Sprintf(`{"type": "error", "message": "%s"}`, err.Error()))
	}
}

func (h *Handler) GetHistory(c *gin.Context) {
	sessionID := c.Param("session_id")
	messages, err := h.sessionSvc.GetHistory(parseUint(sessionID))
	if err != nil {
		response.Error(c, 3003, "会话不存在")
		return
	}
	response.Success(c, gin.H{"items": messages})
}

func parseUint(s string) uint {
	n, _ := fmt.Sscanf(s, "%d", new(uint))
	if n != 1 {
		return 0
	}
	var v uint
	fmt.Sscanf(s, "%d", &v)
	return v
}
```

**Step 3: 更新 routes.go**

```go
package agent

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	sessionSvc := NewSessionService(db)
	chatSvc := NewChatService(db)
	h := NewHandler(sessionSvc, chatSvc)

	rg.POST("/sessions", h.CreateSession)
	rg.POST("/sessions/:session_id/chat", h.Chat)
	rg.GET("/sessions/:session_id/history", h.GetHistory)
}
```

**Step 4: Commit**

```bash
git add backend/internal/modules/agent/
git commit -m "feat(module-c): implement SSE streaming chat with session management"
```

---

### Task 3: 前端 — AI 对话面板

**Files:**
- Create: `frontend/workbench/src/components/chat/ChatPanel.tsx`
- Create: `frontend/workbench/src/components/chat/ChatPanel.test.tsx`

**Step 1: 写失败测试**

```tsx
// ChatPanel.test.tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ChatPanel } from './ChatPanel'

vi.mock('../../lib/api-client', () => ({
  apiClient: {
    post: vi.fn().mockResolvedValue({ id: 1, draft_id: 1 }),
    get: vi.fn().mockResolvedValue({ items: [] }),
  },
}))

describe('ChatPanel', () => {
  it('renders chat input', () => {
    render(<ChatPanel draftId={1} />)
    expect(screen.getByPlaceholderText('输入你的问题...')).toBeInTheDocument()
  })
})
```

**Step 2: 实现 ChatPanel**

```tsx
import { useState, useRef, useEffect } from 'react'
import { apiClient } from '../../lib/api-client'

interface Props {
  draftId: number
  onApplyHTML?: (html: string) => void
}

interface Message {
  role: 'user' | 'assistant'
  text: string
  htmlPreview?: string
}

export function ChatPanel({ draftId, onApplyHTML }: Props) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [htmlPreview, setHtmlPreview] = useState('')
  const [sessionId, setSessionId] = useState<number | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    apiClient.post<{ id: number }>('/ai/sessions', { draft_id: draftId })
      .then(s => setSessionId(s.id))
  }, [draftId])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView()
  }, [messages])

  const handleSend = async () => {
    if (!input.trim() || !sessionId) return
    const userMsg = input.trim()
    setInput('')
    setMessages(prev => [...prev, { role: 'user', text: userMsg }])
    setLoading(true)
    setHtmlPreview('')

    try {
      const resp = await fetch(`/api/v1/ai/sessions/${sessionId}/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: userMsg }),
      })

      const reader = resp.body!.getReader()
      const decoder = new TextDecoder()
      let currentText = ''
      let currentHTML = ''
      let inHTML = false

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const chunk = decoder.decode(value)
        const lines = chunk.split('\n')

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          try {
            const event = JSON.parse(line.slice(6))
           	switch (event.type) {
              case 'text':
                currentText += event.content
                setMessages(prev => {
                  const updated = [...prev]
                  const last = updated[updated.length - 1]
                  if (last?.role === 'assistant') {
                    updated[updated.length - 1] = { ...last, text: currentText }
                  } else {
                    updated.push({ role: 'assistant', text: currentText })
                  }
                  return updated
                })
                break
              case 'html_start':
                inHTML = true
                currentHTML = ''
                break
              case 'html_chunk':
                currentHTML += event.content
                setHtmlPreview(currentHTML)
                break
              case 'html_end':
                inHTML = false
                break
            }
          } catch {}
        }
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col h-full border-l bg-white">
      <div className="p-3 border-b font-medium text-sm">AI 助手</div>
      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {messages.map((msg, i) => (
          <div key={i} className={`text-sm ${msg.role === 'user' ? 'text-right' : ''}`}>
            <div className={`inline-block rounded-lg px-3 py-2 max-w-[90%] ${
              msg.role === 'user' ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-800'
            }`}>
              <pre className="whitespace-pre-wrap font-sans">{msg.text}</pre>
            </div>
          </div>
        ))}
        {htmlPreview && (
          <div className="border rounded p-2 bg-green-50">
            <div className="text-xs text-green-700 mb-1">HTML 预览已就绪</div>
            <button
              onClick={() => onApplyHTML?.(htmlPreview)}
              className="text-xs bg-green-600 text-white px-3 py-1 rounded"
            >
              应用到简历
            </button>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>
      <div className="p-3 border-t">
        <div className="flex gap-2">
          <input
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && !e.shiftKey && handleSend()}
            placeholder="输入你的问题..."
            disabled={loading}
            className="flex-1 border rounded px-3 py-2 text-sm"
          />
          <button
            onClick={handleSend}
            disabled={loading || !input.trim()}
            className="bg-blue-600 text-white px-4 py-2 rounded text-sm disabled:opacity-50"
          >
            发送
          </button>
        </div>
      </div>
    </div>
  )
}
```

**Step 3: 集成到 EditorPage（在右侧面板渲染 ChatPanel）**

在 EditorPage.tsx 中添加：

```tsx
import { ChatPanel } from './components/chat/ChatPanel'

// 在主布局中，编辑器右侧添加：
<div className="flex-1 flex">
  <div className="flex-1 overflow-auto">
    {/* A4 Canvas */}
  </div>
  <div className="w-[360px]">
    <ChatPanel draftId={draftId!} onApplyHTML={(html) => {
      editor?.commands.setContent(html)
    }} />
  </div>
</div>
```

**Step 4: Commit**

```bash
git add frontend/workbench/src/
git commit -m "feat(module-c): implement AI chat panel with SSE streaming"
```

---

## 验证清单

- [ ] `go test ./internal/modules/agent/... -v` 全部通过
- [ ] `curl -X POST localhost:8080/api/v1/ai/sessions -d '{"draft_id":1}'` 创建会话
- [ ] `curl -X POST localhost:8080/api/v1/ai/sessions/1/chat -d '{"message":"优化简历"}' -H "Accept: text/event-stream"` 返回 SSE 流
- [ ] SSE 流包含 `text`、`html_start`、`html_chunk`、`html_end`、`done` 事件
- [ ] `curl localhost:8080/api/v1/ai/sessions/1/history` 返回对话历史
- [ ] 前端 AI 面板显示对话气泡，能流式输出
- [ ] "应用到简历" 按钮能将 AI 返回的 HTML 写入编辑器
- [ ] `USE_MOCK=true` 模式下 mock 流式响应正常工作
