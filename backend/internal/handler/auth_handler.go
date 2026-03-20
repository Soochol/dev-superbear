package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

type AuthHandler struct {
	queries   *sqlc.Queries
	jwtSecret string
}

func NewAuthHandler(queries *sqlc.Queries, jwtSecret string) *AuthHandler {
	return &AuthHandler{queries: queries, jwtSecret: jwtSecret}
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// TODO: Google OAuth authorization code -> access token -> userinfo
	c.JSON(http.StatusOK, gin.H{"message": "OAuth callback placeholder"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"userId": userID})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("nexus_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
