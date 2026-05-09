package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/middleware"
	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
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

// ── Request/Response types ──────────────────────────────────────────

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type sendCodeReq struct {
	Email string `json:"email"`
}

type verifyEmailReq struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type userResp struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	DevCode       string `json:"dev_code,omitempty"`
}

func toUserResp(u *models.User, devMode bool) userResp {
	r := userResp{ID: u.ID, Username: u.Username}
	if u.Email != nil {
		r.Email = *u.Email
	}
	r.EmailVerified = u.EmailVerified
	if devMode && !u.EmailVerified && u.VerificationCode != "" {
		r.DevCode = u.VerificationCode
	}
	return r
}

// ── Handlers ────────────────────────────────────────────────────────

// Login authenticates a user and issues a JWT cookie.
// Supports both username and email login (detected via @).
func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "请求格式错误")
		return
	}

	user, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		switch err {
		case ErrInvalidUsername:
			response.Error(c, 40000, "用户名需 3-64 个字符")
		case ErrInvalidPassword:
			response.Error(c, 40000, "密码需 6-128 个字符")
		case ErrInvalidCredentials:
			response.Error(c, 40100, "用户名或密码错误")
		default:
			response.Error(c, 50000, "登录失败")
		}
		return
	}

	token, err := middleware.IssueToken(user.ID, user.Username, h.secret, h.ttl)
	if err != nil {
		response.Error(c, 50000, "签发凭证失败")
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

	response.Success(c, toUserResp(user, h.service.email.IsDevMode()))
}

// Register creates a new user and sends a verification code to their email.
func (h *Handler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "请求格式错误")
		return
	}

	user, err := h.service.Register(req.Username, req.Password, req.Email)
	if err != nil {
		switch err {
		case ErrInvalidUsername:
			response.Error(c, 40000, "用户名需 3-64 个字符")
		case ErrInvalidPassword:
			response.Error(c, 40000, "密码需 6-128 个字符")
		case ErrInvalidEmail:
			response.Error(c, 40000, "邮箱格式不正确")
		case ErrUsernameTaken:
			response.Error(c, 40000, "用户名已被注册")
		case ErrEmailTaken:
			response.Error(c, 40000, "邮箱已被注册")
		default:
			response.Error(c, 50000, "注册失败")
		}
		return
	}

	response.Success(c, toUserResp(user, h.service.email.IsDevMode()))
}

// SendCode resends a verification code to the email.
func (h *Handler) SendCode(c *gin.Context) {
	var req sendCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "请求格式错误")
		return
	}

	code, err := h.service.SendVerificationCode(req.Email)
	if err != nil {
		switch err {
		case ErrEmailNotFound:
			response.Error(c, 40000, "邮箱未注册，请先注册")
		case ErrEmailAlreadyVerified:
			response.Error(c, 40000, "邮箱已验证")
		default:
			response.Error(c, 50000, "发送验证码失败")
		}
		return
	}

	if h.service.email.IsDevMode() {
		response.Success(c, gin.H{"dev_code": code})
	} else {
		response.Success(c, nil)
	}
}

// VerifyEmail verifies the email with the code.
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "请求格式错误")
		return
	}

	user, err := h.service.VerifyEmail(req.Email, req.Code)
	if err != nil {
		switch err {
		case ErrEmailNotFound:
			response.Error(c, 40000, "邮箱未注册，请先注册")
		case ErrEmailAlreadyVerified:
			response.Error(c, 40000, "邮箱已验证")
		case ErrInvalidVerificationCode:
			response.Error(c, 40000, "验证码错误")
		case ErrVerificationCodeExpired:
			response.Error(c, 40000, "验证码已过期")
		default:
			response.Error(c, 50000, "验证失败")
		}
		return
	}

	response.Success(c, toUserResp(user, h.service.email.IsDevMode()))
}

// CheckUsername returns whether a username is available.
func (h *Handler) CheckUsername(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		response.Error(c, 40000, "缺少参数")
		return
	}

	available, err := h.service.CheckUsername(q)
	if err != nil {
		response.Error(c, 50000, "检查失败")
		return
	}
	response.Success(c, gin.H{"available": available})
}

// CheckEmail returns whether an email is available.
func (h *Handler) CheckEmail(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		response.Error(c, 40000, "缺少参数")
		return
	}

	available, err := h.service.CheckEmail(q)
	if err != nil {
		response.Error(c, 50000, "检查失败")
		return
	}
	response.Success(c, gin.H{"available": available})
}

// Me returns the currently authenticated user.
func (h *Handler) Me(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "未登录")
		return
	}

	user, err := h.service.GetByID(userID)
	if err != nil {
		response.Error(c, 40100, "未登录")
		return
	}

	response.Success(c, toUserResp(user, h.service.email.IsDevMode()))
}

// Logout clears the auth cookie.
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
