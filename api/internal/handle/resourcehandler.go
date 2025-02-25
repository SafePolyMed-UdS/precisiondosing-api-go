package handle

import (
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/responder"
	"precisiondosing-api-go/internal/utils/abdata"
	"precisiondosing-api-go/internal/utils/helper"

	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

// Central struct to hold all the configurations and database connection pool.
type ResourceHandle struct {
	ServerCfg cfg.ServerConfig
	MetaCfg   cfg.MetaConfig
	AuthCfg   cfg.AuthTokenConfig
	ResetCfg  cfg.ResetTokenConfig
	Mailer    *responder.Mailer
	Databases Databases
	ABDATA    *abdata.API
	DebugMode bool
}

type Databases struct {
	GormDB  *gorm.DB
	SqlxDB  *sqlx.DB
	MongoDB *mongodb.MongoConnection
}

func NewResourceHandle(
	apiCfg *cfg.APIConfig,
	databases Databases,
	abdata *abdata.API,
	mailer *responder.Mailer,
	debug bool,
) *ResourceHandle {
	res := &ResourceHandle{
		ServerCfg: apiCfg.Server,
		MetaCfg:   apiCfg.Meta,
		AuthCfg:   apiCfg.AuthToken,
		ResetCfg:  apiCfg.ResetToken,
		Databases: databases,
		ABDATA:    abdata,
		Mailer:    mailer,
		DebugMode: debug,
	}

	res.MetaCfg.URL = helper.RemoveTrailingSlash(res.MetaCfg.URL)
	res.MetaCfg.Group = helper.RemoveTrailingSlash(res.MetaCfg.Group)
	res.MetaCfg.Group = helper.AddLeadingSlash(res.MetaCfg.Group)

	return res
}
