package database

import (
	"errors"
	"fmt"
	"os"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/utils/hash"
	"precisiondosing-api-go/internal/utils/validate"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	db.Set("gorm:table_options", "ENGINE=InnoDB")

	if err := db.AutoMigrate(&model.User{}, &model.UserEmailChange{}, &model.UserPwdReset{}); err != nil {
		return fmt.Errorf("migrate user models: %w", err)
	}

	// Seed database with default admin user if no active admin user exists
	if err := seed(db); err != nil {
		return fmt.Errorf("seed user database: %w", err)
	}

	if err := db.AutoMigrate(&model.Order{}); err != nil {
		return fmt.Errorf("migrate order model: %w", err)
	}

	return nil
}

func seed(db *gorm.DB) error {
	count, err := model.CountActiveAdmins(db)
	if err != nil {
		return fmt.Errorf("count active admins: %w", err)
	}

	if count == 0 {
		adminEmail := os.Getenv("ADMIN_EMAIL")
		if adminEmail == "" {
			return errors.New("env ADMIN_EMAIL not set")
		}

		adminPWD := os.Getenv("ADMIN_PASSWORD")
		if adminPWD == "" {
			return errors.New("env ADMIN_PASSWORD not set")
		}

		if err = seedAdminUser(db, adminEmail, adminPWD); err != nil {
			return fmt.Errorf("seed default admin account: %w", err)
		}
	}

	return nil
}

func seedAdminUser(db *gorm.DB, email string, pwd string) error {
	if err := validate.Email(email); err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}

	if err := validate.Password(pwd); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	pwd, err := hash.Create(pwd)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Email:     email,
		LastName:  "Admin",
		FirstName: "Admin",
		Org:       "Admin",
		Role:      "admin",
		Status:    "active",
		PwdHash:   &pwd,
	}

	if err = user.Save(db); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	return nil
}
