package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"gorm.io/gorm"
)

// UserRepository handles database operations for User
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// FindByID finds a user by UUID
func (r *UserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByGoogleID finds a user by Google OAuth ID
func (r *UserRepository) FindByGoogleID(googleID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("google_id = ?", googleID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// SearchUsers searches users by name or email (partial match)
func (r *UserRepository) SearchUsers(query string, excludeUserID uuid.UUID, limit int) ([]model.User, error) {
	var users []model.User
	err := r.db.
		Where("(name ILIKE ? OR email ILIKE ?) AND id != ?", "%"+query+"%", "%"+query+"%", excludeUserID).
		Limit(limit).
		Find(&users).Error
	return users, err
}

// UpdateOnlineStatus sets a user's online status and last seen time
func (r *UserRepository) UpdateOnlineStatus(id uuid.UUID, isOnline bool) error {
	updates := map[string]interface{}{
		"is_online": isOnline,
	}
	if !isOnline {
		updates["last_seen"] = gorm.Expr("NOW()")
	}
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

// VerifyEmail marks user's email as verified
func (r *UserRepository) VerifyEmail(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.User{}).
		Where("id = ?", userID).
		Update("email_verified_at", now).Error
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(userID uuid.UUID, hashedPassword string) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", userID).
		Update("password", hashedPassword).Error
}

// UpdateAvatar updates a user's avatar URL
func (r *UserRepository) UpdateAvatar(userID uuid.UUID, avatarURL string) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", userID).
		Update("avatar", avatarURL).Error
}
