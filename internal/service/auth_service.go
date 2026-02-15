package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/quocanhngo/gotalk/internal/repository"
	"github.com/quocanhngo/gotalk/pkg/auth"
	"github.com/quocanhngo/gotalk/pkg/mailer"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	otpLength        = 6
	otpExpiryMinutes = 5
	otpRateLimit     = 3               // max OTPs per hour
	googleTokenURL   = "https://oauth2.googleapis.com/tokeninfo?id_token="
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo       *repository.UserRepository
	otpRepo        *repository.OTPRepository
	jwtManager     *auth.JWTManager
	mailer         *mailer.Mailer
	rdb            *redis.Client
	googleClientID string
}

func NewAuthService(
	userRepo *repository.UserRepository,
	otpRepo *repository.OTPRepository,
	jwtManager *auth.JWTManager,
	mailer *mailer.Mailer,
	rdb *redis.Client,
	googleClientID string,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		otpRepo:        otpRepo,
		jwtManager:     jwtManager,
		mailer:         mailer,
		rdb:            rdb,
		googleClientID: googleClientID,
	}
}

// ==================== Register (Email + OTP) ====================

// Register creates a new unverified user account and sends OTP
func (s *AuthService) Register(req model.RegisterRequest) (*model.OTPSentResponse, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err == nil {
		// Email exists
		if existingUser.IsEmailVerified() {
			return nil, errors.New("email already registered")
		}
		// User registered but never verified - resend OTP
		return s.sendOTP(existingUser, model.OTPPurposeEmailVerification)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	user := &model.User{
		Name:         req.Name,
		Email:        req.Email,
		Password:     string(hashedPassword),
		AuthProvider: model.AuthProviderEmail,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("failed to create user")
	}

	// Send OTP email
	return s.sendOTP(user, model.OTPPurposeEmailVerification)
}

// VerifyOTP verifies an OTP code and activates the account
func (s *AuthService) VerifyOTP(req model.VerifyOTPRequest) (*model.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Find valid OTP
	otp, err := s.otpRepo.FindValidOTP(user.ID, req.Code, model.OTPPurposeEmailVerification)
	if err != nil {
		return nil, errors.New("invalid or expired OTP code")
	}

	// Mark OTP as used
	if err := s.otpRepo.MarkAsUsed(otp.ID); err != nil {
		return nil, errors.New("failed to verify OTP")
	}

	// Verify user's email
	if err := s.userRepo.VerifyEmail(user.ID); err != nil {
		return nil, errors.New("failed to verify email")
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Name)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// Refresh user data
	user, _ = s.userRepo.FindByID(user.ID)

	return &model.AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

// ResendOTP generates and sends a new OTP code
func (s *AuthService) ResendOTP(req model.ResendOTPRequest) (*model.OTPSentResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if user.IsEmailVerified() {
		return nil, errors.New("email already verified")
	}

	return s.sendOTP(user, model.OTPPurposeEmailVerification)
}

// ==================== Login (Email/Password) ====================

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, errors.New("failed to find user")
	}

	// Check if user registered with Google (no password set)
	if user.AuthProvider == model.AuthProviderGoogle {
		return nil, errors.New("this account uses Google login. Please sign in with Google")
	}

	// Check if email is verified
	if !user.IsEmailVerified() {
		return nil, errors.New("email not verified. Please check your inbox for the verification code")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Name)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &model.AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

// ==================== Login (Google OAuth2) ====================

// GoogleLogin authenticates via Google ID token
func (s *AuthService) GoogleLogin(req model.GoogleLoginRequest) (*model.AuthResponse, error) {
	// Verify Google ID token
	googleUser, err := s.verifyGoogleToken(req.IDToken)
	if err != nil {
		return nil, fmt.Errorf("invalid Google token: %w", err)
	}

	// Check if user exists with this Google ID
	user, err := s.userRepo.FindByGoogleID(googleUser.GoogleID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("failed to find user")
		}

		// Check if email is already registered (with email provider)
		existingUser, err := s.userRepo.FindByEmail(googleUser.Email)
		if err == nil {
			// Link Google ID to existing account
			existingUser.GoogleID = &googleUser.GoogleID
			existingUser.Name = googleUser.Name // Update name from Google
			existingUser.AuthProvider = model.AuthProviderGoogle
			if existingUser.Avatar == "" && googleUser.Picture != "" {
				existingUser.Avatar = googleUser.Picture
			}
			now := time.Now()
			existingUser.EmailVerifiedAt = &now
			if err := s.userRepo.Create(existingUser); err != nil {
				// Update instead
				s.userRepo.VerifyEmail(existingUser.ID)
			}
			user = existingUser
		} else {
			// Create new user from Google info
			now := time.Now()
			user = &model.User{
				Name:            googleUser.Name,
				Email:           googleUser.Email,
				Avatar:          googleUser.Picture,
				AuthProvider:    model.AuthProviderGoogle,
				GoogleID:        &googleUser.GoogleID,
				EmailVerifiedAt: &now, // Google emails are pre-verified
			}
			if err := s.userRepo.Create(user); err != nil {
				return nil, errors.New("failed to create user")
			}
		}
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Name)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &model.AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

// ==================== Forgot/Reset Password ====================

// ForgotPassword sends a password reset OTP
func (s *AuthService) ForgotPassword(req model.ForgotPasswordRequest) (*model.OTPSentResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		// Don't reveal if email exists or not
		return &model.OTPSentResponse{
			Message:   "If the email exists, a reset code has been sent",
			Email:     req.Email,
			ExpiresIn: otpExpiryMinutes * 60,
		}, nil
	}

	if user.AuthProvider == model.AuthProviderGoogle {
		return nil, errors.New("this account uses Google login. Password reset is not available")
	}

	return s.sendOTP(user, model.OTPPurposePasswordReset)
}

