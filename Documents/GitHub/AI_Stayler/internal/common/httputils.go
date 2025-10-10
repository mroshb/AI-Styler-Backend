package common

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, statusCode int, code, message string, details interface{}) {
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if details != nil {
		errorResponse["error"].(map[string]interface{})["details"] = details
	}

	WriteJSON(w, statusCode, errorResponse)
}

// GinWrap adapts http.HandlerFunc to gin.HandlerFunc
func GinWrap(fn func(w http.ResponseWriter, r *http.Request)) gin.HandlerFunc {
	return func(c *gin.Context) {
		fn(c.Writer, c.Request)
	}
}

// Gin context helpers - these are gin-specific versions of the context functions
