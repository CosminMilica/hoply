// Package models holds the application and skip entities and their payloads.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Application is what a person did with one job. A status timestamp is nil
// when the application never entered that status.
type Application struct {
	JobID     uuid.UUID  `json:"job_id"`
	Status    string     `json:"status"`
	SavedAt   *time.Time `json:"saved_at"`
	CVReadyAt *time.Time `json:"cv_ready_at"`
	AppliedAt *time.Time `json:"applied_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type StatusInput struct {
	Status string `json:"status"`
}

type ApplicationList struct {
	Items []Application `json:"items"`
}

type SkipList struct {
	JobIDs []uuid.UUID `json:"job_ids"`
}
