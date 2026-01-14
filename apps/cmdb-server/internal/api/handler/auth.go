package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/service/auth"
)

type AuthHandler struct {
	authService *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    *AuthData `json:"data,omitempty"`
}

type AuthData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	tokenPair, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		code := 40100
		status := http.StatusUnauthorized
		msg := "authentication failed"

		if errors.Is(err, auth.ErrUserNotFound) {
			msg = "user not found"
		} else if errors.Is(err, auth.ErrUserDisabled) {
			msg = "user account is disabled"
		} else if errors.Is(err, auth.ErrInvalidCredentials) {
			msg = "invalid username or password"
		}

		c.JSON(status, LoginResponse{
			Code:    code,
			Message: msg,
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Code:    0,
		Message: "success",
		Data: &AuthData{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresAt:    tokenPair.ExpiresAt.Format(time.RFC3339),
			TokenType:    tokenPair.TokenType,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "logout successful",
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{
			Code:    40001,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		code := 40101
		status := http.StatusUnauthorized
		msg := "token refresh failed"

		if errors.Is(err, auth.ErrTokenExpired) {
			code = 40102
			msg = "refresh token has expired"
		} else if errors.Is(err, auth.ErrInvalidToken) {
			msg = "invalid refresh token"
		} else if errors.Is(err, auth.ErrUserDisabled) {
			msg = "user account is disabled"
		}

		c.JSON(status, LoginResponse{
			Code:    code,
			Message: msg,
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Code:    0,
		Message: "success",
		Data: &AuthData{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresAt:    tokenPair.ExpiresAt.Format(time.RFC3339),
			TokenType:    tokenPair.TokenType,
		},
	})
}
