package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

const (
	defaultTokenTTL = 365 * 24 * time.Hour
)

type Handler struct {
	service      *Service
	secret       string
	ttl          time.Duration
	cookieSecure bool
}

func NewHandler(service *Service, secret string, ttl time.Duration, cookieSecure bool) *Handler {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return &Handler{service: service, secret: secret, ttl: ttl, cookieSecure: cookieSecure}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResp struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "invalid request body")
		return
	}

	user, err := h.service.LoginOrRegister(req.Username, req.Password)
	if err != nil {
		switch err {
		case ErrInvalidUsername:
			response.Error(c, 40000, "username length must be between 3 and 64")
		case ErrInvalidPassword:
			response.Error(c, 40000, "password length must be between 6 and 128")
		case ErrInvalidCredentials:
			response.Error(c, 40100, "invalid username or password")
		default:
			response.Error(c, 50000, "failed to login")
		}
		return
	}

	token, err := middleware.IssueToken(user.ID, user.Username, h.secret, h.ttl)
	if err != nil {
		response.Error(c, 50000, "failed to issue token")
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		middleware.AccessTokenCookieName,
		token,
		int(h.ttl.Seconds()),
		"/",
		"",
		h.cookieSecure,
		true,
	)

	response.Success(c, userResp{
		ID:       user.ID,
		Username: user.Username,
	})
}

func (h *Handler) Me(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "unauthorized")
		return
	}

	user, err := h.service.GetByID(userID)
	if err != nil {
		response.Error(c, 40100, "unauthorized")
		return
	}

	response.Success(c, userResp{
		ID:       user.ID,
		Username: user.Username,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	response.Success(c, nil)
}
