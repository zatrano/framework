package repository

import (
	"fmt"
	"math"
)

// Page holds the result of a paginated query.
type Page[T any] struct {
	Items      []T        `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// PaginationMeta contains metadata about the current page.
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	From        int   `json:"from"`
	To          int   `json:"to"`
	HasPrev     bool  `json:"has_prev"`
	HasNext     bool  `json:"has_next"`
}

// NewPaginationMeta computes pagination metadata.
func NewPaginationMeta(currentPage, perPage int, total int64) PaginationMeta {
	if perPage <= 0 {
		perPage = 15
	}
	if currentPage <= 0 {
		currentPage = 1
	}
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages < 1 {
		totalPages = 1
	}
	from := (currentPage-1)*perPage + 1
	to := currentPage * perPage
	if int64(to) > total {
		to = int(total)
	}
	if total == 0 {
		from = 0
		to = 0
	}
	return PaginationMeta{
		CurrentPage: currentPage,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		From:        from,
		To:          to,
		HasPrev:     currentPage > 1,
		HasNext:     currentPage < totalPages,
	}
}

// Links generates HTML pagination links for template rendering.
//
//	meta.Links("/users", "?search=foo") // → slice of Link
type Link struct {
	Label  string
	URL    string
	Active bool
}

// Links returns a slice of pagination links for HTML rendering.
// baseURL is the route path (e.g. "/users") and queryString is the
// extra query string to append (e.g. "&search=alice").
func (m PaginationMeta) Links(baseURL, queryString string) []Link {
	var links []Link

	// Previous
	if m.HasPrev {
		links = append(links, Link{
			Label: "← Previous",
			URL:   fmt.Sprintf("%s?page=%d%s", baseURL, m.CurrentPage-1, queryString),
		})
	}

	// Page numbers
	start := m.CurrentPage - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > m.TotalPages {
		end = m.TotalPages
	}
	if end-start < 4 && start > 1 {
		start = end - 4
		if start < 1 {
			start = 1
		}
	}

	for i := start; i <= end; i++ {
		links = append(links, Link{
			Label:  fmt.Sprintf("%d", i),
			URL:    fmt.Sprintf("%s?page=%d%s", baseURL, i, queryString),
			Active: i == m.CurrentPage,
		})
	}

	// Next
	if m.HasNext {
		links = append(links, Link{
			Label: "Next →",
			URL:   fmt.Sprintf("%s?page=%d%s", baseURL, m.CurrentPage+1, queryString),
		})
	}

	return links
}

// ─── Pagination options ─────────────────────────────────────────────────────

// PaginateOpts configures a paginated query.
type PaginateOpts struct {
	Page    int
	PerPage int
}

// Normalize ensures sane defaults and clamps values.
func (o *PaginateOpts) Normalize() {
	if o.Page <= 0 {
		o.Page = 1
	}
	if o.PerPage <= 0 {
		o.PerPage = 15
	}
	if o.PerPage > 200 {
		o.PerPage = 200
	}
}

// Offset returns the SQL OFFSET for the given page options.
func (o *PaginateOpts) Offset() int {
	return (o.Page - 1) * o.PerPage
}
