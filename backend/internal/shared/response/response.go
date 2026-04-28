package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Data:    data,
		Message: "ok",
	})
}

func Error(c *gin.Context, code int, message string) {
	status := httpStatusFromCode(code)
	c.JSON(status, APIResponse{
		Code:    code,
		Data:    nil,
		Message: message,
	})
}

/*
ErrorWithStatus writes an error response using the provided HTTP status.
This is useful for module-specific 4-digit error codes (e.g. 4001) where the
HTTP status can't be derived from the numeric code.
*/
func ErrorWithStatus(c *gin.Context, status int, code int, message string) {
	c.JSON(status, APIResponse{
		Code:    code,
		Data:    nil,
		Message: message,
	})
}

func httpStatusFromCode(code int) int {
	if code >= 50000 {
		return http.StatusInternalServerError
	}
	if code >= 40400 {
		return http.StatusNotFound
	}
	if code >= 40300 {
		return http.StatusForbidden
	}
	if code >= 40100 {
		return http.StatusUnauthorized
	}
	return http.StatusBadRequest
}
