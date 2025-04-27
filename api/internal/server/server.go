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
	"precisiondosing-api-go/internal/jobs/jobrunner"
	"precisiondosing-api-go/internal/jobs/jobsender"
	"precisiondosing-api-go/internal/middleware"
	"precisiondosing-api-go/internal/pbpk"
	"precisiondosing-api-go/internal/precheck"
	"precisiondosing-api-go/internal/services/individualdb"
	"precisiondosing-api-go/internal/services/medinfo"
	"precisiondosing-api-go/internal/utils/callr"
	"precisiondosing-api-go/internal/utils/validate"
	"runtime"
	"strings"
	"syscall"
	"time"

	"precisiondosing-api-go/internal/utils/log"

	"github.com/gin-gonic/gin"
)

type Server struct {
	engine       *gin.Engine
	serverConfig cfg.ServerConfig
	jobRunner    *jobrunner.JobRunner
	jobSender    *jobsender.JobSender
	logger       log.Logger
}

func New(config *cfg.APIConfig, debug bool) (*Server, error) {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// setup router
	router := gin.New()

	// trusted proxies
	trusedProxies := parseTrustedProxies(config.Server.TrustedProxies)
	if err := router.SetTrustedProxies(trusedProxies); err != nil {
		return nil, fmt.Errorf("cannot set trusted proxies: %w", err)
	}

	// middleware
	router.Use(gin.CustomRecovery(middleware.RecoveryHandler))
	if debug {
		router.Use(gin.Logger())
	}

	// init handler
	resourceHandle, err := initHandler(config, debug)
	if err != nil {
		return nil, fmt.Errorf("error initializing handler: %w", err)
	}

	// routes
	registerRoutes(router, resourceHandle)

	// init job runner
	jobRunner := jobrunner.New(
		config.JobRunner,
		resourceHandle.Prechecker,
		resourceHandle.CallR,
		resourceHandle.Databases.GormDB,
	)

	// init job sender
	jobSender := jobsender.New(config.MMCAPI, resourceHandle.Databases.GormDB)

	// server
	srv := &Server{
		engine:       router,
		serverConfig: config.Server,
		jobRunner:    jobRunner,
		jobSender:    jobSender,
		logger:       log.WithComponent("server"),
	}

	return srv, nil
}

func (s *Server) Run() {
	srv := &http.Server{
		Addr:         s.serverConfig.Address,
		Handler:      s.engine,
		ReadTimeout:  s.serverConfig.ReadWriteTimeout,
		WriteTimeout: s.serverConfig.ReadWriteTimeout,
		IdleTimeout:  s.serverConfig.IdleTimeout,
	}

	s.jobRunner.Start()
	s.jobSender.Start()

	// Graceful shutdown for the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = srv.ListenAndServe()
	}()
	s.logger.Info("started", log.Str("Address", s.serverConfig.Address))
	<-quit
	s.logger.Info("shutting down...")

	s.jobRunner.Stop()
	s.jobSender.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("shutdown error", log.Err(err))
	}

	s.logger.Info("exited")
}

func initHandler(config *cfg.APIConfig, debug bool) (*handle.ResourceHandle, error) {
	// init databases
	databases, err := initDatabases(config, debug)
	if err != nil {
		return nil, fmt.Errorf("error initializing databases: %w", err)
	}

	// init prechecker
	prechecker, err := initPrechecker(config, databases.MongoDB)
	if err != nil {
		return nil, fmt.Errorf("error initializing prechecker: %w", err)
	}

	// create JSON validators
	jsonValidators, err := initJSONValidators(&config.Schema)
	if err != nil {
		return nil, fmt.Errorf("cannot init JSON validators: %w", err)
	}

	// create CallR
	os := runtime.GOOS
	rscriptPath := config.RLang.RScriptPathUnix
	if os == "windows" {
		rscriptPath = config.RLang.RScriptPathWin
	}

	callR := callr.New(
		rscriptPath,
		config.RLang.DoseAdjustScript,
		config.Database.Host,
		config.Database.DBName,
		config.Database.Username,
		config.Database.Password,
		config.RLang.RWorker,
	)

	resourceHandle := handle.NewResourceHandle(config, databases, prechecker, callR, jsonValidators, debug)
	return resourceHandle, nil
}

func registerRoutes(r *gin.Engine, resourceHandle *handle.ResourceHandle) {
	api := r.Group(resourceHandle.MetaCfg.Group)

	RegistgerSwaggerRoutes(r, api, resourceHandle)
	RegisterSysRoutes(api, resourceHandle)
	RegisterUserRoutes(api, resourceHandle)
	RegisterAdminRoutes(api, resourceHandle)
	RegisterDSSRoutes(api, resourceHandle)
	RegisterModelRoutes(api, resourceHandle)
	if resourceHandle.DebugMode {
		RegisterTestRoutes(api, resourceHandle)
	}
}

func parseTrustedProxies(proxies string) []string {
	if proxies == "" {
		return nil
	}
	return strings.Split(proxies, ",")
}

func initDatabases(config *cfg.APIConfig, debug bool) (handle.Databases, error) {
	dbs := handle.Databases{}
	// init dbs
	gorm, err := database.New(config.Database, config.Log, debug)
	if err != nil {
		return dbs, fmt.Errorf("cannot create SQL database: %w", err)
	}
	dbs.GormDB = gorm

	// migrate database
	if err = database.Migrate(gorm); err != nil {
		return dbs, fmt.Errorf("cannot migrate SQL database: %w", err)
	}

	// init mongo db
	individualsDB, err := individualdb.New(config.IndividualDB)
	if err != nil {
		return dbs, fmt.Errorf("cannot connect to mongodb: %w", err)
	}
	dbs.MongoDB = individualsDB

	return dbs, nil
}

func initPrechecker(config *cfg.APIConfig, mongo *individualdb.IndividualDB) (*precheck.PreCheck, error) {
	// models
	modelDefinitions := pbpk.MustParseAll(config.Models)

	// init Abdata
	aCfg := config.MedInfoAPI
	medinfoAPI := medinfo.NewAPI(aCfg.URL, aCfg.Login, aCfg.Password, aCfg.ExpiryThreshold)
	if err := medinfoAPI.Refresh(); err != nil {
		return nil, fmt.Errorf("cannot login to MedInfo: %w", err)
	}

	// init medinfo
	prechecker := precheck.New(mongo, medinfoAPI, modelDefinitions)
	return prechecker, nil
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
