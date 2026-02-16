package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// UpdateProfile updates user's name and/or avatar
func (r *UserRepository) UpdateProfile(userID uuid.UUID, name, avatar string) error {
	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	return r.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

// UpdateSettings updates user settings
func (r *UserRepository) UpdateSettings(userID uuid.UUID, theme string, notifEnabled *bool, soundEnabled *bool, lang string) error {
	updates := map[string]interface{}{}
	if theme != "" {
		updates["theme"] = theme
	}
	if notifEnabled != nil {
		updates["is_notification_enabled"] = *notifEnabled
	}
	if soundEnabled != nil {
		updates["is_sound_enabled"] = *soundEnabled
	}
	if lang != "" {
		updates["language"] = lang
	}
	return r.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

// AddDevice adds or updates a device token
func (r *UserRepository) AddDevice(userID uuid.UUID, token string, deviceType string) error {
	device := model.UserDevice{
		UserID:       userID,
		FCMToken:     token,
		DeviceType:   deviceType,
		LastActiveAt: time.Now(),
	}
	// Upsert: on conflict do update
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "fcm_token"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"last_active_at": time.Now(),
			"device_type":    deviceType,
		}),
	}).Create(&device).Error
}

// GetUserDevices gets all devices for a user
func (r *UserRepository) GetUserDevices(userID uuid.UUID) ([]model.UserDevice, error) {
	var devices []model.UserDevice
	err := r.db.Where("user_id = ?", userID).Find(&devices).Error
	return devices, err
}

// GetOrCreateGoogleUser finds a user by email/google_id or creates a new one
func (r *UserRepository) GetOrCreateGoogleUser(userInfo model.GoogleUserInfo) (*model.User, error) {
	var user model.User

	// Check by email first
	if err := r.db.Where("email = ?", userInfo.Email).First(&user).Error; err == nil {
		// User exists
		updates := map[string]interface{}{}

		// If GoogleID is missing, update it
		if user.GoogleID == nil {
			id := userInfo.GoogleID
			updates["google_id"] = &id
			updates["auth_provider"] = "google"

			// Mark email as verified if not
			if !user.IsEmailVerified() && userInfo.Verified {
				now := time.Now()
				updates["email_verified_at"] = &now
			}
		} else if *user.GoogleID != userInfo.GoogleID {
			// Update GoogleID if different? usually shouldn't happen for same email
			id := userInfo.GoogleID
			updates["google_id"] = &id
		}

		// Update avatar if missing or empty
		if user.Avatar == "" && userInfo.Picture != "" {
			updates["avatar"] = userInfo.Picture
		}

		if len(updates) > 0 {
			if err := r.db.Model(&user).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
		return &user, nil
	}

	// User not found, create new one
	googleID := userInfo.GoogleID

	now := time.Now()
	verifiedAt := &now
	if !userInfo.Verified {
		verifiedAt = nil
	}

	newUser := model.User{
		Email:                 userInfo.Email,
		Name:                  userInfo.Name,
		Avatar:                userInfo.Picture,
		GoogleID:              &googleID,
		AuthProvider:          "google",
		EmailVerifiedAt:       verifiedAt,
		Theme:                 "system",
		IsNotificationEnabled: true,
		Language:              "vi",
	}

	if err := r.db.Create(&newUser).Error; err != nil {
		return nil, err
	}

	return &newUser, nil
}
