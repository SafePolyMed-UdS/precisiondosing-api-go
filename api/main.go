package main

import (
	"fmt"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/server"
	"precisiondosing-api-go/internal/utils/logger"

	_ "github.com/joho/godotenv/autoload"
)

var (
	versionTag = "dev" //nolint:gochecknoglobals // version tag
)

func main() {
	args := cfg.ParseCmdLineArgs()

	// config
	cfg.MustParseEnvFile(&args.EnvFile)
	config := cfg.MustParseYAML(args.ConfigFile)
	config.Meta.VersionTag = versionTag

	// log
	logger.MustInit(config.Log, args.DebugMode)

	// server
	srv, err := server.New(config, args.DebugMode)
	if err != nil {
		logger.LogInternalError(err)
		panic(fmt.Sprintf("Cannot create server: %v", err))
	}

	_ = srv
	//srv.Run()
}
