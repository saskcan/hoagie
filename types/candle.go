package types

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// Candle represents a candle
type Candle struct {
	ProductID uuid.UUID `json:"product_id"`
	Date      time.Time `json:"date"`
	Frequency string    `json:"frequency"`
	Open      float32   `json:"open"`
	High      float32   `json:"high"`
	Low       float32   `json:"low"`
	Close     float32   `json:"close"`
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	Volume    uint      `json:"volume"`
}
