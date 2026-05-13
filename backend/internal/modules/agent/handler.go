package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

const (
	CodeModelTimeout    = 3001
	CodeModelFormat     = 3002
	CodeSessionNotFound = 3003
	CodeDraftNotFound   = 3004
	CodeMaxIterations   = 3005
	CodeParamInvalid    = 3000
	CodeInternalError   = 3999
)

type Handler struct {
	sessionSvc *SessionService
	chatSvc    *ChatService
	editSvc    *EditService
}

func NewHandler(sessionSvc *SessionService, chatSvc *ChatService, editSvc *EditService) *Handler {
	return &Handler{sessionSvc: sessionSvc, chatSvc: chatSvc, editSvc: editSvc}
}

type createSessionReq struct {
	DraftID uint `json:"draft_id" binding:"required"`
}

type chatReq struct {
	Message string `json:"message" binding:"required"`
}

func (h *Handler) CreateSession(c *gin.Context) {
	var req createSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "draft_id is required")
		return
	}
	session, err := h.sessionSvc.Create(req.DraftID)
	if err != nil {
		response.Error(c, CodeDraftNotFound, "草稿不存在")
		return
	}
	response.Success(c, session)
}

func (h *Handler) ListSessions(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Query("draft_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "draft_id is required")
		return
	}
	sessions, err := h.sessionSvc.ListByDraftID(uint(draftID))
	if err != nil {
		response.Error(c, CodeInternalError, "failed to list sessions")
		return
	}
	response.Success(c, sessions)
}

func (h *Handler) GetSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("session_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid session_id")
		return
	}
	session, err := h.sessionSvc.GetByID(uint(id))
	if err != nil {
		response.ErrorWithStatus(c, http.StatusNotFound, CodeSessionNotFound, "会话不存在")
		return
	}
	response.Success(c, session)
}

func (h *Handler) DeleteSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("session_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid session_id")
		return
	}
	if err := h.sessionSvc.Delete(uint(id)); err != nil {
		response.ErrorWithStatus(c, http.StatusNotFound, CodeSessionNotFound, "会话不存在")
		return
	}
	response.Success(c, nil)
}

// errorCode maps sentinel errors to module error codes.
func errorCode(err error) int {
	switch {
	case errors.Is(err, ErrSessionNotFound):
		return CodeSessionNotFound
	case errors.Is(err, ErrDraftNotFound):
		return CodeDraftNotFound
	case errors.Is(err, ErrModelTimeout):
		return CodeModelTimeout
	case errors.Is(err, ErrModelFormat):
		return CodeModelFormat
	case errors.Is(err, ErrMaxIterations):
		return CodeMaxIterations
	default:
		return CodeInternalError
	}
}

func (h *Handler) Chat(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("session_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid session_id")
		return
	}
	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "message is required")
		return
	}

	logger := middleware.LoggerFromContext(c)
	logger.Info("chat_started", "sessionID", sessionID, "messageLen", len(req.Message))

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Error(c, CodeInternalError, "streaming not supported")
		return
	}

	sendEvent := func(event string) error {
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", event); err != nil {
			return fmt.Errorf("write SSE event: %w", err)
		}
		flusher.Flush()
		return nil
	}

	if err := h.chatSvc.StreamChatReAct(c.Request.Context(), uint(sessionID), req.Message, sendEvent); err != nil {
		errJSON, _ := json.Marshal(map[string]interface{}{
			"type":    "error",
			"code":    errorCode(err),
			"message": err.Error(),
		})
		// 尝试发送错误事件，如果失败则忽略（连接可能已断开）
		_ = sendEvent(string(errJSON))
	}
	// 尝试发送完成事件，如果失败则忽略（连接可能已断开）
	_ = sendEvent(`{"type":"done"}`)
}

func (h *Handler) GetHistory(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("session_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid session_id")
		return
	}
	messages, err := h.sessionSvc.GetHistory(uint(sessionID))
	if err != nil {
		response.Error(c, CodeInternalError, "failed to get history")
		return
	}
	var toolCalls []struct {
		Name      string                 `json:"name"`
		Status    string                 `json:"status"`
		Params    map[string]interface{} `json:"params,omitempty"`
		Result    string                 `json:"result,omitempty"`
		CreatedAt string                 `json:"created_at"`
	}
	var dbToolCalls []models.AIToolCall
	if err := h.chatSvc.db.Where("session_id = ?", sessionID).Order("id ASC").Find(&dbToolCalls).Error; err != nil {
		response.Error(c, CodeInternalError, "failed to get tool calls")
		return
	}
	for _, call := range dbToolCalls {
		result := ""
		if call.Error != nil && *call.Error != "" {
			result = toolErrorForClient(*call.Error)
		} else if call.Result != nil {
			if b, err := json.Marshal(call.Result); err == nil {
				result = string(b)
			}
		}
		toolCalls = append(toolCalls, struct {
			Name      string                 `json:"name"`
			Status    string                 `json:"status"`
			Params    map[string]interface{} `json:"params,omitempty"`
			Result    string                 `json:"result,omitempty"`
			CreatedAt string                 `json:"created_at"`
		}{
			Name:      call.ToolName,
			Status:    call.Status,
			Params:    call.Params,
			Result:    result,
			CreatedAt: call.CreatedAt.Format(time.RFC3339Nano),
		})
	}

	response.Success(c, gin.H{"items": messages, "tool_calls": toolCalls})
}

func (h *Handler) Undo(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("draft_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid draft_id")
		return
	}
	html, err := h.editSvc.Undo(uint(draftID))
	if err != nil {
		response.Error(c, CodeInternalError, err.Error())
		return
	}
	response.Success(c, gin.H{"html_content": html})
}

func (h *Handler) Redo(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("draft_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid draft_id")
		return
	}
	html, err := h.editSvc.Redo(uint(draftID))
	if err != nil {
		response.Error(c, CodeInternalError, err.Error())
		return
	}
	response.Success(c, gin.H{"html_content": html})
}
