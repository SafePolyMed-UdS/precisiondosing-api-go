package dsscontroller

import (
	"encoding/json"
	"fmt"
	"io"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/utils/abdata"

	"github.com/gin-gonic/gin"
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

func (sc *DSSController) readPatientData(c *gin.Context) (*PatientData, error) {
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
	patientData := PatientData{}
	if err = json.Unmarshal(bodyBytes, &patientData); err != nil {
		return nil, fmt.Errorf("invalid body JSON structure: %w", err)
	}

	return &patientData, nil
}
