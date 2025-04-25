package precheck

import (
	"encoding/json"
	"fmt"
	"math"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/utils/abdata"
)

type Result struct {
	// Message to the user to explain briefly the reason for AdaptionPossible
	Message string `json:"message"`
	// Number of drugs in the patient data
	NDrugs            int                          `json:"n_drugs"`
	Interactions      []abdata.CompoundInteraction `json:"interactions"`
	OrganImpairment   bool                         `json:"impairment"`
	VirtualIndividual json.RawMessage              `json:"virtual_individual"`
}

type PreCheck struct {
	mongoDB *mongodb.MongoConnection
	ABDATA  *abdata.API
}

func New(mongoDB *mongodb.MongoConnection, abdata *abdata.API) PreCheck {
	return PreCheck{
		mongoDB: mongoDB,
		ABDATA:  abdata,
	}
}

func appendMsg(old_msg string, new_msg string) string {
	if old_msg != "" {
		old_msg += "\n"
	}
	old_msg += new_msg
	return old_msg
}

// Error will only be returned if a check could not be performed
// E.g. MedInfo is down
func (p *PreCheck) Check(data *model.PatientData) (*Result, error) {
	response := &Result{}

	response.NDrugs = len(data.Drugs)
	// Impairment check
	p.impairmentCheck(response, data)

	// MedInfo check
	err := p.abdataCheck(response, data)
	if err != nil {
		return response, fmt.Errorf("error in PreCheck: %w", err)
	}

	// Virtual individual check
	err = p.virtualIndividualCheck(response, data)
	if err != nil {
		return response, fmt.Errorf("error in PreCheck: %w", err)
	}

	// Check for matching PBPK models
	//ok, err = p.pbpkModelCheck(response, interactions)
	//if !ok || err != nil {
	//	return response, err
	//}

	return response, nil
}

// func PBPKModelCheck(resp *PreCheckResponse,
// 	interactions []abdata.CompoundInteraction,
// 	models []pbpk.Model,
// ) (bool, error) {
// 	// this this from MedInfo Check (victim -> perpetrators)
// 	victimMap := map[string][]string{}

// 	// Create a map of victims and perpetrators
// 	for _, interaction := range interactions {
// 		victim := strings.ToLower(interaction.CompoundsL[0])
// 		perpetrators := interaction.CompoundsR
// 		for _, perpetrator := range perpetrators {
// 			perpetrator = strings.ToLower(perpetrator)
// 			if _, ok := victimMap[perpetrator]; !ok {
// 				victimMap[perpetrator] = []string{}
// 			} else {
// 				victimMap[perpetrator] = append(victimMap[perpetrator], victim)
// 			}
// 		}
// 	}

// 	// sort perpetrators
// 	for _, v := range victimMap {
// 		sort.Strings(v)
// 	}

// 	// Check if any perpetrator is also a victim in the map
// 	for _, v := range victimMap {
// 		for _, p := range v {
// 			if _, ok := victimMap[p]; ok {
// 				resp.Message = "Perpetrator is also a victim"
// 				resp.Code = "PC-ERR-PV"
// 				return false, nil
// 			}
// 		}
// 	}

// 	// A -> B
// 	// B, D -> C
// 	// -->
// 	// A, B -> C

// 	// check for appropriate models that match the victim and perpetrators
// 	for victim, perpetrators := range victimMap {
// 		found := false
// 		for _, model := range models {
// 			if model.Victim == victim {
// 				if slices.Equal(model.Perpetrators, perpetrators) {
// 					found = true
// 					break
// 				}
// 			}
// 		}

// 		if !found {
// 			detailsMap := map[string]interface{}{
// 				"victim":       victim,
// 				"perpetrators": perpetrators,
// 			}

// 			detailsJSON, _ := json.Marshal(detailsMap)

// 			resp.Message = "No appropriate PBPK model found for interaction"
// 			resp.Code = "PC-ERR-MM"
// 			resp.Details = json.RawMessage(detailsJSON)
// 			return found, nil
// 		}
// 	}

// 	return true, nil
// }

func (p *PreCheck) impairmentCheck(resp *Result, data *model.PatientData) {
	ld := data.PatientCharacteristics.LiverDisease
	kd := data.PatientCharacteristics.KidneyDisease

	if ld {
		resp.Message = appendMsg(resp.Message, "Impairment Check: Liver disease")
	}
	if kd {
		resp.Message = appendMsg(resp.Message, "Impairment Check: Kidney disease")
	}

	resp.OrganImpairment = ld || kd
}

func (p *PreCheck) virtualIndividualCheck(resp *Result, data *model.PatientData) error {
	age := data.PatientCharacteristics.Age
	weight := int(math.Round(data.PatientCharacteristics.Weight))
	height := data.PatientCharacteristics.Height
	sex := data.PatientCharacteristics.Sex
	population := data.PatientCharacteristics.Ethnicity

	individualPayload, err := p.mongoDB.FetchIndividual(population, sex, age, height, weight)
	if err != nil {
		return fmt.Errorf("error fetching individual: %w", err)
	}
	str := string(individualPayload)
	preCheckSucess := individualPayload != nil && str != "\"[]\""

	if !preCheckSucess {
		resp.Message = appendMsg(resp.Message, "Virtual Individual Check: No virtual individual matched demographic data")
		return fmt.Errorf("No virtual individual matched demographic data")
	} else {
		resp.VirtualIndividual = individualPayload
	}

	return nil
}

func drugCompounds(data *model.PatientData) []string {
	compounds := []string{}
	for _, drug := range data.Drugs {
		compounds = append(compounds, drug.ActiveSubstances...)
	}
	return compounds
}

func (p *PreCheck) abdataCheck(resp *Result, data *model.PatientData) error {
	compounds := drugCompounds(data)
	if len(compounds) < 2 {
		resp.Message = appendMsg(resp.Message, "MedInfo Check: Less than 2 compounds. No interaction check performed")
		return nil
	}

	interactions, err := p.ABDATA.GetCommpoundInteractions(compounds)
	resp.Interactions = interactions
	if err != nil {
		resp.Message = appendMsg(resp.Message, "MedinfoCheck: Failed to fetch interactions")
		return fmt.Errorf("error fetching interactions: %w", err)
	}

	if len(interactions) == 0 {
		resp.Message = appendMsg(resp.Message, "MedInfo Check: No interactions expected")
	}

	return nil
}
