package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(publicRG, protectedRG *gin.RouterGroup, db *gorm.DB, jwtSecret string, tokenTTL time.Duration, cookieSecure bool, cookieDomain string, emailService *EmailService, uploadDir string) {
	svc := NewService(db, emailService)
	h := NewHandler(svc, jwtSecret, tokenTTL, cookieSecure, cookieDomain, uploadDir)

	publicRG.POST("/auth/login", h.Login)
	publicRG.POST("/auth/logout", h.Logout)
	publicRG.POST("/auth/register", h.Register)
	publicRG.POST("/auth/send-code", h.SendCode)
	publicRG.POST("/auth/verify-email", h.VerifyEmail)
	publicRG.GET("/auth/check-username", h.CheckUsername)
	publicRG.GET("/auth/check-email", h.CheckEmail)
	publicRG.GET("/auth/avatar/:user_id", h.ServeAvatar)
	protectedRG.GET("/auth/me", h.Me)
	protectedRG.PUT("/auth/profile", h.UpdateProfile)
	protectedRG.PUT("/auth/password", h.ChangePassword)
	protectedRG.POST("/auth/avatar", h.UploadAvatar)
	protectedRG.GET("/auth/points/records", h.GetPointsRecords)
	protectedRG.GET("/auth/points/stats", h.GetPointsStats)
	protectedRG.GET("/auth/points/dashboard", h.GetPointsDashboard)
}
