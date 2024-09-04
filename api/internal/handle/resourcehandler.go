package handle

import (
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/utils/abdata"

	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

// Central struct to hold all the configurations and database connection pool.
type ResourceHandle struct {
	MetaCfg   cfg.MetaConfig
	AuthCfg   cfg.AuthTokenConfig
	ResetCfg  cfg.ResetTokenConfig
	Databases Databases
	ABDATA    *abdata.API
}

type Databases struct {
	GormDB  *gorm.DB
	SqlxDB  *sqlx.DB
	MongoDB *mongodb.MongoConnection
}

func NewResourceHandle(apiCfg *cfg.APIConfig, databases Databases, abdata *abdata.API) *ResourceHandle {
	return &ResourceHandle{
		MetaCfg:   apiCfg.Meta,
		AuthCfg:   apiCfg.AuthToken,
		ResetCfg:  apiCfg.ResetToken,
		Databases: databases,
		ABDATA:    abdata,
	}
}
