package precheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/mongodb"
	"precisiondosing-api-go/internal/pbpk"
	"precisiondosing-api-go/internal/utils/abdata"
	"slices"
	"sort"
	"strings"
)

type Result struct {
	Message           string                       `json:"message"`
	Compounds         map[string]bool              `json:"compounds"`
	Interactions      []abdata.CompoundInteraction `json:"interactions"`
	OrganImpairment   bool                         `json:"impairment"`
	VirtualIndividual json.RawMessage              `json:"virtual_individual"`
	ModelID           string                       `json:"model_id"`
}

type Error struct {
	Msg         string
	Recoverable bool
	Err         error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *Error) Unwrap() error {
	return e.Err
}

func NewError(msg string, recoverable bool, wrappedErr ...error) *Error {
	var err error
	if len(wrappedErr) > 0 {
		err = wrappedErr[0]
	}
	return &Error{
		Msg:         msg,
		Recoverable: recoverable,
		Err:         err,
	}
}

type PreCheck struct {
	mongoDB    *mongodb.MongoConnection
	ABDATA     *abdata.API
	PBPKModels []pbpk.Model
}

func New(mongoDB *mongodb.MongoConnection, abdata *abdata.API, pbpkModels []pbpk.Model) *PreCheck {
	return &PreCheck{
		mongoDB:    mongoDB,
		ABDATA:     abdata,
		PBPKModels: pbpkModels,
	}
}

func appendMsg(oldMsg string, newMsg string) string {
	if oldMsg != "" {
		oldMsg += "\n"
	}
	oldMsg += newMsg
	return oldMsg
}

// Error will only be returned if a check could not be performed
// E.g. MedInfo is down
func (p *PreCheck) Check(data *model.PatientData) (*Result, *Error) {
	response := &Result{}

	nDrugs := len(data.Drugs)
	if nDrugs == 0 {
		response.Message = "No drugs provided. No adjustment can be performed"
		return response, NewError("no drugs provided", false)
	}

	// get compounds (unique and lowercase)
	response.Compounds = drugCompounds(data)

	// Impairment check
	p.impairmentCheck(response, data)

	// MedInfo check
	err := p.abdataCheck(response)
	if err != nil {
		return response, err
	}

	// Virtual individual check
	err = p.virtualIndividualCheck(response, data)
	if err != nil {
		return response, err
	}

	// PBPK model check
	err = p.pbpkModelCheck(response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (p *PreCheck) pbpkModelCheck(resp *Result) *Error {
	// the potential victim that the user set to adjust
	var victim string
	for k, v := range resp.Compounds {
		if v {
			victim = k
			break
		}
	}

	if victim == "" {
		return NewError("no victim for adjustment found in compounds", false)
	}

	var perpetrators []string
	for _, interaction := range resp.Interactions {
		v := strings.ToLower(interaction.CompoundsL[0])
		if v == victim {
			ps := interaction.CompoundsR
			for _, p := range ps {
				perpetrators = append(perpetrators, strings.ToLower(p))
			}
		}
	}
	sort.Strings(perpetrators)

	// find the model that matches the victim and perpetrators
	var modelID string
	for _, model := range p.PBPKModels {
		if model.Victim == victim {
			if slices.Equal(model.Perpetrators, perpetrators) {
				modelID = model.ID
				break
			}
		}
	}

	if modelID == "" {
		resp.Message = appendMsg(resp.Message, "PBPK Model Check: No model found for victim and perpetrators")
		return NewError("no model found for victim and perpetrators", false)
	}

	resp.ModelID = modelID
	return nil
}

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

func (p *PreCheck) virtualIndividualCheck(resp *Result, data *model.PatientData) *Error {
	age := data.PatientCharacteristics.Age
	weight := int(math.Round(data.PatientCharacteristics.Weight))
	height := data.PatientCharacteristics.Height
	sex := data.PatientCharacteristics.Sex
	population := data.PatientCharacteristics.Ethnicity

	individualPayload, err := p.mongoDB.FetchIndividual(population, sex, age, height, weight)
	if err != nil {
		return NewError("fetching individual", true, err)
	}

	trimmed := bytes.TrimSpace(individualPayload)
	preCheckSuccess := len(trimmed) > 0 &&
		!bytes.Equal(trimmed, []byte("[]")) &&
		!bytes.Equal(trimmed, []byte("{}")) &&
		!bytes.Equal(trimmed, []byte("null"))

	if !preCheckSuccess {
		resp.Message = appendMsg(resp.Message, "Virtual Individual Check: No virtual individual matched demographic data")
		return NewError("no virtual individual matched demographic data", false)
	}

	resp.VirtualIndividual = individualPayload
	return nil
}

// gets all active substances from the drugs -> lowercase -> unique
func drugCompounds(data *model.PatientData) map[string]bool {
	compounds := map[string]bool{}
	for _, drug := range data.Drugs {
		compoundsList := drug.ActiveSubstances
		adjust := drug.AdjustDose
		for _, c := range compoundsList {
			compounds[strings.ToLower(c)] = adjust
		}
	}

	return compounds
}

func (p *PreCheck) abdataCheck(resp *Result) *Error {
	compounds := resp.Compounds
	if len(compounds) < 2 {
		resp.Message = appendMsg(resp.Message, "MedInfo Check: Less than 2 compounds. No interaction check performed")
		return nil
	}

	compoundNames := slices.Sorted(maps.Keys(compounds))
	interactions, err := p.ABDATA.GetCommpoundInteractions(compoundNames)
	resp.Interactions = interactions
	if err != nil {
		resp.Message = appendMsg(resp.Message, "MedinfoCheck: Failed to fetch interactions")
		return NewError("fetching interactions", true, err)
	}

	if len(interactions) == 0 {
		resp.Message = appendMsg(resp.Message, "MedInfo Check: No interactions expected")
	}

	return nil
}
