package enapi

import (
	"context"
	"fmt"
)

// ResourceType represents an apparatus resource type classification.
type ResourceType struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// Apparatus represents a fire department apparatus/vehicle.
type Apparatus struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Identifier        string        `json:"identifier"`
	IdentifierType    string        `json:"identifier_type"`
	ResourceType      *ResourceType `json:"resource_type"`
	VehicleUnitNumber string        `json:"vehicle_unit_number"`
	ApparatusUse      FlexString    `json:"apparatus_use"`
	CallSign          string        `json:"call_sign"`
}

// CreateApparatusRequest is the request body for creating an apparatus.
type CreateApparatusRequest struct {
	Name              string  `json:"name"`
	Identifier        *string `json:"identifier,omitempty"`
	IdentifierType    *string `json:"identifier_type,omitempty"`
	ResourceTypeCode  string  `json:"resource_code"`
	VehicleUnitNumber *string `json:"vehicle_unit_number,omitempty"`
	CallSign          string  `json:"call_sign"`
}

// ListApparatus returns all apparatus.
func (c *Client) ListApparatus(ctx context.Context) ([]Apparatus, error) {
	var dest []Apparatus
	if err := c.doGet(ctx, "/apparatus", nil, &dest); err != nil {
		return nil, fmt.Errorf("enapi: ListApparatus: %w", err)
	}
	return dest, nil
}

// CreateApparatus creates a new apparatus.
func (c *Client) CreateApparatus(ctx context.Context, req CreateApparatusRequest) (*Apparatus, error) {
	var dest Apparatus
	if err := c.doPost(ctx, "/apparatus", req, &dest); err != nil {
		return nil, fmt.Errorf("enapi: CreateApparatus: %w", err)
	}
	return &dest, nil
}

// GetApparatus returns a single apparatus by ID.
func (c *Client) GetApparatus(ctx context.Context, id string) (*Apparatus, error) {
	var dest Apparatus
	if err := c.doGet(ctx, "/apparatus/"+id, nil, &dest); err != nil {
		return nil, fmt.Errorf("enapi: GetApparatus: %w", err)
	}
	return &dest, nil
}
