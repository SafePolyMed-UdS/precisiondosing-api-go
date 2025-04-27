package main

import (
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/server"
	"precisiondosing-api-go/internal/utils/log"

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
	log.MustInit(config.Log, args.DebugMode)
	logger := log.WithComponent("server")

	// server
	srv, err := server.New(config, args.DebugMode)
	if err != nil {
		logger.Panic("cannot create server", log.Err(err))
	}

	_ = srv
	srv.Run()
}
