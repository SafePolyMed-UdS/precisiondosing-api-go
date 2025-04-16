package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	OrderID       string           `gorm:"type:char(36);not null"`
	Order         json.RawMessage  `gorm:"type:json;not null"`
	ResultSuccess bool             `gorm:"type:bool"`
	ResultJSON    *json.RawMessage `gorm:"type:json"`
	ResultPDF     string           `gorm:"type:longtext"`
	CreatedAt     *time.Time       `gorm:"type:timestamp"`
	StartedAt     *time.Time       `gorm:"type:timestamp"`
	CompletedAt   *time.Time       `gorm:"type:timestamp"`
	SentAt        *time.Time       `gorm:"type:timestamp"`
}

func (j *Order) BeforeCreate(_ *gorm.DB) error {
	j.OrderID = uuid.New().String()
	return nil
}
