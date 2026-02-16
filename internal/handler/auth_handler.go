package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/quocanhngo/gotalk/internal/service"
	"github.com/quocanhngo/gotalk/pkg/storage"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *service.AuthService
	storage     storage.Storage
}

func NewAuthHandler(authService *service.AuthService, storage storage.Storage) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		storage:     storage,
	}
}

// Register godoc
// @Summary Register a new user (sends OTP for verification)
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body model.RegisterRequest true "Register request"
// @Success 201 {object} model.OTPSentResponse
// @Failure 400 {object} model.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	resp, err := h.authService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// VerifyOTP godoc
// @Summary Verify email with OTP code
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body model.VerifyOTPRequest true "Verify OTP request"
// @Success 200 {object} model.AuthResponse
// @Failure 400 {object} model.ErrorResponse
// @Router /auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req model.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	resp, err := h.authService.VerifyOTP(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ResendOTP godoc
// @Summary Resend OTP verification code
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body model.ResendOTPRequest true "Resend OTP request"
// @Success 200 {object} model.OTPSentResponse
// @Failure 400 {object} model.ErrorResponse
// @Router /auth/resend-otp [post]
func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req model.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	resp, err := h.authService.ResendOTP(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Login godoc
// @Summary Login with email and password
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body model.LoginRequest true "Login request"
// @Success 200 {object} model.AuthResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	resp, err := h.authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GoogleLogin godoc
// @Summary Login with Google OAuth2
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body model.GoogleLoginRequest true "Google login request"
// @Success 200 {object} model.LoginResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/google [post]
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req model.GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	resp, err := h.authService.LoginWithGoogle(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ForgotPassword godoc
// @Summary Request password reset OTP
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body model.ForgotPasswordRequest true "Forgot password request"
// @Success 200 {object} model.OTPSentResponse
// @Router /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req model.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	resp, err := h.authService.ForgotPassword(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ResetPassword godoc
// @Summary Reset password with OTP code
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body model.ResetPasswordRequest true "Reset password request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req model.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	if err := h.authService.ResetPassword(req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{Message: "Password reset successfully"})
}

// GetProfile godoc
// @Summary Get current user profile
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.UserResponse
// @Router /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	profile, err := h.authService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// SearchUsers godoc
// @Summary Search users by username or email
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Success 200 {array} model.UserResponse
// @Router /users/search [get]
func (h *AuthHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Search query is required"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	users, err := h.authService.SearchUsers(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to search users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// Logout godoc
// @Summary Logout
// @Description Invalidate current token and set user offline
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.SuccessResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "Token required"})
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "Invalid token format"})
		return
	}
	tokenString := parts[1]

	if err := h.authService.Logout(userID, tokenString); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{Message: "Logged out successfully"})
}

// UpdateProfile godoc
// @Summary Update user profile
// @Tags Auth
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string false "User name"
// @Param avatar formData file false "Avatar image file"
// @Success 200 {object} model.UserResponse
// @Router /auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid form data", Message: err.Error()})
		return
	}

	req := model.UpdateProfileRequest{}

	// Get name from form
	if names := form.Value["name"]; len(names) > 0 {
		req.Name = names[0]
	}

	// Handle avatar file upload
	if files := form.File["avatar"]; len(files) > 0 {
		fileHeader := files[0]

		// Open the file
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Failed to read file", Message: err.Error()})
			return
		}
		defer file.Close()

		// Upload to MinIO
		if h.storage != nil {
			result, err := h.storage.Upload(c.Request.Context(), file, fileHeader, "avatars")
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to upload avatar", Message: err.Error()})
				return
			}
			req.Avatar = result.URL
		} else {
			c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{Error: "File upload service unavailable"})
			return
		}
	}

	user, err := h.authService.UpdateProfile(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateSettings godoc
// @Summary Update user settings
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.UpdateSettingsRequest true "Update settings request"
// @Success 200 {object} model.UserResponse
// @Router /auth/settings [put]
func (h *AuthHandler) UpdateSettings(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var req model.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	user, err := h.authService.UpdateSettings(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetSettings godoc
// @Summary Get user settings
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.UserResponse
// @Router /auth/settings [get]
func (h *AuthHandler) GetSettings(c *gin.Context) {
	h.GetProfile(c)
}

// RegisterDevice godoc
// @Summary Register device for push notifications
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.RegisterDeviceRequest true "Register device request"
// @Success 200 {object} model.SuccessResponse
// @Router /auth/device [post]
func (h *AuthHandler) RegisterDevice(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var req model.RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	if err := h.authService.RegisterDevice(userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{Message: "Device registered successfully"})
}
