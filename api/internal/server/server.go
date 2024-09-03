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
	"precisiondosing-api-go/internal/utils/abdata"
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

	// create database
	gorm, sqlx, err := database.New(config.Database)
	if err != nil {
		return nil, fmt.Errorf("cannot create database: %w", err)
	}

	// migrate database
	if err = database.Migrate(gorm); err != nil {
		return nil, fmt.Errorf("cannot migrate database: %w", err)
	}

	// init mongo db
	individualsDB, err := mongodb.New(config.Mongo)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to mongodb: %w", err)
	}

	// init abdata
	abdata, err := initABDATA(&config.ABDATA)
	if err != nil {
		return nil, fmt.Errorf("cannot init ABDATA: %w", err)
	}

	// setup handler
	resourceHandle := handle.NewResourceHandle(
		config.Meta,
		config.AuthToken,
		config.ResetToken,
		gorm,
		sqlx,
		abdata,
		individualsDB,
	)

	// setup router
	r := gin.New()

	// trusted proxies
	trusedProxies := parseTrustedProxies(config.Server.TrustedProxies)
	if err = r.SetTrustedProxies(trusedProxies); err != nil {
		return nil, fmt.Errorf("cannot set trusted proxies: %w", err)
	}

	// middleware
	r.Use(gin.CustomRecovery(middleware.RecoveryHandler))
	if debug {
		r.Use(gin.Logger())
	}

	// routes
	registerRoutes(r, resourceHandle)

	// server
	srv := &Server{
		Engine:       r,
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
	api := r.Group("/api")

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

func initABDATA(config *cfg.ABDATAConfig) (*abdata.API, error) {

	api := abdata.NewJWT(config.URL, config.Login, config.Password)
	if err := api.Refresh(); err != nil {
		return nil, fmt.Errorf("cannot login to ABDATA: %w", err)
	}

	return api, nil
}
