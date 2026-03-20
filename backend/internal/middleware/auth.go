package middleware

import "github.com/gin-gonic/gin"

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Task 5에서 구현
		c.Next()
	}
}
