package enapi

import (
	"context"
	"fmt"
	"net/url"
)

// CrewUser represents a user assigned to a crew schedule.
type CrewUser struct {
	ID               int    `json:"id"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	PersonnelID      string `json:"personnel_id"`
	Start            Time   `json:"start"`
	End              Time   `json:"end"`
	Notes            string `json:"notes"`
	CertificationName string `json:"certification_name"`
	IsHazMat         bool   `json:"isHazMat"`
	Rank             string `json:"rank"`
}

// CrewEquipment represents equipment assigned to a crew schedule.
type CrewEquipment struct {
	Name             string     `json:"name"`
	CallSign         string     `json:"call_sign"`
	PrimaryAction    string     `json:"primary_action"`
	SecondaryAction  string     `json:"secondary_action"`
	Users            []CrewUser `json:"users"`
}

// CrewSchedule represents a crew schedule record.
type CrewSchedule struct {
	ID        string          `json:"id"`
	Start     Time            `json:"start"`
	End       Time            `json:"end"`
	Notes     string          `json:"notes"`
	Equipment []CrewEquipment `json:"equipment"`
}

// CrewEquipmentRequest represents equipment in a create/update request.
type CrewEquipmentRequest struct {
	CallSign        string              `json:"call_sign"`
	PrimaryAction   *string             `json:"primary_action,omitempty"`
	SecondaryAction *string             `json:"secondary_action,omitempty"`
	Users           []CrewUserRequest   `json:"users"`
}

// CrewUserRequest represents a user in a create/update request.
type CrewUserRequest struct {
	PersonnelID string  `json:"personnel_id,omitempty"`
	ID          *int    `json:"id,omitempty"`
	Start       Time    `json:"start"`
	End         Time    `json:"end"`
	Notes       *string `json:"notes,omitempty"`
}

// CreateCrewScheduleRequest is the request body for creating/updating a crew schedule.
type CreateCrewScheduleRequest struct {
	Start     Time                  `json:"start"`
	End       Time                  `json:"end"`
	Notes     *string               `json:"notes,omitempty"`
	Equipment []CrewEquipmentRequest `json:"equipment"`
}

// ListCrewSchedules returns all crew schedules with optional filtering.
func (c *Client) ListCrewSchedules(ctx context.Context, opts ...QueryOption) (*PaginatedResponse[CrewSchedule], error) {
	query := url.Values{}
	for _, opt := range opts {
		opt(query)
	}
	var dest PaginatedResponse[CrewSchedule]
	if err := c.doGet(ctx, "/crew-schedules", query, &dest); err != nil {
		return nil, fmt.Errorf("enapi: ListCrewSchedules: %w", err)
	}
	return &dest, nil
}

// CreateCrewSchedule creates or updates a crew schedule.
func (c *Client) CreateCrewSchedule(ctx context.Context, req CreateCrewScheduleRequest) (*CrewSchedule, error) {
	var dest CrewSchedule
	if err := c.doPost(ctx, "/crew-schedules", req, &dest); err != nil {
		return nil, fmt.Errorf("enapi: CreateCrewSchedule: %w", err)
	}
	return &dest, nil
}
