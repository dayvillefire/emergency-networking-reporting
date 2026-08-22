package enapi

import (
	"context"
	"fmt"
)

// User represents a department member.
type User struct {
	ID                 int     `json:"id"`
	FirstName          string  `json:"first_name"`
	MiddleName         string  `json:"middle_name"`
	LastName           string  `json:"last_name"`
	Username           string  `json:"username"`
	Email              string  `json:"email"`
	Address1           string  `json:"address_1"`
	Address2           string  `json:"address_2"`
	City               string  `json:"city"`
	State              string  `json:"state"`
	Zip                string  `json:"zip"`
	Phone              string  `json:"phone"`
	Gender             int     `json:"gender"`
	BirthDate          Date    `json:"birth_date"`
	Active             IntBool `json:"active"`
	PersonnelID        string  `json:"personnel_id"`
	Affiliation        int     `json:"affiliation"`
	UsualAssignment    int     `json:"usual_assignment"`
	Rank               string  `json:"rank"`
	EmploymentStatus   string  `json:"employment_status"`
	EmploymentStatusAt Time    `json:"employment_status_at"`
	CreatedAt          Time    `json:"created_at"`
	UpdatedAt          Time    `json:"updated_at"`
}

// ListUsers returns all department users.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var dest []User
	if err := c.doGet(ctx, "/users", nil, &dest); err != nil {
		return nil, fmt.Errorf("enapi: ListUsers: %w", err)
	}
	return dest, nil
}
