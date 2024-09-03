package dsscontroller

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/utils/abdata"

	"github.com/gin-gonic/gin"
)

type PreCheckResponse struct {
	Adaption bool            `json:"adaption_possible"`
	Message  string          `json:"message"`
	Details  json.RawMessage `json:"details"`
	Code     string          `json:"response_code"`
}

func (sc *DSSController) PostPrecheck(c *gin.Context) {
	query := PatientData{}

	if !handle.JSONBind(c, &query) {
		return
	}

	// TODO: Validate JSON

	result, err := preCheck(&query, sc.ABDATA, sc.IndibidualsDB)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func preCheck(data *PatientData, abdata *abdata.API, idb *mongodb.MongoConnection) (*PreCheckResponse, error) {
	response := &PreCheckResponse{}

	// Drug count check
	if !drugCountCheck(response, data) {
		return response, nil
	}

	// Impairment check
	if !impairmentCheck(response, data) {
		return response, nil
	}

	// Virtual individual check
	ok, err := virtualIndividualCheck(response, data, idb)
	if !ok || err != nil {
		return response, err
	}

	// ABData check
	ok, err = abdataCheck(response, data, abdata)
	if !ok || err != nil {
		return response, err
	}

	response.Adaption = true
	response.Message = "Adaption possible"
	response.Code = "PC-OK"

	return response, nil
}

func virtualIndividualCheck(resp *PreCheckResponse, data *PatientData, m *mongodb.MongoConnection) (bool, error) {
	age := data.PatientCharacteristics.Age
	weight := int(math.Round(data.PatientCharacteristics.Weight))
	height := data.PatientCharacteristics.Height
	sex := data.PatientCharacteristics.Sex
	// TODO: Map ethnicity to PK-Sim population
	population := *data.PatientCharacteristics.Ethnicity

	individualPayload, err := m.FetchIndividual(population, sex, age, height, weight)
	if err != nil {
		return false, fmt.Errorf("error fetching individual: %w", err)
	}
	str := string(individualPayload)

	hasData := individualPayload != nil && str != "\"[]\""

	if !hasData {
		resp.Message = "No virtual individual found that matches the patient characteristics"
		resp.Code = "PC-ERR-VI"
	}

	return hasData, nil
}

func abdataCheck(resp *PreCheckResponse, data *PatientData, api *abdata.API) (bool, error) {
	compounds := drugCompounds(data)

	interactions, err := api.GetCommpoundInteractions(compounds)
	if err != nil {
		if err.IsHTTPError() {
			if err.StatusCode == http.StatusNotFound {
				resp.Message = "ABDATA precheck error"
				resp.Details = err.Message
				resp.Code = "PC-ERR-AD"
				return false, nil
			}
		}
		return false, err
	}

	if len(interactions.Interactions) == 0 {
		resp.Message = "No interactions expected"
		resp.Code = "PC-ERR-IC"
	}

	return true, nil
}

func drugCountCheck(resp *PreCheckResponse, data *PatientData) bool {
	if len(data.Drugs) < 2 {
		resp.Message = "Patient takes less than 2 drugs"
		resp.Code = "PC-ERR-DC"
		return false
	}

	return true
}

func impairmentCheck(resp *PreCheckResponse, data *PatientData) bool {
	ld := *data.PatientCharacteristics.LiverDisease
	kd := *data.PatientCharacteristics.KidneyDisease

	if ld || kd {
		resp.Message = "Patient has liver or kidney disease"
		resp.Code = "PC-ERR-OI"
		return false
	}

	return true
}
