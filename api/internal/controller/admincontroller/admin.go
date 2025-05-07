package admincontroller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/utils/hash"
	"precisiondosing-api-go/internal/utils/validate"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminController struct {
	DB *gorm.DB
}

func New(resourceHandle *handle.ResourceHandle) *AdminController {
	return &AdminController{
		DB: resourceHandle.Databases.GormDB,
	}
}

func (ac *AdminController) DownloadPDF(c *gin.Context) {
	orderID := c.Param("orderId")

	var order model.Order
	if err := ac.DB.
		Select("order_id", "process_result_pdf").
		Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	if order.ProcessResultPDF == nil {
		handle.NotFoundError(c, "No PDF attached for this order")
		return
	}

	pdfBytes, err := base64.StdEncoding.DecodeString(*order.ProcessResultPDF)
	if err != nil {
		handle.ServerError(c, fmt.Errorf("failed to decode PDF: %w", err))
		return
	}

	// Send the file
	c.Writer.WriteHeader(http.StatusOK)
	if _, err = c.Writer.Write(pdfBytes); err != nil {
		handle.ServerError(c, fmt.Errorf("failed to write PDF to response: %w", err))
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"order_%s.pdf\"", order.OrderID))
	c.Header("Content-Length", strconv.Itoa(len(pdfBytes)))
}

func (ac *AdminController) DownloadOrder(c *gin.Context) {
	orderID := c.Param("orderId")

	var order model.Order
	if err := ac.DB.Select("order_id", "order_data").
		Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, order.OrderData)
}

func (ac *AdminController) DownloadPrecheck(c *gin.Context) {
	orderID := c.Param("orderId")

	var order model.Order
	if err := ac.DB.Select("order_id", "precheck_passed", "precheck_result", "prechecked_at").
		Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	if order.PrecheckedAt == nil {
		handle.NotFoundError(c, "No precheck result available for this order")
		return
	}

	type Result struct {
		Passed    bool             `json:"passed"`
		Result    *json.RawMessage `json:"result"`
		CheckedAt string           `json:"checked_at"`
	}

	result := Result{
		Passed:    order.PrecheckPassed,
		Result:    order.PrecheckResult,
		CheckedAt: order.PrecheckedAt.Format("2006-01-02 15:04:05"),
	}

	handle.Success(c, result)
}

