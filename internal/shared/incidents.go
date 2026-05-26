package shared

import (
	"context"
	"sync"
	"time"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
)

// FetchIncidents fetches all incidents from both NERIS and NFIRS endpoints
// within the given date range. Paginates 100 per page and filters by PSAPTime.
func FetchIncidents(client *enapi.Client, start, end time.Time) ([]NormalizedIncident, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	var mu sync.Mutex
	var nerisIncidents, incIncidents []NormalizedIncident
	var nerisErr, incErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for page := 1; ; page++ {
			resp, err := client.ListNerisIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(page))
			if err != nil {
				nerisErr = err
				return
			}
			for _, inc := range resp.Data {
				t := inc.IncidentPsapTime.Time
				if !t.IsZero() && (t.Equal(start) || t.After(start)) && t.Before(end.Add(24*time.Hour)) {
					mu.Lock()
					nerisIncidents = append(nerisIncidents, NormNerisIncident(inc))
					mu.Unlock()
				}
			}
			if len(resp.Data) < 100 || page*100 >= resp.Total {
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		zeroStreak := 0
		for page := 1; ; page++ {
			resp, err := client.ListIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(page))
			if err != nil {
				incErr = err
				return
			}
			var count int
			for _, inc := range resp.Data {
				if inc.IncidentType == "" {
					continue
				}
				t := inc.Psap.Time
				if !t.IsZero() && (t.Equal(start) || t.After(start)) && t.Before(end.Add(24*time.Hour)) {
					mu.Lock()
					incIncidents = append(incIncidents, NormIncident(inc))
					count++
					mu.Unlock()
				}
			}
			if len(resp.Data) < 100 || page*100 >= resp.Total {
				break
			}
			if count == 0 {
				zeroStreak++
				if zeroStreak >= 5 {
					break
				}
			} else {
				zeroStreak = 0
			}
		}
	}()

	wg.Wait()
	if nerisErr != nil {
		return nil, nerisErr
	}
	if incErr != nil {
		return nil, incErr
	}
	return append(nerisIncidents, incIncidents...), nil
}
