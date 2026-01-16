package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"inspection-tool/apps/cmdb-server/internal/repository"
	"inspection-tool/apps/cmdb-server/internal/service/auth"
)

type AuthHandler struct {
	authService *auth.AuthService
	userRepo    repository.UserRepository
}

func NewAuthHandler(authService *auth.AuthService, userRepo repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userRepo:    userRepo,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LoginUnifiedResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    *LoginData `json:"data"`
}

type LoginData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	tokenPair, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		msg := "authentication failed"

		if errors.Is(err, auth.ErrUserNotFound) {
			msg = "user not found"
		} else if errors.Is(err, auth.ErrUserDisabled) {
			msg = "user account is disabled"
		} else if errors.Is(err, auth.ErrInvalidCredentials) {
			msg = "invalid username or password"
		}

		c.JSON(status, gin.H{
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusOK, LoginUnifiedResponse{
		Code:    0,
		Message: "success",
		Data: &LoginData{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresAt:    tokenPair.ExpiresAt.Unix(),
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
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		status := http.StatusUnauthorized
		msg := "token refresh failed"

		if errors.Is(err, auth.ErrTokenExpired) {
			msg = "refresh token has expired"
		} else if errors.Is(err, auth.ErrInvalidToken) {
			msg = "invalid refresh token"
		} else if errors.Is(err, auth.ErrUserDisabled) {
			msg = "user account is disabled"
		}

		c.JSON(status, gin.H{
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		TokenType:    tokenPair.TokenType,
	})
}

type UserInfoResponse struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Roles       []RoleInfo `json:"roles"`
}

type CurrentUserResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *UserInfoData `json:"data"`
}

type UserInfoData struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Roles       []RoleInfo `json:"roles"`
}

type AccessCodesResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []string `json:"data"`
}

type RoleInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	token, err := extractToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40100,
			"message": err.Error(),
		})
		return
	}

	claims, err := h.authService.ValidateToken(token)
	if err != nil {
		statusCode := http.StatusUnauthorized
		errorCode := 40101
		message := "invalid token"

		if errors.Is(err, auth.ErrTokenExpired) {
			errorCode = 40102
			message = "token expired"
		}

		c.JSON(statusCode, gin.H{
			"code":    errorCode,
			"message": message,
		})
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40400,
			"message": "user not found",
		})
		return
	}

	roles := make([]RoleInfo, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, RoleInfo{
			ID:   role.ID,
			Name: role.Name,
		})
	}

	c.JSON(http.StatusOK, CurrentUserResponse{
		Code:    0,
		Message: "success",
		Data: &UserInfoData{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Roles:       roles,
		},
	})
}

func (h *AuthHandler) GetAccessCodes(c *gin.Context) {
	token, err := extractToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40100,
			"message": err.Error(),
		})
		return
	}

	claims, err := h.authService.ValidateToken(token)
	if err != nil {
		statusCode := http.StatusUnauthorized
		errorCode := 40101
		message := "invalid token"

		if errors.Is(err, auth.ErrTokenExpired) {
			errorCode = 40102
			message = "token expired"
		}

		c.JSON(statusCode, gin.H{
			"code":    errorCode,
			"message": message,
		})
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40400,
			"message": "user not found",
		})
		return
	}

	codes := make([]string, 0)
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			codes = append(codes, perm.Name)
		}
	}

	c.JSON(http.StatusOK, AccessCodesResponse{
		Code:    0,
		Message: "success",
		Data:    codes,
	})
}

func extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("invalid authorization format")
	}

	return authHeader[len(bearerPrefix):], nil
}
