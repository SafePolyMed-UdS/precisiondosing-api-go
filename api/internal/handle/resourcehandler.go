package handle

import (
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/utils/callr"
	"precisiondosing-api-go/internal/utils/helper"
	"precisiondosing-api-go/internal/utils/precheck"
	"precisiondosing-api-go/internal/utils/validate"

	"gorm.io/gorm"
)

// Central struct to hold all the configurations and database connection pool.
type ResourceHandle struct {
	ServerCfg      cfg.ServerConfig
	MetaCfg        cfg.MetaConfig
	AuthCfg        cfg.AuthTokenConfig
	ResetCfg       cfg.ResetTokenConfig
	Databases      Databases
	JSONValidators JSONValidators
	Prechecker     *precheck.PreCheck
	CallR          *callr.CallR
	DebugMode      bool
}

type Databases struct {
	GormDB  *gorm.DB
	MongoDB *mongodb.MongoConnection
}

type JSONValidators struct {
	PreCheck *validate.JSONValidator
}

func NewResourceHandle(
	apiCfg *cfg.APIConfig,
	databases Databases,
	prechecker *precheck.PreCheck,
	callR *callr.CallR,
	jsonValidators JSONValidators,
	debug bool,
) *ResourceHandle {
	res := &ResourceHandle{
		ServerCfg:      apiCfg.Server,
		MetaCfg:        apiCfg.Meta,
		AuthCfg:        apiCfg.AuthToken,
		ResetCfg:       apiCfg.ResetToken,
		Databases:      databases,
		JSONValidators: jsonValidators,
		Prechecker:     prechecker,
		CallR:          callR,
		DebugMode:      debug,
	}

	res.MetaCfg.URL = helper.RemoveTrailingSlash(res.MetaCfg.URL)
	res.MetaCfg.Group = helper.RemoveTrailingSlash(res.MetaCfg.Group)
	res.MetaCfg.Group = helper.AddLeadingSlash(res.MetaCfg.Group)

	return res
}
