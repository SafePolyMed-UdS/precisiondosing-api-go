package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	OrderID string `gorm:"type:char(36);not null;uniqueIndex"` // UUID

	// Input
	OrderData json.RawMessage `gorm:"type:json;not null"` // Original input

	// Precheck stage
	PrecheckResult *json.RawMessage `gorm:"type:json"`      // Result from precheck (success or error JSON)
	PrecheckPassed bool             `gorm:"default:false"`  // Did precheck succeed?
	PrecheckedAt   *time.Time       `gorm:"type:timestamp"` // When precheck completed

	// Processing (R job)
	ProcessResultPDF    *string    `gorm:"type:longtext"`  // Result PDF (success or fallback error PDF)
	DoseAdjusted        bool       `gorm:"default:false"`  // Was the dose adjusted?
	ProcessErrorMessage *string    `gorm:"type:text"`      // Error if R process fails (System error -> no PDF)
	ProcessedAt         *time.Time `gorm:"type:timestamp"` // When processing completed

	// Sending stage
	SentAt    *time.Time `gorm:"type:timestamp"` // When sent
	SendTries int        `gorm:"default:0"`      // How many tries to send

	// Status
	// queued -> staged -> prechecked -> processing -> (processed, error) -> sent
	// !!!! error -> system error -> no PDF
	Status string `gorm:"type:varchar(50);not null;default:'queued'"`
}

func (j *Order) BeforeCreate(_ *gorm.DB) error {
	j.OrderID = uuid.New().String()
	return nil
}
