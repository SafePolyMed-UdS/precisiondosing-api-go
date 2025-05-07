package precheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"precisiondosing-api-go/internal/model"
	"precisiondosing-api-go/internal/pbpk"
	"precisiondosing-api-go/internal/services/individualdb"
	"precisiondosing-api-go/internal/services/medinfo"
	"precisiondosing-api-go/internal/utils/helper"
	"precisiondosing-api-go/internal/utils/log"
	"slices"
	"sort"
	"strings"
	"time"

	cron "github.com/robfig/cron"
)

type Intake struct {
	RawTimeStr  string  `json:"time_str"`
	Dosage      float64 `json:"dosage"`
	Formulation string  `json:"formulation"`
}

type Compound struct {
	Name        string   `json:"name"`
	NameInModel string   `json:"name_in_model"`
	Synonyms    []string `json:"synonyms"`
	Adjust      bool     `json:"adjust"`
	DoseAmount  float64  `json:"dose_amount"`
	DoseUnit    string   `json:"dose_unit"`
	Schedule    []Intake `json:"schedule"`
}

type Result struct {
	Message           string                        `json:"message"`
	Compounds         []Compound                    `json:"compounds"`
	Interactions      []medinfo.CompoundInteraction `json:"interactions"`
	OrganImpairment   bool                          `json:"impairment"`
	VirtualIndividual json.RawMessage               `json:"virtual_individual"`
	ModelID           string                        `json:"model_id"`
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
	mongoDB    *individualdb.IndividualDB
	MedInfoAPI *medinfo.API
	PBPKModels *pbpk.Models
	logger     log.Logger
}

func New(mongoDB *individualdb.IndividualDB, medinfoAPI *medinfo.API, pbpkModels *pbpk.Models) *PreCheck {
	return &PreCheck{
		mongoDB:    mongoDB,
		MedInfoAPI: medinfoAPI,
		PBPKModels: pbpkModels,
		logger:     log.WithComponent("precheck"),
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
		response.Message = "No drugs provided. No adjustment can be performed."
		return response, NewError("no drugs provided", false)
	}

	// get compounds (unique and lowercase)
	err := p.drugCompounds(response, data)
	if err != nil {
		return response, err
	}

	// get compound synonyms
	err = p.commpoundSynonyms(response)
	if err != nil {
		return response, err
	}

	// Impairment check
	p.impairmentCheck(response, data)

	// MedInfo check
	err = p.medinfoCheck(response)
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
	for _, c := range resp.Compounds {
		if c.Adjust {
			victim = c.Name
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
	for _, model := range p.PBPKModels.Definitions {
		if model.Victim == victim {
			if slices.Equal(model.Perpetrators, perpetrators) {
				modelID = model.ID
				break
			}
		}
	}

	if modelID == "" {
		resp.Message = appendMsg(resp.Message, "PBPK Model Check: No model found for victim and perpetrators.")
		return NewError("no model found for victim and perpetrators", false)
	}

	resp.ModelID = modelID
	return nil
}

func (p *PreCheck) impairmentCheck(resp *Result, data *model.PatientData) {
	ld := data.PatientCharacteristics.LiverDisease
	kd := data.PatientCharacteristics.KidneyDisease

	if ld {
		resp.Message = appendMsg(resp.Message, "Impairment Check: Liver disease.")
	}
	if kd {
		resp.Message = appendMsg(resp.Message, "Impairment Check: Kidney disease.")
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
		p.logger.Warn("no virtual individual matched demographic data",
			log.Str("sex", sex),
			log.Int("age", age),
			log.Int("weight", weight),
			log.Int("height", height),
			log.Str("ethnicity", helper.DerefOrDefault(population, "unknown")),
		)

		resp.Message = appendMsg(resp.Message, "Virtual Individual Check: No virtual individual matched demographic data.")
		return NewError("no virtual individual matched demographic data", false)
	}

	resp.VirtualIndividual = individualPayload
	return nil
}

// gets all active substances from the drugs -> lowercase -> unique
// also parse doses and schedules
func (p *PreCheck) drugCompounds(resp *Result, data *model.PatientData) *Error {
	compounds := map[string]Compound{}
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	for _, drug := range data.Drugs {
		compoundsList := drug.ActiveSubstances
		if len(compoundsList) > 1 {
			p.logger.Warn("multiple active substances in a single drug",
				log.Str("peoduct", helper.DerefOrDefault(drug.Product.ProductName, "unknown")),
				log.Str("active_substances", strings.Join(compoundsList, ",")),
			)

			resp.Message = appendMsg(resp.Message, "Multiple active substances in a single drug not supported")
			return NewError("multiple active substances in a single drug", false)
		}

		c := strings.ToLower(compoundsList[0])
		adjust := drug.AdjustDose
		amount := drug.Product.Dose
		unit := drug.Product.DoseUnit
		schedule := []Intake{}
		for _, intake := range drug.IntakeCycle.Intakes {
			s, _ := parser.Parse(intake.Cron)
			next := startOfDay
			for range make([]int, p.PBPKModels.MaxDoses) {
				next = s.Next(next)
				timeStr := next.Format("2006-01-02 15:04")
				schedule = append(schedule, Intake{
					RawTimeStr:  timeStr,
					Dosage:      intake.Dosage,
					Formulation: intake.DosageUnit,
				})
			}
		}

		compounds[c] = Compound{
			Name:       c,
			Adjust:     adjust,
			DoseAmount: amount,
			DoseUnit:   unit,
			Schedule:   schedule,
		}
	}

	for _, k := range compounds {
		resp.Compounds = append(resp.Compounds, k)
	}

	return nil
}

func (p *PreCheck) commpoundSynonyms(resp *Result) *Error {
	var compoundNames []string
	for _, compound := range resp.Compounds {
		compoundNames = append(compoundNames, compound.Name)
	}

	matches, err := p.MedInfoAPI.GetCommpoundSynonyms(compoundNames)
	if err != nil {
		if err.StatusCode == http.StatusNotFound {
			resp.Message = appendMsg(resp.Message, "MedInfo Check: "+err.Err.Error())
		}
		return NewError("fetching synonyms", err.StatusCode != http.StatusNotFound, err)
	}

	for _, match := range matches {
		var synonyms []string
		c := match.Input
		for _, m := range match.Matches {
			for _, s := range m {
				synonyms = append(synonyms, strings.ToLower(s.Name))
			}
		}
		slices.Sort(synonyms)
		for i := range resp.Compounds {
			if resp.Compounds[i].Name == c {
				resp.Compounds[i].Synonyms = synonyms
				break
			}
		}
	}

	return nil
}

func (p *PreCheck) medinfoCheck(resp *Result) *Error {
	compounds := resp.Compounds
	if len(compounds) < 2 {
		resp.Message = appendMsg(resp.Message, "MedInfo Check: Less than 2 compounds. No interaction check performed.")
		return nil
	}

	var compoundNames []string
	for _, compound := range compounds {
		compoundNames = append(compoundNames, compound.Name)
	}

	interactions, err := p.MedInfoAPI.GetCommpoundInteractions(compoundNames)
	resp.Interactions = interactions
	if err != nil {
		p.logger.Warn("medInfo interaction check:", log.Err(err))
		if err.StatusCode == http.StatusNotFound {
			resp.Message = appendMsg(resp.Message, "MedInfo Check: "+string(err.Error()))
		}
		return NewError("fetching interactions", !err.InputError, err)
	}

	if len(interactions) == 0 {
		resp.Message = appendMsg(resp.Message, "MedInfo Check: No interactions expected.")
	}

	return nil
}
