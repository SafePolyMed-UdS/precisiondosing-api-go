package dsscontroller

import (
	"fmt"
	"strings"
	"time"
)

type PatientData struct {
	PatientID              int                    `json:"patient_id" binding:"required"`
	PatientCharacteristics PatientCharacteristics `json:"patient_characteristics" binding:"required"`
	PatientPGXProfile      []PGXProfile           `json:"patient_pgx_profile"`
	Drugs                  []Drug                 `json:"drugs" binding:"required,dive,required"`
}

type PatientCharacteristics struct {
	Age           int     `json:"age" binding:"required"`
	Weight        float64 `json:"weight" binding:"required"`
	Height        int     `json:"height" binding:"required"`
	Sex           string  `json:"sex" binding:"required"`
	Ethnicity     *string `json:"ethnicity"`
	KidneyDisease *bool   `json:"kidney_disease" binding:"required"`
	LiverDisease  *bool   `json:"liver_disease" binding:"required"`
}

type PGXProfile struct {
	Gene                 string `json:"gene" binding:"required"`
	Allele1              string `json:"allele1" binding:"required"`
	Allele1CNVMultiplier int    `json:"allele1_cnv_multiplier" binding:"required"`
	Allele2              string `json:"allele2" binding:"required"`
	Allele2CNVMultiplier int    `json:"allele2_cnv_multiplier" binding:"required"`
}

type Drug struct {
	ActiveSubstances []string    `json:"active_substances" binding:"required"`
	Product          *Product    `json:"product"`
	IntakeCycle      IntakeCycle `json:"intake_cycle" binding:"required"`
}

type Product struct {
	ProductName *string `json:"product_name"`
	ATC         *string `json:"atc"`
}

type IntakeCycle struct {
	// IntakeMode can be either "on_demand" or "regular"
	// If "on_demand", the "frequency", "frequency_modifier" and 'intakes' fields are not required
	// IntakeMode        string      `json:"intake_mode" binding:"required"`
	StartingAt        *CustomTime `json:"starting_at"`
	Frequency         *string     `json:"frequency" binding:"required"`
	FrequencyModifier *int        `json:"frequency_modifier" binding:"required"`
	Intakes           *[]Intake   `json:"intakes" binding:"required,dive,required"`
}

type Intake struct {
	RawTimeStr string  `json:"raw_time_str" binding:"required"`
	Cron       string  `json:"cron" binding:"required"`
	Dosage     float64 `json:"dosage" binding:"required"`
	DosageUnit string  `json:"dosage_unit" binding:"required"`
}

// CustomTime wraps time.Time to provide custom parsing
type CustomTime struct {
	time.Time
}

// UnmarshalJSON handles custom date format parsing
func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	// Remove quotes from the string
	str := strings.Trim(string(b), "\"")

	// Define expected format
	layout := "2006-01-02"

	// Parse the date
	t, err := time.Parse(layout, str)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	// Assign parsed time to CustomTime
	ct.Time = t
	return nil
}
