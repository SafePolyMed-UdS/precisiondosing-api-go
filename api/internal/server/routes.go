package server

import (
	"precisiondosing-api-go/docs"
	"precisiondosing-api-go/internal/controller/admincontroller"
	"precisiondosing-api-go/internal/controller/dsscontroller"
	"precisiondosing-api-go/internal/controller/syscontroller"
	"precisiondosing-api-go/internal/controller/usercontroller"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterSysRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := syscontroller.NewSysController(resourceHandle)

	users := r.Group("/sys")
	{
		users.GET("/ping", c.GetPing)
		users.GET("/info", c.GetInfo)
	}
}

func RegisterUserRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := usercontroller.NewUserController(resourceHandle)

	// no auth here
	user := r.Group("/user")
	{
		user.POST("/login", c.Login)
		user.POST("/refresh-token", c.RefreshToken)
		user.POST("/password/reset", c.ResetPwd)
		user.POST("/password/init", c.ResetPwd)
		user.POST("/password/reset/confirm", c.ResetPwdConfirm)
	}

	authUser := r.Group("/user")
	authUser.Use(middleware.Authentication(&resourceHandle.AuthCfg))
	{
		authUser.PATCH("/password", c.ChangePwd)
		authUser.PATCH("/email", c.ChangeEmail)
		authUser.POST("/email/confirm", c.ConfirmEmailChange)
		authUser.DELETE("/", c.DeleteAccount)
		authUser.GET("/profile", c.GetProfile)
		authUser.PATCH("/profile", c.UpdateProfile)
	}
}

func RegisterAdminRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := admincontroller.NewAdminController(resourceHandle)

	authUser := r.Group("/admin")
	authUser.Use(middleware.Authentication(&resourceHandle.AuthCfg), middleware.AdminAccess())
	{
		authUser.POST("/users/", c.CreateUser)
		authUser.POST("/users/service", c.CreateServiceUser)
		authUser.GET("/users", c.GetUsers)
		authUser.GET("/users/:email", c.GetUserByEmail)
		authUser.DELETE("/users/:email", c.DeleteUserByEmail)
		authUser.PATCH("/users/:email", c.ChangeUserProfile)
	}
}

func RegisterDSSRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := dsscontroller.NewDSSController(resourceHandle)

	dss := r.Group("/dose")
	dss.Use(middleware.Authentication(&resourceHandle.AuthCfg))
	{
		dss.POST("/precheck/", c.PostPrecheck)
		dss.POST("/adjust/", c.AdaptDose)
	}
}

func RegistgerSwaggerRoutes(r *gin.Engine, api *gin.RouterGroup, handle *handle.ResourceHandle) {
	hostURL := handle.MetaCfg.URL
	if handle.DebugMode {
		hostURL = handle.ServerCfg.Address
	}

	basePath := api.BasePath()
	docs.SwaggerInfo.BasePath = basePath
	docs.SwaggerInfo.Host = hostURL
	docs.SwaggerInfo.Title = handle.MetaCfg.Name
	docs.SwaggerInfo.Description = handle.MetaCfg.Description
	docs.SwaggerInfo.Version = handle.MetaCfg.Version

	swaggerURL := basePath + "/swagger/"
	swaggerIndex := swaggerURL + "index.html"

	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, swaggerIndex)
	})

	handlerFn := ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.DefaultModelsExpandDepth(-1))

	api.GET("/swagger/*any", func(c *gin.Context) {
		if c.Request.RequestURI == swaggerURL {
			c.Redirect(302, swaggerIndex)
		} else {
			handlerFn(c)
		}
	})
}
