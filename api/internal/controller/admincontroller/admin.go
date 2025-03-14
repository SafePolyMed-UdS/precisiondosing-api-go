package admincontroller

import (
	"errors"
	"fmt"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/responder"
	"precisiondosing-api-go/internal/utils/hash"
	"precisiondosing-api-go/internal/utils/tokens"
	"precisiondosing-api-go/internal/utils/validate"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminController struct {
	DB       *gorm.DB
	ResetCfg cfg.ResetTokenConfig
	Mailer   *responder.Mailer
}

func NewAdminController(resourceHandle *handle.ResourceHandle) *AdminController {
	return &AdminController{
		DB:       resourceHandle.Databases.GormDB,
		ResetCfg: resourceHandle.ResetCfg,
		Mailer:   resourceHandle.Mailer,
	}
}

// @Summary		Create a new service user
// @Description	__Admin role required__
// @Description	Create a new service user for the API.
// @Description	You can create users with the following roles: `admin`, `user`, `approver`.
// @Tags			Admin
// @Produce		json
// @Param			request	body		CreateServiceUserQuery							true	"Request body"
// @Success		200		{object}	handle.jsendSuccess[map[string]string]			"User created"
// @Failure		400		{object}	handle.jsendFailure[handle.errorResponse]		"Bad request"
// @Failure		422		{object}	handle.jsendFailure[handle.validationResponse]	"Bad query format"
// @Failure		401		{object}	handle.jsendFailure[handle.errorResponse]		"Unauthorized"
// @Failure		403		{object}	handle.jsendFailure[handle.errorResponse]		"Non-admin user"
// @Failure		500		{object}	handle.jSendError								"Internal server error"
//
// @Security		Bearer
//
// @Router			/admin/users/service [post]
func (ac *AdminController) CreateServiceUser(c *gin.Context) {
	type Query struct {
		Email     string `json:"email" binding:"required,email,min=2,max=255" example:"joe@gmail.com"`
		FirstName string `json:"first_name" binding:"required,min=2,max=255" example:"Joe"`
		LastName  string `json:"last_name" binding:"required,min=2,max=255" example:"Doe"`
		Org       string `json:"organization" binding:"required,min=2,max=255" example:"ACME"`
		Role      string `json:"role" binding:"required,oneof=admin user approver"`
		Password  string `json:"password" binding:"required" example:"password123"`
	} //	@name	CreateServiceUserQuery

	var query Query
	if !handle.JSONBind(c, &query) {
		return
	}

	if err := validate.Email(query.Email); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid email: %s", err))
		return
	}

	if err := validate.Name(query.FirstName); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid first name: %s", err))
		return
	}

	if err := validate.Name(query.LastName); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid last name: %s", err))
		return
	}

	if err := validate.Organization(query.Org); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid organization: %s", err))
		return
	}

	if err := validate.Password(query.Password); err != nil {
		handle.BadRequestError(c, "Invalid password")
		return
	}

	hashedPwd, err := hash.Create(query.Password)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	// create user
	user := model.User{
		Email:     query.Email,
		FirstName: query.FirstName,
		LastName:  query.LastName,
		Org:       query.Org,
		Role:      query.Role,
		Status:    "active",
		PwdHash:   &hashedPwd,
	}

	// check if email is available and create a user +
	if err = ac.DB.Transaction(func(tx *gorm.DB) error {
		mailAvailable, mailErr := model.IsEmailAvailable(query.Email, tx, 0)
		if mailErr != nil {
			handle.ServerError(c, mailErr)
			return gorm.ErrInvalidTransaction
		}

		if !mailAvailable {
			handle.BadRequestError(c, "Email already in use")
			return gorm.ErrInvalidTransaction
		}

		if createErr := tx.Create(&user).Error; createErr != nil {
			handle.ServerError(c, createErr)
			return gorm.ErrInvalidTransaction
		}

		return nil
	}); err != nil {
		return
	}

	handle.Success(c, gin.H{"message": "Service user created"})
}

