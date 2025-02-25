package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/database"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/middleware"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/responder"
	"precisiondosing-api-go/internal/utils/abdata"
	"precisiondosing-api-go/internal/utils/validate"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Engine       *gin.Engine
	ServerConfig cfg.ServerConfig
}

func New(config *cfg.APIConfig, debug bool) (*Server, error) {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	databases, err := initDatabases(config)
	if err != nil {
		return nil, fmt.Errorf("error initializing databases: %w", err)
	}

	// init abdata
	abdata, err := initABDATA(config)
	if err != nil {
		return nil, fmt.Errorf("cannot init ABDATA: %w", err)
	}

	// setup router
	router := gin.New()

	// trusted proxies
	trusedProxies := parseTrustedProxies(config.Server.TrustedProxies)
	if err = router.SetTrustedProxies(trusedProxies); err != nil {
		return nil, fmt.Errorf("cannot set trusted proxies: %w", err)
	}

	// middleware
	router.Use(gin.CustomRecovery(middleware.RecoveryHandler))
	if debug {
		router.Use(gin.Logger())
	}

	// create Mailer
	mailer := responder.NewMailer(config.Mailer, config.Meta, debug)

	// create JSON validators
	jsonValidators, err := initJSONValidators(&config.Schema)
	if err != nil {
		return nil, fmt.Errorf("cannot init JSON validators: %w", err)
	}

	// routes
	resourceHandle := handle.NewResourceHandle(config, databases, abdata, mailer, jsonValidators, debug)
	registerRoutes(router, resourceHandle)

	// server
	srv := &Server{
		Engine:       router,
		ServerConfig: config.Server,
	}

	return srv, nil
}

func (r *Server) Run() {
	srv := &http.Server{
		Addr:         r.ServerConfig.Address,
		Handler:      r.Engine,
		ReadTimeout:  r.ServerConfig.ReadWriteTimeout,
		WriteTimeout: r.ServerConfig.ReadWriteTimeout,
		IdleTimeout:  r.ServerConfig.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = srv.ListenAndServe()
	}()
	log.Info().Msgf("Server started on %s", r.ServerConfig.Address)
	<-quit
	log.Info().Msg("Server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exiting")
}

func registerRoutes(r *gin.Engine, resourceHandle *handle.ResourceHandle) {
	api := r.Group(resourceHandle.MetaCfg.Group)

	RegistgerSwaggerRoutes(r, api, resourceHandle)
	RegisterSysRoutes(api, resourceHandle)
	RegisterUserRoutes(api, resourceHandle)
	RegisterAdminRoutes(api, resourceHandle)
	RegisterDSSRoutes(api, resourceHandle)
}

func parseTrustedProxies(proxies string) []string {
	if proxies == "" {
		return nil
	}
	return strings.Split(proxies, ",")
}

func initDatabases(config *cfg.APIConfig) (handle.Databases, error) {
	dbs := handle.Databases{}
	// init sql database
	gorm, sqlx, err := database.New(config.Database)
	if err != nil {
		return dbs, fmt.Errorf("cannot create SQL database: %w", err)
	}
	dbs.GormDB = gorm
	dbs.SqlxDB = sqlx

	// migrate database
	if err = database.Migrate(gorm); err != nil {
		return dbs, fmt.Errorf("cannot migrate SQL database: %w", err)
	}

	// init mongo db
	individualsDB, err := mongodb.New(config.Mongo)
	if err != nil {
		return dbs, fmt.Errorf("cannot connect to mongodb: %w", err)
	}
	dbs.MongoDB = individualsDB

	return dbs, nil
}

func initABDATA(config *cfg.APIConfig) (*abdata.API, error) {
	aCfg := config.ABDATA
	api := abdata.NewJWT(aCfg.URL, aCfg.Login, aCfg.Password)
	if err := api.Refresh(); err != nil {
		return nil, fmt.Errorf("cannot login to ABDATA: %w", err)
	}

	return api, nil
}

func initJSONValidators(config *cfg.SchemaConfig) (handle.JSONValidators, error) {

	validators := handle.JSONValidators{}

	var err error
	validators.PreCheck, err = validate.NewJSONValidator(config.PreCheck)
	if err != nil {
		return validators, fmt.Errorf("cannot create PreCheck validator: %w", err)
	}

	return validators, nil
}
