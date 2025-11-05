package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a gin middleware for logging requests
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format
		return fmt.Sprintf("[%s] | %s | %3d | %13v | %15s | %-7s %s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// Recovery returns a gin middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.JSON(500, gin.H{
				"error":     "Internal Server Error",
				"message":   err,
				"timestamp": time.Now().Format(time.RFC3339),
			})
		}
		c.AbortWithStatus(500)
	})
}
