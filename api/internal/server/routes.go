package server

import (
	"precisiondosing-api-go/docs"
	"precisiondosing-api-go/internal/controller/admincontroller"
	"precisiondosing-api-go/internal/controller/dsscontroller"
	"precisiondosing-api-go/internal/controller/modelcontroller"
	"precisiondosing-api-go/internal/controller/syscontroller"
	"precisiondosing-api-go/internal/controller/testcontroller"
	"precisiondosing-api-go/internal/controller/usercontroller"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterSysRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := syscontroller.New(resourceHandle)

	users := r.Group("/sys")
	{
		users.GET("/ping", c.GetPing)
		users.GET("/info", c.GetInfo)
	}
}

func RegisterUserRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := usercontroller.New(resourceHandle)

	// no auth here
	user := r.Group("/user")
	{
		user.POST("/login", c.Login)
		user.POST("/refresh-token", c.RefreshToken)
	}
}

func RegisterAdminRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := admincontroller.New(resourceHandle)

	admin := r.Group("/admin")
	admin.Use(middleware.Authentication(&resourceHandle.AuthCfg), middleware.AdminAccess())
	{
		// user endpoints
		admin.POST("/users/service", c.CreateServiceUser)
		admin.GET("/users", c.GetUsers)
		admin.GET("/users/:email", c.GetUserByEmail)
		admin.DELETE("/users/:email", c.DeleteUserByEmail)
		admin.PATCH("/users/:email", c.ChangeUserProfile)

		// download endpoints
		admin.GET("/download/pdf/:orderId", c.DownloadPDF)
		admin.GET("/download/order/:orderId", c.DownloadOrder)
		admin.GET("/download/precheck/:orderId", c.DownloadPrecheck)

		// order overview endpoints
		admin.GET("/orders", c.GetOrders)
		admin.GET("/orders/:orderId", c.GetOrderByID)

		// send endpoints
		admin.PATCH("/orders/send/failed", c.ResetFailedSends)
		admin.PATCH("/orders/send/:orderId", c.ResendOrder)

		// reset endpoints
		admin.PATCH("/orders/requeue/errors", c.RequeFailedOrders)
		admin.PATCH("/orders/requeue/:id", RequeOrderByID)

		// delete endpoints
		admin.DELETE("/orders/:orderId", c.DeleteOrderByID)

	}
}

func RegisterDSSRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := dsscontroller.New(resourceHandle)

	dss := r.Group("/dose")
	dss.Use(middleware.Authentication(&resourceHandle.AuthCfg))
	{
		dss.POST("/precheck/", c.PostPrecheck)
		dss.POST("/adjust/", c.PostAdjust)
	}
}

func RegisterModelRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := modelcontroller.New(resourceHandle.Prechecker.PBPKModels.Definitions)

	models := r.Group("/models")
	models.Use(middleware.Authentication(&resourceHandle.AuthCfg))
	{
		models.GET("/", c.GetModels)
	}
}

func RegisterTestRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := testcontroller.New()

	test := r.Group("/test")
	test.Use(middleware.Authentication(&resourceHandle.AuthCfg))
	{
		test.POST("/acceptresult/:orderId", c.AcceptResult)
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
