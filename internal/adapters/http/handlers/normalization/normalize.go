package normalization

import (
	"github.com/reybrally/order-service/internal/app/orders"
	"strings"
	"time"
)

func NormalizeSearchFilters(f *orders.SearchFilters) {
	if f == nil {
		return
	}

	now := time.Now().UTC()

	// Преобразование CreatedFrom и CreatedTo в UTC
	if f.CreatedFrom != nil {
		t := toUTCSeconds(*f.CreatedFrom)
		f.CreatedFrom = &t
	}

	if f.CreatedTo != nil {
		t := toUTCSeconds(*f.CreatedTo)
		if t.After(now) {
			t = now
		}
		f.CreatedTo = &t
	}

	if f.CreatedFrom != nil && f.CreatedTo != nil && f.CreatedFrom.After(*f.CreatedTo) {
		from := *f.CreatedTo
		to := *f.CreatedFrom
		f.CreatedFrom = &from
		f.CreatedTo = &to
	}

	f.OrderUID = normalizeString(f.OrderUID)
	f.TrackNumber = normalizeString(f.TrackNumber, func(s string) string { return strings.ToUpper(s) })
	f.CustomerID = normalizeString(f.CustomerID)
	f.Provider = normalizeString(f.Provider, func(s string) string { return strings.ToLower(s) })
	f.Currency = normalizeString(f.Currency, func(s string) string { return strings.ToUpper(s) })

	if f.Query != nil {
		q := strings.TrimSpace(*f.Query)
		if len(q) < 2 {
			f.Query = nil
		} else {
			f.Query = &q
		}
	}
}

func NormalizeRequest(p *orders.PageRequest) {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.SortBy == "" {
		p.SortBy = "date_created"
	}
	if p.SortDir == "" {
		p.SortDir = "desc"
	}
}

func toUTCSeconds(t time.Time) time.Time {
	return t.UTC().Truncate(time.Second)
}

func normalizeString(p *string, normFunc ...func(string) string) *string {
	if p == nil {
		return nil
	}

	result := strings.TrimSpace(*p)
	if len(result) == 0 {
		return nil
	}

	for _, norm := range normFunc {
		result = norm(result)
	}

	return &result
}
