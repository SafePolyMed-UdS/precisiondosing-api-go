package dsscontroller

import (
	"encoding/json"
	"fmt"
	"io"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/pbpk"
	"precisiondosing-api-go/internal/utils/abdata"

	"github.com/gin-gonic/gin"
	cron "github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type DSSController struct {
	Meta           cfg.MetaConfig
	DB             *gorm.DB
	ABDATA         *abdata.API
	IndibidualsDB  *mongodb.MongoConnection
	PBPKModels     []pbpk.Model
	JSONValidators handle.JSONValidators
}

func NewDSSController(resourceHandle *handle.ResourceHandle) *DSSController {
	return &DSSController{
		Meta:           resourceHandle.MetaCfg,
		DB:             resourceHandle.Databases.GormDB,
		ABDATA:         resourceHandle.ABDATA,
		PBPKModels:     resourceHandle.PBPKModels,
		JSONValidators: resourceHandle.JSONValidators,
		IndibidualsDB:  resourceHandle.Databases.MongoDB,
	}
}

func (sc *DSSController) readPatientData(c *gin.Context) (*model.PatientData, error) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	var jsonBody map[string]interface{}
	if err = json.Unmarshal(bodyBytes, &jsonBody); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}

	// Validate the JSON body
	err = sc.JSONValidators.PreCheck.Validate(jsonBody)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}

	// Bind the JSON body to the query struct
	patientData := model.PatientData{}
	if err = json.Unmarshal(bodyBytes, &patientData); err != nil {
		return nil, fmt.Errorf("invalid body JSON structure: %w", err)
	}

	// cron tab check
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	for _, drug := range patientData.Drugs {
		for _, intake := range drug.IntakeCycle.Intakes {
			_, err = parser.Parse(intake.Cron)
			if err != nil {
				return nil, fmt.Errorf("error parsing cron expression: %w", err)
			}
		}
	}

	return &patientData, nil
}
