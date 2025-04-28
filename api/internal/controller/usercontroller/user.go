package usercontroller

import (
	"fmt"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/utils/hash"
	"precisiondosing-api-go/internal/utils/tokens"
	"precisiondosing-api-go/internal/utils/validate"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	DB      *gorm.DB
	AuthCfg cfg.AuthTokenConfig
}

func New(resourceHandle *handle.ResourceHandle) *UserController {
	return &UserController{
		DB:      resourceHandle.Databases.GormDB,
		AuthCfg: resourceHandle.AuthCfg,
	}
}

type loginResponse struct {
	tokens.AuthTokens
	Role      string     `json:"role" example:"user"`                       // User role
	LastLogin *time.Time `json:"last_login" example:"2021-07-01T12:00:00Z"` // Last login time
} //	@name	LoginResponse

func newLoginResponse(tokens *tokens.AuthTokens, role string, lastLogin *time.Time) *loginResponse {
	result := &loginResponse{
		AuthTokens: *tokens,
		Role:       role,
		LastLogin:  lastLogin,
	}

	return result
}

func switchRole(dbRole string, requestedRole *string) (string, error) {
	if requestedRole == nil || *requestedRole == dbRole {
		return dbRole, nil
	}

	err := validate.CanSwitchToRole(*requestedRole, dbRole)
	if err != nil {
		return dbRole, fmt.Errorf("cannot switch to role: %w", err)
	}
	return *requestedRole, nil
}

// @Summary		Login for the API to get JWT token
// @Description	Acciqures a JWT token for the user to access the API
// @Description	Only active users can login
// @Description	Users can downgrade their role by providing the role in the request (optional)
// @Tags			Login
// @Produce		json
// @Param			request	body		LoginQuery										true	"Request body"
// @Success		200		{object}	handle.jsendSuccess[loginResponse]				"JWT token"
// @Failure		401		{object}	handle.jsendFailure[handle.errorResponse]		"Unauthorized"
// @Failure		422		{object}	handle.jsendFailure[handle.validationResponse]	"Bad query format"
// @Failure		403		{object}	handle.jsendFailure[handle.errorResponse]		"User is not active"
// @Failure		500		{object}	handle.jSendError								"Internal server error"
//
// @Router			/user/login [post]
func (uc *UserController) Login(c *gin.Context) {
	type Query struct {
		Login    string  `json:"login" binding:"required" example:"joe@me.com"`
		Password string  `json:"password" binding:"required" example:"password"`
		Role     *string `json:"role" binding:"omitempty,oneof=admin user approver" example:"user"`
	} //	@name	LoginQuery

	var query Query
	if !handle.JSONBind(c, &query) {
		return
	}

	user, err := model.GetUserByEmail(uc.DB, query.Login)
	if err != nil {
		handle.UnauthorizedError(c, "Invalid ceredentials")
		return
	}

	if user.PwdHash == nil {
		handle.UnauthorizedError(c, "Invalid ceredentials")
		return
	}

	if validPwd, _ := hash.Check(*user.PwdHash, query.Password); !validPwd {
		handle.UnauthorizedError(c, "Invalid credentials")
		return
	}

	if user.Status != "active" {
		handle.ForbiddenError(c, "User account is not active")
		return
	}

	newRole, err := switchRole(user.Role, query.Role)
	if err != nil {
		handle.ForbiddenError(c, "Unauthorized role access")
		return
	}

	claims := tokens.CustomClaims{
		ID:    user.ID,
		Email: user.Email,
		Role:  newRole,
	}
	token, err := tokens.CreateAuthTokens(&claims, &uc.AuthCfg)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	res := newLoginResponse(token, newRole, user.LastLogin)
	_ = user.UpdateLastLogin(uc.DB)

	handle.Success(c, res)
}

// @Summary		Refresh JWT token
// @Description	Refreshes the JWT token for the user to access the API
// @Tags			Login
// @Produce		json
// @Param			request	body		RefreshQuery									true	"Request body"
// @Success		200		{object}	handle.jsendSuccess[loginResponse]				"JWT token"
// @Failure		401		{object}	handle.jsendFailure[handle.errorResponse]		"Unauthorized"
// @Failure		422		{object}	handle.jsendFailure[handle.validationResponse]	"Bad query format"
// @Failure		403		{object}	handle.jsendFailure[handle.errorResponse]		"Not active/role invalid/user deleted"
// @Failure		500		{object}	handle.jSendError								"Internal server error"
//
// @Router			/user/refresh-token [post]
func (uc *UserController) RefreshToken(c *gin.Context) {
	type Query struct {
		Token string `json:"refresh_token" binding:"required" example:"my_refresh_token"`
	} //	@name	RefreshQuery

	var query Query
	if !handle.JSONBind(c, &query) {
		return
	}

	claims, err := tokens.CheckRefreshToken(query.Token, &uc.AuthCfg)
	if err != nil {
		handle.UnauthorizedError(c, "Invalid refresh token")
		return
	}

	user, err := model.GetUserByID(uc.DB, claims.ID)
	if err != nil {
		handle.ForbiddenError(c, "User not found")
		return
	}

	if user.Status != "active" {
		handle.ForbiddenError(c, "User account is not active")
		return
	}

	if err = validate.CanSwitchToRole(claims.Role, user.Role); err != nil {
		handle.ForbiddenError(c, "Unauthorized role access")
		return
	}

	updatedClaims := tokens.CustomClaims{
		ID:    user.ID,
		Email: user.Email,
		Role:  claims.Role,
	}
	newToken, err := tokens.CreateAuthTokens(&updatedClaims, &uc.AuthCfg)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	res := newLoginResponse(newToken, user.Role, user.LastLogin)
	_ = user.UpdateLastLogin(uc.DB)
	handle.Success(c, res)
}
