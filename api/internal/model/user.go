package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model `json:"-"`
	Email      string     `gorm:"index:idx_email_deleted_at,unique;size:255;not null" json:"email"`
	LastName   string     `gorm:"not null;size:255" json:"last_name"`
	FirstName  string     `gorm:"not null;size:255" json:"first_name"`
	Org        string     `gorm:"not null;size:255" json:"organization"`
	Role       string     `gorm:"type:enum('admin','user','debug');not null" json:"role"`
	Status     string     `gorm:"type:enum('active','inactive');default:'active';not null" json:"status"`
	LastLogin  *time.Time `gorm:"type:timestamp;" json:"last_login"`
	PwdHash    *string    `gorm:"default:null;size:255" json:"-"`
	// Soft delete
	DeletedAt gorm.DeletedAt `gorm:"index:idx_email_deleted_at,unique" json:"-"`
	Orders    []Order        `json:"-"`
}

func (u *User) Save(db *gorm.DB) error {
	return db.Save(u).Error
}

func (u *User) UpdateLastLogin(db *gorm.DB) error {
	now := time.Now()
	return db.Model(u).Updates(User{LastLogin: &now}).Error
}

func CountActiveAdmins(db *gorm.DB) (int64, error) {
	var count int64
	if err := db.Model(&User{}).Where(&User{Role: "admin", Status: "active"}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func GetUserByEmail(db *gorm.DB, email string, joins ...string) (*User, error) {
	for _, join := range joins {
		db = db.Joins(join)
	}

	var user User
	if err := db.Where(&User{Email: email}).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByID(db *gorm.DB, id uint, joins ...string) (*User, error) {
	for _, join := range joins {
		db = db.Joins(join)
	}

	var user User
	if err := db.First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func IsEmailAvailable(mail string, tx *gorm.DB, excludeID uint) (bool, error) {
	var exists bool

	userQuery := tx.Model(&User{}).Select("1").Where(&User{Email: mail})
	if excludeID != 0 {
		userQuery = userQuery.Where("id != ?", excludeID)
	}
	if err := userQuery.Limit(1).Find(&exists).Error; err != nil || exists {
		return false, err
	}

	return true, nil
}
