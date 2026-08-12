package pagination

import (
	"math"
)

// LengthAware holds paginated results.
type LengthAware[T any] struct {
	Data         []T    `json:"data"`
	CurrentPage  int    `json:"current_page"`
	PerPage      int    `json:"per_page"`
	Total        int64  `json:"total"`
	LastPage     int    `json:"last_page"`
	From         int    `json:"from"`
	To           int    `json:"to"`
	Path         string `json:"path,omitempty"`
	FirstPageURL string `json:"first_page_url,omitempty"`
	LastPageURL  string `json:"last_page_url,omitempty"`
	NextPageURL  string `json:"next_page_url,omitempty"`
	PrevPageURL  string `json:"prev_page_url,omitempty"`
}

// New creates a length-aware paginator.
func New[T any](items []T, total int64, page, perPage int, path string) *LengthAware[T] {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage < 1 {
		lastPage = 1
	}

	from := 0
	to := 0
	if total > 0 && len(items) > 0 {
		from = (page-1)*perPage + 1
		to = from + len(items) - 1
	}

	p := &LengthAware[T]{
		Data:        items,
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		LastPage:    lastPage,
		From:        from,
		To:          to,
		Path:        path,
	}
	p.FirstPageURL = pageURL(path, 1)
	p.LastPageURL = pageURL(path, lastPage)
	if page < lastPage {
		p.NextPageURL = pageURL(path, page+1)
	}
	if page > 1 {
		p.PrevPageURL = pageURL(path, page-1)
	}
	return p
}

func pageURL(path string, page int) string {
	if path == "" {
		return ""
	}
	sep := "?"
	for i := 0; i < len(path); i++ {
		if path[i] == '?' {
			sep = "&"
			break
		}
	}
	return path + sep + "page=" + itoa(page)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := make([]byte, 0, 12)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