type orderOverview struct {
	OrderID             string     `json:"order_id"`
	User                string     `json:"user"`
	DoseAdjusted        bool       `json:"dose_adjusted"`
	PrecheckPassed      bool       `json:"precheck_passed"`
	PrecheckError       *string    `json:"precheck_error,omitempty"`
	ProcessErrorMessage *string    `json:"process_error,omitempty"`
	LastSendError       *string    `json:"last_send_error,omitempty"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	ProcessedAt         *time.Time `json:"processed_at,omitempty"`
	SentAt              *time.Time `json:"sent_at,omitempty"`
}

func (ac *AdminController) GetOrders(c *gin.Context) {
	var orders []model.Order
	query := ac.DB.Preload("User")

	// Optional filters
	status := c.Query("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	owner := c.Query("user")
	if owner != "" {
		if id, err := strconv.Atoi(owner); err == nil {
			query = query.Where("user_id = ?", id)
		} else {
			query = query.Joins("JOIN users ON users.id = orders.user_id").
				Where("users.email = ?", owner)
		}
	}

	if err := query.
		Omit("order_data", "precheck_result", "process_result_pdf").
		Order("created_at desc").Find(&orders).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	var response []orderOverview
	for _, o := range orders {
		response = append(response, orderOverview{
			OrderID:             o.OrderID,
			User:                o.User.Email,
			DoseAdjusted:        o.DoseAdjusted,
			PrecheckPassed:      o.PrecheckPassed,
			ProcessErrorMessage: o.ProcessErrorMessage,
			LastSendError:       o.LastSendError,
			Status:              o.Status,
			CreatedAt:           o.CreatedAt,
			ProcessedAt:         o.ProcessedAt,
			SentAt:              o.SentAt,
		})
	}

	if len(response) == 0 {
		handle.NotFoundError(c, "No orders found that match the query")
		return
	}

	handle.Success(c, response)
}

func (ac *AdminController) GetOrderByID(c *gin.Context) {
	orderID := c.Param("orderId")
	var order model.Order

	// You could add validation here to make sure orderID is an integer if needed
	query := ac.DB.Preload("User")

	if err := query.
		Omit("order_data", "precheck_result", "process_result_pdf").
		Where("order_id = ?", orderID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
		} else {
			handle.ServerError(c, err)
		}
		return
	}

	response := orderOverview{
		OrderID:             order.OrderID,
		User:                order.User.Email,
		DoseAdjusted:        order.DoseAdjusted,
		PrecheckPassed:      order.PrecheckPassed,
		ProcessErrorMessage: order.ProcessErrorMessage,
		LastSendError:       order.LastSendError,
		Status:              order.Status,
		CreatedAt:           order.CreatedAt,
		ProcessedAt:         order.ProcessedAt,
		SentAt:              order.SentAt,
	}

	handle.Success(c, response)
}

func (ac *AdminController) ResetFailedSends(c *gin.Context) {
	result := ac.DB.Model(&model.Order{}).
		Where("status = ?", model.StatusSendFailed).
		Updates(map[string]interface{}{
			"status":               model.StatusProcessed,
			"last_send_error":      nil,
			"last_send_attempt_at": nil,
			"next_send_attempt_at": nil,
			"sent_at":              nil,
			"send_tries":           0,
		})

	if result.Error != nil {
		handle.ServerError(c, result.Error)
		return
	}

	handle.Success(c, gin.H{
		"message": "Orders with failed sends resetted",
		"orders":  result.RowsAffected,
	})
}

func (ac *AdminController) ResendOrder(c *gin.Context) {
	orderID := c.Param("orderId")

	var order model.Order
	if err := ac.DB.
		Select("id", "order_id", "status").
		Where("order_id = ?", orderID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handle.NotFoundError(c, "Order not found")
			return
		}
		handle.ServerError(c, err)
		return
	}

	if order.Status != model.StatusSendFailed &&
		order.Status != model.StatusSent {
		handle.BadRequestError(c, "Order must be in send_failed or sent state")
		return
	}

	if err := ac.DB.Model(&order).Updates(map[string]interface{}{
		"status":               model.StatusProcessed,
		"last_send_error":      nil,
		"last_send_attempt_at": nil,
		"next_send_attempt_at": nil,
		"sent_at":              nil,
		"send_tries":           0,
	}).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, gin.H{
		"message": "Order reset to processing state",
	})
}

// @Summary		Create a new service user
// @Description	__Admin role required__
// @Description	Create a new service user for the API.
// @Description	You can create users with the following roles: `admin`, `user`, `debug`.
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
		Role      string `json:"role" binding:"required,oneof=admin user debug"`
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
		Role   string `form:"role" binding:"omitempty,oneof=admin user debug"`
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

// @Summary		Get user by email
// @Description	__Admin role required__
// @Description	Retrieve a single user by their email address.
// @Tags			Admin
// @Produce		json
// @Param			email	path		string										true	"User email"
// @Success		200		{object}	handle.jsendSuccess[model.User]				"User found"
// @Failure		400		{object}	handle.jsendFailure[handle.errorResponse]	"Bad request"
// @Failure		401		{object}	handle.jsendFailure[handle.errorResponse]	"Unauthorized"
// @Failure		403		{object}	handle.jsendFailure[handle.errorResponse]	"Non-admin user"
// @Failure		404		{object}	handle.jsendFailure[handle.errorResponse]	"User not found"
// @Failure		500		{object}	handle.jSendError							"Internal server error"
// @Security		Bearer
// @Router			/admin/users/{email} [get]
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

// @Summary		Delete user by email
// @Description	__Admin role required__
// @Description	Delete a user by their email address. Cannot delete own account.
// @Tags			Admin
// @Produce		json
// @Param			email	path		string										true	"User email"
// @Success		200		{object}	handle.jsendSuccess[map[string]string]		"User deleted"
// @Failure		400		{object}	handle.jsendFailure[handle.errorResponse]	"Bad request"
// @Failure		401		{object}	handle.jsendFailure[handle.errorResponse]	"Unauthorized"
// @Failure		403		{object}	handle.jsendFailure[handle.errorResponse]	"Cannot delete own account"
// @Failure		404		{object}	handle.jsendFailure[handle.errorResponse]	"User not found"
// @Failure		500		{object}	handle.jSendError							"Internal server error"
// @Security		Bearer
// @Router			/admin/users/{email} [delete]
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

// @Summary		Change user profile
// @Description	__Admin role required__
// @Description	Update a user's role or status. Cannot change own role or status.
// @Tags			Admin
// @Accept			json
// @Produce		json
// @Param			email					path		string										true	"User email"
// @Param			ChangeUserProfileQuery	body		ChangeUserProfileQuery						true	"Role and/or status updates"
// @Success		200						{object}	handle.jsendSuccess[map[string]string]		"User profile updated"
// @Failure		400						{object}	handle.jsendFailure[handle.errorResponse]	"Bad request"
// @Failure		401						{object}	handle.jsendFailure[handle.errorResponse]	"Unauthorized"
// @Failure		403						{object}	handle.jsendFailure[handle.errorResponse]	"Cannot change own role or status"
// @Failure		404						{object}	handle.jsendFailure[handle.errorResponse]	"User not found"
// @Failure		500						{object}	handle.jSendError							"Internal server error"
// @Security		Bearer
// @Router			/admin/users/{email} [patch]
func (ac *AdminController) ChangeUserProfile(c *gin.Context) {
	type Query struct {
		Role   string `json:"role" binding:"omitempty,oneof=admin user debug" example:"user"`
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
