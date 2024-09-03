package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	OrderID     string           `gorm:"type:char(36);not null"`
	Order       json.RawMessage  `gorm:"type:json;not null"`
	Output      *json.RawMessage `gorm:"type:json"`
	CompletedAt *time.Time       `gorm:"type:timestamp"`
}

func (j *Order) BeforeCreate(_ *gorm.DB) error {
	j.OrderID = uuid.New().String()
	return nil
}
