package shared

import (
	"context"
	"fmt"
	"time"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
)

// BuildNameMap calls ListUsers and returns a map from PersonnelID and database
// ID to "Last, First" display name. Returns nil on error.
func BuildNameMap(client *enapi.Client) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	users, err := client.ListUsers(ctx)
	if err != nil {
		return nil
	}
	m := make(map[string]string, len(users))
	for _, u := range users {
		name := fmt.Sprintf("%s, %s", u.LastName, u.FirstName)
		if u.PersonnelID != "" {
			m[u.PersonnelID] = name
		}
		m[fmt.Sprintf("%d", u.ID)] = name
	}
	return m
}
