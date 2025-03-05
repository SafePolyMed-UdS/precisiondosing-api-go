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
	// Read the patient data from the request body
	patientData, err := sc.readPatientData(c)
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}

	// Precheck
	result, err := PreCheck(patientData, sc.ABDATA, sc.IndibidualsDB)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, result)
}

func PreCheck(data *PatientData, abdata *abdata.API, idb *mongodb.MongoConnection) (*PreCheckResponse, error) {
	response := &PreCheckResponse{}

	// Impairment check
	if !impairmentCheck(response, data) {
		return response, nil
	}

	// ABData check
	ok, err := abdataCheck(response, data, abdata)
	if !ok || err != nil {
		return response, err
	}

	// Virtual individual check
	ok, err = virtualIndividualCheck(response, data, idb)
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
	population := data.PatientCharacteristics.Ethnicity

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
	if len(compounds) < 2 {
		resp.Message = "Less than 2 compounds. No interactions expected"
		resp.Code = "PC-OK-AD-NI"
		return true, nil
	}

	interactions, err := api.GetCommpoundInteractions(compounds)
	if err != nil {
		resp.Message = "ABDATA precheck error"
		if err.IsHTTPError() {
			switch err.StatusCode {
			case http.StatusNotFound:
				resp.Details = json.RawMessage(`"ABDATA service not available"`)
				resp.Code = "PC-ERR-AD-NF"
				return false, nil
			case http.StatusUnauthorized:
				resp.Details = json.RawMessage(`"Unexpected ABDATA error"`)
				resp.Code = "PC-ERR-AD-UA"
				return false, nil
			}
		}
		return false, err
	}

	if len(interactions) == 0 {
		resp.Message += "Medinfo: No interactions expected"
		resp.Code = "PC-OK-AD-NI"
	}

	return true, nil
}

func impairmentCheck(resp *PreCheckResponse, data *PatientData) bool {
	ld := data.PatientCharacteristics.LiverDisease
	kd := data.PatientCharacteristics.KidneyDisease

	if ld || kd {
		resp.Message = "Patient has liver or kidney disease"
		resp.Code = "PC-ERR-OI"
		return false
	}

	return true
}
