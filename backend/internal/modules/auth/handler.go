package auth

import (
	"bytes"
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
	cookieDomain string
	uploadDir    string
}

func NewHandler(service *Service, secret string, ttl time.Duration, cookieSecure bool, cookieDomain string, uploadDir string) *Handler {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return &Handler{service: service, secret: secret, ttl: ttl, cookieSecure: cookieSecure, cookieDomain: cookieDomain, uploadDir: uploadDir}
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
	AvatarURL     string `json:"avatar_url,omitempty"`
	Points        int    `json:"points"`
	DevCode       string `json:"dev_code,omitempty"`
}

func toUserResp(u *models.User, devMode bool) userResp {
	r := userResp{ID: u.ID, Username: u.Username, Points: u.Points}
	if u.Email != nil {
		r.Email = *u.Email
	}
	r.EmailVerified = u.EmailVerified
	if u.AvatarURL != nil {
		r.AvatarURL = *u.AvatarURL
	}
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
		h.cookieDomain,
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

// UpdateProfile updates the user's display name.
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "未登录")
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "请求格式错误")
		return
	}
	user, err := h.service.UpdateProfile(userID, req.Nickname)
	if err != nil {
		response.Error(c, 40000, "昵称需 2-32 个字符")
		return
	}
	response.Success(c, toUserResp(user, h.service.email.IsDevMode()))
}

// ChangePassword changes the user's password.
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "未登录")
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 40000, "请求格式错误")
		return
	}
	if err := h.service.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		switch err {
		case ErrInvalidPassword:
			response.Error(c, 40000, "新密码需 6-128 个字符")
		case ErrInvalidCredentials:
			response.Error(c, 40000, "原密码错误")
		default:
			response.Error(c, 50000, "修改失败")
		}
		return
	}
	response.Success(c, nil)
}

// Logout clears the auth cookie.
func (h *Handler) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.cookieDomain,
		MaxAge:   -1,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	response.Success(c, nil)
}

// UploadAvatar handles avatar image upload with server-side compression.
func (h *Handler) UploadAvatar(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "未登录")
		return
	}

	file, _, err := c.Request.FormFile("avatar")
	if err != nil {
		response.Error(c, 40000, "请选择图片文件")
		return
	}
	defer file.Close()

	// Validate: max 5MB
	const maxSize = 5 << 20
	buf := make([]byte, maxSize+1)
	n, _ := file.Read(buf)
	if n > maxSize {
		response.Error(c, 40000, "图片不能超过 5MB")
		return
	}

	user, err := h.service.UpdateAvatar(userID, bytes.NewReader(buf[:n]), h.uploadDir)
	if err != nil {
		response.Error(c, 40000, "图片格式不支持或处理失败")
		return
	}
	response.Success(c, toUserResp(user, h.service.email.IsDevMode()))
}

// ServeAvatar serves the avatar file for a given user.
func (h *Handler) ServeAvatar(c *gin.Context) {
	userID := c.Param("user_id")
	filePath := h.service.GetAvatarPath(userID, h.uploadDir)
	if filePath == "" {
		c.Header("Content-Type", "image/svg+xml")
		c.String(http.StatusOK, `<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256"><rect fill="%23e2e8f0" width="256" height="256"/><circle cx="128" cy="100" r="48" fill="%2394a3b8"/><ellipse cx="128" cy="220" rx="80" ry="60" fill="%2394a3b8"/></svg>`)
		return
	}
	c.File(filePath)
}

// GetPointsRecords returns the user's points transaction history.
func (h *Handler) GetPointsRecords(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "未登录")
		return
	}

	records, err := h.service.GetPointsRecords(userID, 50)
	if err != nil {
		response.Error(c, 50000, "获取积分记录失败")
		return
	}
	response.Success(c, gin.H{"items": records})
}

// GetPointsStats returns the user's points summary statistics.
func (h *Handler) GetPointsStats(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "未登录")
		return
	}

	stats, err := h.service.GetPointsStats(userID)
	if err != nil {
		response.Error(c, 50000, "获取积分统计失败")
		return
	}
	response.Success(c, stats)
}

// GetPointsDashboard returns full dashboard data with charts.
func (h *Handler) GetPointsDashboard(c *gin.Context) {
	userID := middleware.UserIDFromContext(c)
	if userID == "" {
		response.Error(c, 40100, "未登录")
		return
	}

	dashboard, err := h.service.GetDashboard(userID)
	if err != nil {
		response.Error(c, 50000, "获取仪表盘数据失败")
		return
	}
	response.Success(c, dashboard)
}
