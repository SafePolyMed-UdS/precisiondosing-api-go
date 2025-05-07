package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusQueued     = "queued"
	StatusStaged     = "staged"
	StatusPrechecked = "prechecked"
	StatusProcessing = "processing"
	StatusProcessed  = "processed"
	StatusError      = "error"
	StatusSent       = "sent"
	StatusSendFailed = "send_failed"
)

type Order struct {
	gorm.Model
	OrderID string `gorm:"type:char(36);not null;uniqueIndex"` // UUID
	// Foreign key field
	UserID uint `gorm:"not null"`
	User   User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

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
	SentAt            *time.Time `gorm:"type:timestamp"`
	SendTries         int        `gorm:"default:0"`
	LastSendAttemptAt *time.Time `gorm:"type:timestamp"`
	LastSendError     *string    `gorm:"type:text"`
	NextSendAttemptAt *time.Time `gorm:"type:timestamp"`

	// queued -> staged -> prechecked -> processing -> (processed, error) -> (sent, send_failed)
	//
	// error -> system error (e.g., processing error, no PDF, send failed after retries)
	Status string `gorm:"type:varchar(50);not null;default:'queued'"`
}

func (j *Order) BeforeCreate(_ *gorm.DB) error {
	if j.OrderID == "" {
		j.OrderID = uuid.New().String()
	}
	return nil
}
