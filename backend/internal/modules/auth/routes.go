package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(publicRG, protectedRG *gin.RouterGroup, db *gorm.DB, jwtSecret string, tokenTTL time.Duration, cookieSecure bool) {
	svc := NewService(db)
	h := NewHandler(svc, jwtSecret, tokenTTL, cookieSecure)

	publicRG.POST("/auth/login", h.Login)
	publicRG.POST("/auth/logout", h.Logout)
	protectedRG.GET("/auth/me", h.Me)
}
