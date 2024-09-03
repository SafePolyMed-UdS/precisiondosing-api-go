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
	MetaCfg       cfg.MetaConfig
	AuthCfg       cfg.AuthTokenConfig
	ResetCfg      cfg.ResetTokenConfig
	Gorm          *gorm.DB
	SQLX          *sqlx.DB
	ABDATA        *abdata.API
	IndibidualsDB *mongodb.MongoConnection
}

func NewResourceHandle(
	metaCfg cfg.MetaConfig,
	authCfg cfg.AuthTokenConfig,
	resetCfg cfg.ResetTokenConfig,
	gorm *gorm.DB,
	sqlx *sqlx.DB,
	abdata *abdata.API,
	individualsDB *mongodb.MongoConnection,
) *ResourceHandle {
	return &ResourceHandle{
		MetaCfg:       metaCfg,
		AuthCfg:       authCfg,
		ResetCfg:      resetCfg,
		Gorm:          gorm,
		SQLX:          sqlx,
		ABDATA:        abdata,
		IndibidualsDB: individualsDB,
	}
}
