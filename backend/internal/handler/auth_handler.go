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
	env       string
}

func NewAuthHandler(queries *sqlc.Queries, jwtSecret, env string) *AuthHandler {
	return &AuthHandler{queries: queries, jwtSecret: jwtSecret, env: env}
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// TODO: Google OAuth authorization code -> access token -> userinfo
	c.JSON(http.StatusOK, gin.H{"message": "OAuth callback placeholder"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"userId": userID})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	secure := h.env == "production"
	c.SetCookie("nexus_token", "", -1, "/", "", secure, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
