package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, gin.H{"id": 1})

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != float64(0) {
		t.Errorf("expected code 0, got %v", body["code"])
	}
	if body["message"] != "ok" {
		t.Errorf("expected message ok, got %v", body["message"])
	}
}

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 40001, "参数错误")

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != float64(40001) {
		t.Errorf("expected code 40001, got %v", body["code"])
	}
}

func TestErrorServerCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 50001, "内部错误")

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestErrorNotFoundCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 40401, "未找到")

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
