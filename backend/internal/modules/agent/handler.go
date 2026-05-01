package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

const (
	CodeModelTimeout    = 3001
	CodeModelFormat     = 3002
	CodeSessionNotFound = 3003
	CodeDraftNotFound   = 3004
	CodeParamInvalid    = 30000
	CodeInternalError   = 50000
)

type Handler struct {
	sessionSvc *SessionService
	chatSvc    *ChatService
}

func NewHandler(sessionSvc *SessionService, chatSvc *ChatService) *Handler {
	return &Handler{sessionSvc: sessionSvc, chatSvc: chatSvc}
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

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Error(c, CodeInternalError, "streaming not supported")
		return
	}

	sendEvent := func(event string) {
		fmt.Fprintf(c.Writer, "data: %s\n\n", event)
		flusher.Flush()
	}

	if err := h.chatSvc.StreamChat(uint(sessionID), req.Message, sendEvent); err != nil {
		errJSON, _ := json.Marshal(map[string]interface{}{
			"type":    "error",
			"code":    errorCode(err),
			"message": err.Error(),
		})
		sendEvent(string(errJSON))
	}
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
	response.Success(c, gin.H{"items": messages})
}
