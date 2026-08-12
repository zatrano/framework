package pagination

// Simple holds a page of results without a total count.
type Simple[T any] struct {
	Data        []T    `json:"data"`
	CurrentPage int    `json:"current_page"`
	PerPage     int    `json:"per_page"`
	From        int    `json:"from"`
	To          int    `json:"to"`
	Path        string `json:"path,omitempty"`
	NextPageURL string `json:"next_page_url,omitempty"`
	PrevPageURL string `json:"prev_page_url,omitempty"`
	HasMore     bool   `json:"has_more"`
}

// NewSimple creates a simple paginator. hasMore indicates another page exists.
func NewSimple[T any](items []T, page, perPage int, path string, hasMore bool) *Simple[T] {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}

	from := 0
	to := 0
	if len(items) > 0 {
		from = (page-1)*perPage + 1
		to = from + len(items) - 1
	}

	p := &Simple[T]{
		Data:        items,
		CurrentPage: page,
		PerPage:     perPage,
		From:        from,
		To:          to,
		Path:        path,
		HasMore:     hasMore,
	}
	if hasMore {
		p.NextPageURL = pageURL(path, page+1)
	}
	if page > 1 {
		p.PrevPageURL = pageURL(path, page-1)
	}
	return p
}
