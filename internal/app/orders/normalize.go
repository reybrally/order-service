package orders

import (
	"strings"
	"time"
)

func NormalizeSearchFilters(f *SearchFilters) {
	if f == nil {
		return
	}

	now := time.Now().UTC()

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

	f.OrderUID = normPtr(f.OrderUID, withTrimCollapse)
	f.TrackNumber = normPtr(f.TrackNumber, func(s string) string {
		return strings.ToUpper(withTrimCollapse(s))
	})
	f.CustomerID = normPtr(f.CustomerID, withTrimCollapse)

	f.Provider = normPtr(f.Provider, func(s string) string {
		return strings.ToLower(withTrimCollapse(s))
	})

	f.Currency = normPtr(f.Currency, func(s string) string {
		return strings.ToUpper(withTrimCollapse(s))
	})

	if f.Query != nil {
		q := withTrimCollapse(*f.Query)
		if len(q) < 2 {
			f.Query = nil
		} else {
			f.Query = &q
		}
	}
}

func NormalizeRequest(p *PageRequest) {
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

func withTrimCollapse(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

func normPtr(p *string, norm func(string) string) *string {
	if p == nil {
		return nil
	}
	v := norm(*p)
	if v == "" {
		return nil
	}
	return &v
}
