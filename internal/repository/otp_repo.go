package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"gorm.io/gorm"
)

// OTPRepository handles database operations for OTP codes
type OTPRepository struct {
	db *gorm.DB
}

func NewOTPRepository(db *gorm.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

// Create inserts a new OTP code
func (r *OTPRepository) Create(otp *model.OTPCode) error {
	return r.db.Create(otp).Error
}

// FindValidOTP finds an unused, non-expired OTP code for a user and purpose
func (r *OTPRepository) FindValidOTP(userID uuid.UUID, code string, purpose model.OTPPurpose) (*model.OTPCode, error) {
	var otp model.OTPCode
	err := r.db.
		Where("user_id = ? AND code = ? AND purpose = ? AND expires_at > ? AND used_at IS NULL",
			userID, code, purpose, time.Now()).
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// MarkAsUsed marks an OTP code as used
func (r *OTPRepository) MarkAsUsed(otpID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.OTPCode{}).
		Where("id = ?", otpID).
		Update("used_at", now).Error
}

// InvalidateAllForUser invalidates all pending OTPs for a user and purpose
// (useful when sending a new code - old ones should be invalidated)
func (r *OTPRepository) InvalidateAllForUser(userID uuid.UUID, purpose model.OTPPurpose) error {
	now := time.Now()
	return r.db.Model(&model.OTPCode{}).
		Where("user_id = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?",
			userID, purpose, time.Now()).
		Update("used_at", now).Error
}

// CleanupExpired removes all expired OTP codes (housekeeping)
func (r *OTPRepository) CleanupExpired() error {
	return r.db.
		Where("expires_at < ? AND used_at IS NULL", time.Now()).
		Delete(&model.OTPCode{}).Error
}

// CountRecentOTPs counts how many OTPs were sent to a user recently (rate limiting)
func (r *OTPRepository) CountRecentOTPs(userID uuid.UUID, purpose model.OTPPurpose, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.OTPCode{}).
		Where("user_id = ? AND purpose = ? AND created_at > ?", userID, purpose, since).
		Count(&count).Error
	return count, err
}
