package dsscontroller

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/pbpk"
	"precisiondosing-api-go/internal/utils/abdata"
	"slices"
	"sort"
	"strings"

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
	result, err := PreCheck(patientData, sc.ABDATA, sc.PBPKModels, sc.IndibidualsDB)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, result)
}

func PreCheck(data *PatientData,
	abdata *abdata.API,
	models []pbpk.Model,
	idb *mongodb.MongoConnection,
) (*PreCheckResponse, error) {
	response := &PreCheckResponse{}

	// Check if there are any drugs
	if len(data.Drugs) == 0 {
		response.Message = "No drugs found"
		response.Code = "PC-ERR-ND"
		return response, nil
	}

	// Impairment check
	if !impairmentCheck(response, data) {
		return response, nil
	}

	// MedInfo check
	ok, interactions, err := abdataCheck(response, data, abdata)
	if !ok || err != nil {
		return response, err
	}

	// Check for matching PBPK models
	ok, err = PBPKModelCheck(response, interactions, models)
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

func PBPKModelCheck(resp *PreCheckResponse,
	interactions []abdata.CompoundInteraction,
	models []pbpk.Model,
) (bool, error) {
	// this this from MedInfo Check (victim -> perpetrators)
	victimMap := map[string][]string{}

	// Create a map of victims and perpetrators
	for _, interaction := range interactions {
		victim := strings.ToLower(interaction.CompoundsL[0])
		perpetrators := interaction.CompoundsR
		for _, perpetrator := range perpetrators {
			perpetrator = strings.ToLower(perpetrator)
			if _, ok := victimMap[perpetrator]; !ok {
				victimMap[perpetrator] = []string{}
			} else {
				victimMap[perpetrator] = append(victimMap[perpetrator], victim)
			}
		}
	}

	// sort perpetrators
	for _, v := range victimMap {
		sort.Strings(v)
	}

	// Check if any perpetrator is also a victim in the map
	for _, v := range victimMap {
		for _, p := range v {
			if _, ok := victimMap[p]; ok {
				resp.Message = "Perpetrator is also a victim"
				resp.Code = "PC-ERR-PV"
				return false, nil
			}
		}
	}

	// A -> B
	// B, D -> C
	// -->
	// A, B -> C

	// check for appropriate models that match the victim and perpetrators
	for victim, perpetrators := range victimMap {
		found := false
		for _, model := range models {
			if model.Victim == victim {
				if slices.Equal(model.Perpetrators, perpetrators) {
					found = true
					break
				}
			}
		}

		if !found {
			detailsMap := map[string]interface{}{
				"victim":       victim,
				"perpetrators": perpetrators,
			}

			detailsJSON, _ := json.Marshal(detailsMap)

			resp.Message = "No appropriate PBPK model found for interaction"
			resp.Code = "PC-ERR-MM"
			resp.Details = json.RawMessage(detailsJSON)
			return found, nil
		}
	}

	return true, nil
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

func abdataCheck(resp *PreCheckResponse, data *PatientData, api *abdata.API) (bool, []abdata.CompoundInteraction, error) {
	compounds := drugCompounds(data)
	if len(compounds) < 2 {
		resp.Message = "Less than 2 compounds. No interactions expected"
		resp.Code = "PC-OK-AD-NI"
		return true, nil, nil
	}

	interactions, err := api.GetCommpoundInteractions(compounds)
	if err != nil {
		resp.Message = "ABDATA precheck error"
		if err.IsHTTPError() {
			switch err.StatusCode {
			case http.StatusNotFound:
				resp.Details = json.RawMessage(`"ABDATA service not available"`)
				resp.Code = "PC-ERR-AD-NF"
				return false, interactions, nil
			case http.StatusUnauthorized:
				resp.Details = json.RawMessage(`"Unexpected ABDATA error"`)
				resp.Code = "PC-ERR-AD-UA"
				return false, interactions, nil
			}
		}
		return false, interactions, err
	}

	if len(interactions) == 0 {
		resp.Message += "Medinfo: No interactions expected"
		resp.Code = "PC-OK-AD-NI"
	}

	return true, interactions, nil
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
