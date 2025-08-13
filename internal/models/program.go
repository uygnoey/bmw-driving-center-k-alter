package models

import "time"

// Program represents a driving program to monitor
type Program struct {
	Name     string   `yaml:"name" json:"name"`
	Keywords []string `yaml:"keywords" json:"keywords"`
	IsOpen   bool     `json:"is_open"`
	LastChecked time.Time `json:"last_checked"`
}

// ReservationStatus represents the current status of reservations
type ReservationStatus struct {
	Programs    []Program `json:"programs"`
	CheckedAt   time.Time `json:"checked_at"`
	HasOpenings bool      `json:"has_openings"`
}