// ResetPassword verifies OTP and sets a new password
func (s *AuthService) ResetPassword(req model.ResetPasswordRequest) error {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return errors.New("user not found")
	}

	// Find valid OTP
	otp, err := s.otpRepo.FindValidOTP(user.ID, req.Code, model.OTPPurposePasswordReset)
	if err != nil {
		return errors.New("invalid or expired reset code")
	}

	// Mark OTP as used
	if err := s.otpRepo.MarkAsUsed(otp.ID); err != nil {
		return errors.New("failed to process reset code")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	return s.userRepo.UpdatePassword(user.ID, string(hashedPassword))
}

// ==================== Profile ====================

// GetProfile returns the current user's profile
func (s *AuthService) GetProfile(userID uuid.UUID) (*model.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	resp := user.ToResponse()
	return &resp, nil
}

// SearchUsers searches for users by name or email
func (s *AuthService) SearchUsers(query string, excludeUserID uuid.UUID) ([]model.UserResponse, error) {
	users, err := s.userRepo.SearchUsers(query, excludeUserID, 20)
	if err != nil {
		return nil, err
	}

	var result []model.UserResponse
	for _, u := range users {
		result = append(result, u.ToResponse())
	}
	return result, nil
}

// Logout invalidates the token and sets user offline
func (s *AuthService) Logout(userID uuid.UUID, tokenString string) error {
	// 1. Set offline
	if err := s.userRepo.UpdateOnlineStatus(userID, false); err != nil {
		return err
	}

	// 2. Parse token to get expiry
	claims, err := s.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return err
	}

	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn <= 0 {
		return nil
	}

	// 3. Blacklist token
	return s.rdb.Set(context.Background(), "blacklist:"+tokenString, "revoked", expiresIn).Err()
}

// ==================== Internal Helpers ====================

// sendOTP generates a code, saves it, and emails it
func (s *AuthService) sendOTP(user *model.User, purpose model.OTPPurpose) (*model.OTPSentResponse, error) {
	time.Sleep(1 * time.Second) // Small delay to prevent race conditions in tests if any
	// Rate limiting: max 3 OTPs per hour
	count, _ := s.otpRepo.CountRecentOTPs(user.ID, purpose, time.Now().Add(-1*time.Hour))
	if count >= int64(otpRateLimit) {
		return nil, errors.New("too many OTP requests. Please try again later")
	}

	// Invalidate old OTPs
	_ = s.otpRepo.InvalidateAllForUser(user.ID, purpose)

	// Generate 6-digit code
	code, err := generateOTPCode(otpLength)
	if err != nil {
		return nil, errors.New("failed to generate OTP code")
	}

	// Save OTP to database
	otp := &model.OTPCode{
		UserID:    user.ID,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(time.Duration(otpExpiryMinutes) * time.Minute),
	}
	if err := s.otpRepo.Create(otp); err != nil {
		return nil, errors.New("failed to save OTP")
	}

	// Send email asynchronously
	go func() {
		var emailErr error
		switch purpose {
		case model.OTPPurposeEmailVerification:
			// Used Name instead of Username
			emailErr = s.mailer.SendOTP(user.Email, user.Name, code, otpExpiryMinutes)
		case model.OTPPurposePasswordReset:
			emailErr = s.mailer.SendPasswordReset(user.Email, user.Name, code, otpExpiryMinutes)
		}
		if emailErr != nil {
			fmt.Printf("❌ Failed to send email: %v\n", emailErr)
		}
	}()

	return &model.OTPSentResponse{
		Message:   "Verification code sent to your email",
		Email:     user.Email,
		ExpiresIn: otpExpiryMinutes * 60,
	}, nil
}

// generateOTPCode generates a cryptographically secure random numeric code
func generateOTPCode(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}

// verifyGoogleToken validates a Google ID token and extracts user info
func (s *AuthService) verifyGoogleToken(idToken string) (*model.GoogleUserInfo, error) {
	resp, err := http.Get(googleTokenURL + idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google token verification failed: %s", string(body))
	}

	var tokenInfo struct {
		Aud           string `json:"aud"`
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to parse token info: %w", err)
	}

	// Verify the token was issued for our app
	if s.googleClientID != "" && tokenInfo.Aud != s.googleClientID {
		return nil, errors.New("token was not issued for this application")
	}

	return &model.GoogleUserInfo{
		GoogleID: tokenInfo.Sub,
		Email:    tokenInfo.Email,
		Name:     tokenInfo.Name,
		Picture:  tokenInfo.Picture,
		Verified: tokenInfo.EmailVerified == "true",
	}, nil
}
