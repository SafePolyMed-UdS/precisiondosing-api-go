package server

import (
	"precisiondosing-api-go/docs"
	"precisiondosing-api-go/internal/controller/admincontroller"
	"precisiondosing-api-go/internal/controller/downloadcontroller"
	"precisiondosing-api-go/internal/controller/dsscontroller"
	"precisiondosing-api-go/internal/controller/modelcontroller"
	"precisiondosing-api-go/internal/controller/ordercontroller"
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
	}
}

func RegisterDownloadRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := downloadcontroller.New(resourceHandle)

	download := r.Group("/download")
	download.Use(middleware.Authentication(&resourceHandle.AuthCfg), middleware.AdminAccess())
	{
		// download endpoints
		download.GET("/pdf/:order_id", c.DownloadPDF)
		download.GET("/order/:order_id", c.DownloadOrder)
		download.GET("/precheck/:order_id", c.DownloadPrecheck)
	}
}

func RegisterOrderRoutes(r *gin.RouterGroup, resourceHandle *handle.ResourceHandle) {
	c := ordercontroller.New(resourceHandle)

	order := r.Group("/orders")
	order.Use(middleware.Authentication(&resourceHandle.AuthCfg), middleware.AdminAccess())
	{
		order.GET("/", c.GetOrders)
		order.GET("/:order_id", c.GetOrderByID)

		order.PATCH("/send/failed", c.ResetFailedSends)
		order.PATCH("/send/:order_id", c.ResendOrder)

		// reset endpoints
		order.PATCH("/requeue/errors", c.RequeueErrorOrders)
		order.PATCH("/requeue/:order_id", c.RequeueOrderByID)

		// delete endpoints
		order.DELETE("/delete/:order_id", c.DeleteOrderByID)
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
