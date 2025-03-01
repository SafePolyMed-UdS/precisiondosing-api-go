package dsscontroller

import (
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/utils/abdata"

	"gorm.io/gorm"
)

type DSSController struct {
	Meta           cfg.MetaConfig
	DB             *gorm.DB
	ABDATA         *abdata.API
	IndibidualsDB  *mongodb.MongoConnection
	JSONValidators handle.JSONValidators
}

func NewDSSController(resourceHandle *handle.ResourceHandle) *DSSController {
	return &DSSController{
		Meta:           resourceHandle.MetaCfg,
		DB:             resourceHandle.Databases.GormDB,
		ABDATA:         resourceHandle.ABDATA,
		JSONValidators: resourceHandle.JSONValidators,
		IndibidualsDB:  resourceHandle.Databases.MongoDB,
	}
}

func drugCompounds(data *PatientData) []string {
	compounds := []string{}
	for _, drug := range data.Drugs {
		compounds = append(compounds, drug.ActiveSubstances...)
	}
	return compounds
}
