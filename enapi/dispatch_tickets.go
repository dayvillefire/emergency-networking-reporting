package enapi

import (
	"context"
	"fmt"
)

// DispatchTicket represents a dispatch ticket record.
type DispatchTicket struct {
	ID                 string  `json:"id"`
	IncidentNumber     string  `json:"incident_number"`
	PCRNumber          string  `json:"pcr_number"`
	Address            string  `json:"address"`
	Apartment          *string `json:"apartment"`
	City               string  `json:"city"`
	State              string  `json:"state"`
	Zip                string  `json:"zip"`
	Longitude          string  `json:"longitude"`
	Latitude           string  `json:"latitude"`
	District           *string `json:"district"`
	Phone              *string `json:"phone"`
	InitialCall        Time    `json:"initial_call"`
	Notified           *Time   `json:"notified"`
	EnRoute            *Time   `json:"en_route"`
	OnScene            *Time   `json:"on_scene"`
	AtPatient          *Time   `json:"at_patient"`
	LeftForDestination *Time   `json:"left_for_destination"`
	AtDestination      *Time   `json:"at_destination"`
	Completed          *Time   `json:"completed"`
	Shift              string  `json:"shift"`
	Unit               string  `json:"unit"`
	Notes              string  `json:"notes"`
}

// DispatchUnitRequest represents a unit in a dispatch ticket create request.
type DispatchUnitRequest struct {
	Unit               string                `json:"unit"`
	Notified           Time                  `json:"notified"`
	EnRoute            *Time                 `json:"en_route,omitempty"`
	OnScene            *Time                 `json:"on_scene,omitempty"`
	AtPatient          *Time                 `json:"at_patient,omitempty"`
	LeftForDestination *Time                 `json:"left_for_destination,omitempty"`
	AtDestination      *Time                 `json:"at_destination,omitempty"`
	Completed          *Time                 `json:"completed,omitempty"`
	Crew               []DispatchCrewRequest `json:"crew,omitempty"`
}

// DispatchCrewRequest represents a crew member in a dispatch request.
type DispatchCrewRequest struct {
	Identifier string `json:"identifier"`
}

// DispatchNarrativeRequest represents a narrative in a dispatch create request.
type DispatchNarrativeRequest struct {
	WrittenAt     Time   `json:"written_at"`
	PersonnelName string `json:"personnel_name"`
	Narrative     string `json:"narrative"`
}

// CreateDispatchTicketRequest is the request body for creating dispatch tickets.
type CreateDispatchTicketRequest struct {
	IncidentNumber string                     `json:"incident_number"`
	InitialCall    Time                       `json:"initial_call"`
	PCRNumber      string                     `json:"pcr_number"`
	Address        string                     `json:"address"`
	City           string                     `json:"city"`
	State          string                     `json:"state"`
	Longitude      *string                    `json:"longitude,omitempty"`
	Latitude       *string                    `json:"latitude,omitempty"`
	Apartment      *string                    `json:"apartment,omitempty"`
	Zip            *string                    `json:"zip,omitempty"`
	District       *string                    `json:"district,omitempty"`
	Shift          *string                    `json:"shift,omitempty"`
	Station        *string                    `json:"station,omitempty"`
	CallType       *string                    `json:"call_type,omitempty"`
	IncidentType   *int                       `json:"incident_type,omitempty"`
	Notes          *string                    `json:"notes,omitempty"`
	Phone          *string                    `json:"phone,omitempty"`
	Units          []DispatchUnitRequest      `json:"units,omitempty"`
	Narratives     []DispatchNarrativeRequest `json:"narratives,omitempty"`
}

// UpdateDispatchTicketRequest is the request body for updating a dispatch ticket.
type UpdateDispatchTicketRequest struct {
	IncidentNumber     string `json:"incident_number"`
	Unit               string `json:"unit"`
	Notified           *Time  `json:"notified,omitempty"`
	EnRoute            *Time  `json:"en_route,omitempty"`
	OnScene            *Time  `json:"on_scene,omitempty"`
	AtPatient          *Time  `json:"at_patient,omitempty"`
	LeftForDestination *Time  `json:"left_for_destination,omitempty"`
	AtDestination      *Time  `json:"at_destination,omitempty"`
	Completed          *Time  `json:"completed,omitempty"`
}

// ListDispatchTickets returns all dispatch tickets.
func (c *Client) ListDispatchTickets(ctx context.Context) ([]DispatchTicket, error) {
	var dest []DispatchTicket
	if err := c.doGet(ctx, "/dispatch-tickets", nil, &dest); err != nil {
		return nil, fmt.Errorf("enapi: ListDispatchTickets: %w", err)
	}
	return dest, nil
}

// CreateDispatchTickets creates a new dispatch ticket with optional unit and narrative data.
func (c *Client) CreateDispatchTickets(ctx context.Context, req CreateDispatchTicketRequest) ([]DispatchTicket, error) {
	var dest []DispatchTicket
	if err := c.doPost(ctx, "/dispatch-tickets", req, &dest); err != nil {
		return nil, fmt.Errorf("enapi: CreateDispatchTickets: %w", err)
	}
	return dest, nil
}

// UpdateDispatchTicket updates an existing dispatch ticket (identified by incident_number + unit).
func (c *Client) UpdateDispatchTicket(ctx context.Context, req UpdateDispatchTicketRequest) (*DispatchTicket, error) {
	var dest DispatchTicket
	if err := c.doPut(ctx, "/dispatch-tickets", req, &dest); err != nil {
		return nil, fmt.Errorf("enapi: UpdateDispatchTicket: %w", err)
	}
	return &dest, nil
}
