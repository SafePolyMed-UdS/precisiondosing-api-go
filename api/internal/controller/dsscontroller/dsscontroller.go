package dsscontroller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/middleware"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/precheck"
	"precisiondosing-api-go/internal/utils/log"

	"github.com/gin-gonic/gin"
	cron "github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type DSSController struct {
	Meta           cfg.MetaConfig
	DB             *gorm.DB
	JSONValidators handle.JSONValidators
	Prechecker     *precheck.PreCheck
	logger         log.Logger
}

func New(resourceHandle *handle.ResourceHandle) *DSSController {
	return &DSSController{
		Meta:           resourceHandle.MetaCfg,
		DB:             resourceHandle.Databases.GormDB,
		Prechecker:     resourceHandle.Prechecker,
		JSONValidators: resourceHandle.JSONValidators,
		logger:         log.WithComponent("dsscontroller"),
	}
}

func (sc *DSSController) PostPrecheck(c *gin.Context) {
	patientData, err := sc.readPatientData(c)
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}

	result, precheckErr := sc.Prechecker.Check(patientData)
	if precheckErr != nil {
		// recoverable errors are errors that cannot be fixed by the user
		// e.g. Databases not reachable, or other system errors
		if precheckErr.Recoverable {
			handle.ServerError(c, precheckErr)
		} else {
			// non-recoverable errors are errors that can be fixed by the user
			handle.BadRequestError(c, precheckErr.Error())
		}
		return
	}

	sc.logger.Info("precheck successful",
		log.Str("endpoint", c.FullPath()),
		log.Str("ip", c.ClientIP()),
		log.Str("user-agent", c.Request.UserAgent()),
	)
	handle.Success(c, result)
}

func (sc *DSSController) PostAdjust(c *gin.Context) {
	patientData, err := sc.readPatientData(c)
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}

	marshalledData, _ := json.Marshal(patientData)
	newOrder := model.Order{OrderData: marshalledData}
	newOrder.UserID = middleware.UserID(c)

	if err = sc.DB.Create(&newOrder).Error; err != nil {
		handle.ServerError(c, err)
		return
	}

	type AdaptResponse struct {
		OrderID string `json:"order_id"`
		Message string `json:"message"`
	}

	result := AdaptResponse{
		OrderID: newOrder.OrderID,
		Message: "Order queued",
	}

	sc.logger.Info("adjustment queued",
		log.Str("orderID", newOrder.OrderID),
		log.Str("endpoint", c.FullPath()),
		log.Str("ip", c.ClientIP()),
		log.Str("user-agent", c.Request.UserAgent()),
	)
	handle.Success(c, result)
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

	// check if one (only one) drug has the "adjust_dose" flag set to true
	adjustDoseCount := 0
	for _, drug := range patientData.Drugs {
		if drug.AdjustDose {
			adjustDoseCount++
		}
	}

	if adjustDoseCount != 1 {
		return nil, errors.New("exactly one drug must have the 'adjust_dose' flag set to true")
	}

	return &patientData, nil
}
