package workbench

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	service := NewDraftService(db)
	handler := NewHandler(service)

	rg.GET("/drafts/:draft_id", handler.GetDraft)
	rg.PUT("/drafts/:draft_id", handler.UpdateDraft)
}
