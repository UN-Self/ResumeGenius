package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(publicRG, protectedRG *gin.RouterGroup, db *gorm.DB, jwtSecret string, tokenTTL time.Duration, cookieSecure bool, emailService *EmailService) {
	svc := NewService(db, emailService)
	h := NewHandler(svc, jwtSecret, tokenTTL, cookieSecure)

	publicRG.POST("/auth/login", h.Login)
	publicRG.POST("/auth/logout", h.Logout)
	publicRG.POST("/auth/register", h.Register)
	publicRG.POST("/auth/send-code", h.SendCode)
	publicRG.POST("/auth/verify-email", h.VerifyEmail)
	publicRG.GET("/auth/check-username", h.CheckUsername)
	publicRG.GET("/auth/check-email", h.CheckEmail)
	protectedRG.GET("/auth/me", h.Me)
}