// @Summary		Create a new user
// @Description	__Admin role required__
// @Description	Create a new user for the API. Ths user will receive an email with a token to set their password.
// @Description	You can create users with the following roles: `admin`, `user`, `approver`.
// @Tags			Admin
// @Produce		json
// @Param			request	body		CreateUserQuery									true	"Request body"
// @Success		200		{object}	handle.jsendSuccess[map[string]string]			"User created"
// @Failure		400		{object}	handle.jsendFailure[handle.errorResponse]		"Bad request"
// @Failure		422		{object}	handle.jsendFailure[handle.validationResponse]	"Bad query format"
// @Failure		401		{object}	handle.jsendFailure[handle.errorResponse]		"Unauthorized"
// @Failure		403		{object}	handle.jsendFailure[handle.errorResponse]		"Non-admin user"
// @Failure		500		{object}	handle.jSendError								"Internal server error"
//
// @Security		Bearer
//
// @Router			/admin/users [post]
func (ac *AdminController) CreateUser(c *gin.Context) {
	type Query struct {
		Email     string `json:"email" binding:"required,email,min=2,max=255" example:"joe@gmail.com"`
		FirstName string `json:"first_name" binding:"required,min=2,max=255" example:"Joe"`
		LastName  string `json:"last_name" binding:"required,min=2,max=255" example:"Doe"`
		Org       string `json:"organization" binding:"required,min=2,max=255" example:"ACME"`
		Role      string `json:"role" binding:"required,oneof=admin user approver"`
	} //	@name	CreateUserQuery

	var query Query
	if !handle.JSONBind(c, &query) {
		return
	}

	if err := validate.Email(query.Email); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid email: %s", err))
		return
	}

	if err := validate.Name(query.FirstName); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid first name: %s", err))
		return
	}

	if err := validate.Name(query.LastName); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid last name: %s", err))
		return
	}

	if err := validate.Organization(query.Org); err != nil {
		handle.BadRequestError(c, fmt.Sprintf("Invalid organization: %s", err))
		return
	}

	// create user
	user := model.User{
		Email:     query.Email,
		FirstName: query.FirstName,
		LastName:  query.LastName,
		Org:       query.Org,
		Role:      query.Role,
		PwdReset:  &model.UserPwdReset{},
	}
	resetTokens, err := tokens.CreateResetTokens()
	if err != nil {
		handle.ServerError(c, err)
		return
	}
	user.PwdReset.ResetTokenHash = resetTokens.TokenHash
	user.PwdReset.TokenExpiry = time.Now().Add(ac.ResetCfg.ExpirationTime)

	// check if email is available and create a user +
	if err = ac.DB.Transaction(func(tx *gorm.DB) error {
		mailAvailable, mailErr := model.IsEmailAvailable(query.Email, tx, 0)
		if mailErr != nil {
			handle.ServerError(c, mailErr)
			return gorm.ErrInvalidTransaction
		}

		if !mailAvailable {
			handle.BadRequestError(c, "Email already in use")
			return gorm.ErrInvalidTransaction
		}

		if createErr := tx.Create(&user).Error; createErr != nil {
			handle.ServerError(c, createErr)
			return gorm.ErrInvalidTransaction
		}

		fullName := fmt.Sprintf("%s %s", query.FirstName, query.LastName)
		mailerErr := ac.Mailer.SendNewAccoundEmail(
			fullName,
			query.Email,
			resetTokens.Token,
			user.PwdReset.TokenExpiry,
		)
		if mailerErr != nil {
			handle.ServerError(c, mailerErr)
			return gorm.ErrInvalidTransaction
		}

		return nil
	}); err != nil {
		return
	}

	handle.Success(c, gin.H{"message": "User created"})
}

// @Summary		Get all users
// @Description	__Admin role required__
// @Description	List all users for the API.
// @Tags			Admin
// @Produce		json
// @Success		200	{object}	handle.jsendSuccess[map[string]string]		"User created"
// @Failure		400	{object}	handle.jsendFailure[handle.errorResponse]	"Bad request"
// @Failure		401	{object}	handle.jsendFailure[handle.errorResponse]	"Unauthorized"
// @Failure		403	{object}	handle.jsendFailure[handle.errorResponse]	"Non-admin user"
// @Failure		500	{object}	handle.jSendError							"Internal server error"
//
// @Security		Bearer
//
// @Router			/admin/users [get]
func (ac *AdminController) GetUsers(c *gin.Context) {
	var query struct {
		Role   string `form:"role" binding:"omitempty,oneof=admin user approver"`
		Status string `form:"status" binding:"omitempty,oneof=active inactive"`
	}

	if !handle.QueryBind(c, &query) {
		return
	}

	db := ac.DB
	if query.Role != "" {
		db = db.Where(&model.User{Role: query.Role})
	}

	if query.Status != "" {
		db = db.Where(&model.User{Status: query.Status})
	}

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	if len(users) == 0 {
		handle.NotFoundError(c, "No users found that match the query")
		return
	}

	handle.Success(c, users)
}

func (ac *AdminController) GetUserByEmail(c *gin.Context) {
	user, err := model.GetUserByEmail(ac.DB, c.Param("email"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "User not found")
			return
		}

		handle.ServerError(c, err)
		return
	}

	handle.Success(c, user)
}

func (ac *AdminController) DeleteUserByEmail(c *gin.Context) {
	emailToDelete := c.Param("email")
	adminEmail := c.GetString("user_email")

	if emailToDelete == adminEmail {
		handle.ForbiddenError(c, "Cannot delete own account")
		return
	}

	user, err := model.GetUserByEmail(ac.DB, emailToDelete)
	if err != nil {
		handle.NotFoundError(c, "User not found")
		return
	}

	if err = ac.DB.Delete(&user).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, gin.H{"message": "User deleted"})
}

func (ac *AdminController) ChangeUserProfile(c *gin.Context) {
	type Query struct {
		Role   string `json:"role" binding:"omitempty,oneof=admin user approver" example:"user"`
		Status string `json:"status" binding:"omitempty,oneof=active inactive" example:"inactive"`
	} //	@name	ChangeUserProfileQuery
	adminID := c.GetUint("user_id")

	var query Query
	if !handle.JSONBind(c, &query) {
		return
	}

	user, err := model.GetUserByEmail(ac.DB, c.Param("email"))
	if err != nil {
		handle.NotFoundError(c, "User not found")
		return
	}

	if query.Role == "" && query.Status == "" {
		handle.BadRequestError(c, "No changes requested")
		return
	}

	if user.ID == adminID {
		adminCount, adminErr := model.CountActiveAdmins(ac.DB)
		if adminErr != nil {
			handle.ServerError(c, err)
			return
		}

		if adminCount == 1 {
			handle.ForbiddenError(c, "Cannot change admin's role or status (only one admin left)")
			return
		}

		handle.ForbiddenError(c, "Cannot change own role or status")
		return
	}

	if query.Role != "" {
		user.Role = query.Role
	}

	if query.Status != "" {
		user.Status = query.Status
	}

	if err = ac.DB.Save(&user).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, gin.H{"message": "User profile updated"})
}
