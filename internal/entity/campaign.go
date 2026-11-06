package entity

import (
	"time"
)

// Campaign represents an over-the-air software campaign scheduled for a fleet.
type Campaign struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	TargetVersion string    `json:"target_version"`
	FleetID       string    `json:"fleet_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
