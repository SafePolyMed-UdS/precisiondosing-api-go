package database

import (
	"fmt"
	"log"
	"os"
	"precisiondosing-api-go/cfg"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(dbConfig cfg.DatabaseConfig, logConfig cfg.LogConfig, debug bool) (*gorm.DB, error) {
	dsn := fmt.Sprintf("@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbConfig.Host, dbConfig.DBName)

	if dbConfig.Password == "" {
		dsn = dbConfig.Username + dsn
	} else {
		dsn = fmt.Sprintf("%s:%s", dbConfig.Username, dbConfig.Password) + dsn
	}

	var logLvl logger.LogLevel
	switch logConfig.DBLevel {
	case "ERROR":
		logLvl = logger.Error
	case "WARN":
		logLvl = logger.Warn
	case "INFO":
		logLvl = logger.Info
	case "SILENT":
		logLvl = logger.Silent
	default:
		return nil, fmt.Errorf("invalid log level: %s", logConfig.DBLevel)
	}

	logCfg := logger.Config{
		SlowThreshold:             logConfig.SlowQueryThreshold,
		LogLevel:                  logLvl,
		IgnoreRecordNotFoundError: !debug,
		ParameterizedQueries:      !debug,
		Colorful:                  debug,
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io.Writer
		logCfg,
	)

	gorm, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, fmt.Errorf("cannot open database %s: %w", dbConfig.DBName, err)
	}

	db, err := gorm.DB()
	if err != nil {
		return nil, fmt.Errorf("cannot get database %s: %w", dbConfig.DBName, err)
	}

	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(dbConfig.MaxConnLifetime)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot ping database %s: %w", dbConfig.DBName, err)
	}

	return gorm, nil
}
