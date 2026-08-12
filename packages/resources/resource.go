package resources

import (
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/pagination"
)

// Transformer converts a model into an API array.
type Transformer[T any] func(item T) map[string]any

// Make transforms a single item.
func Make[T any](item T, transform Transformer[T]) map[string]any {
	return transform(item)
}

// Collection transforms many items.
func Collection[T any](items []T, transform Transformer[T]) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, transform(item))
	}
	return out
}

// Wrap wraps data under a "data" key.
func Wrap(data any) map[string]any {
	return map[string]any{"data": data}
}

// WrapCollection wraps a transformed collection.
func WrapCollection[T any](items []T, transform Transformer[T]) map[string]any {
	return Wrap(Collection(items, transform))
}

// Paginate transforms a length-aware paginator.
func Paginate[T any](page *pagination.LengthAware[T], transform Transformer[T]) map[string]any {
	return map[string]any{
		"data":           Collection(page.Data, transform),
		"current_page":   page.CurrentPage,
		"per_page":       page.PerPage,
		"total":          page.Total,
		"last_page":      page.LastPage,
		"from":           page.From,
		"to":             page.To,
		"path":           page.Path,
		"first_page_url": page.FirstPageURL,
		"last_page_url":  page.LastPageURL,
		"next_page_url":  page.NextPageURL,
		"prev_page_url":  page.PrevPageURL,
	}
}

// JSON returns a JSON response for a single transformed item.
func JSON[T any](item T, transform Transformer[T]) *http.Response {
	return http.JSON(Wrap(Make(item, transform)))
}

// JSONCollection returns a JSON response for a collection.
func JSONCollection[T any](items []T, transform Transformer[T]) *http.Response {
	return http.JSON(WrapCollection(items, transform))
}

// JSONPaginated returns a JSON response for a paginator.
func JSONPaginated[T any](page *pagination.LengthAware[T], transform Transformer[T]) *http.Response {
	return http.JSON(Paginate(page, transform))
}

// Merge merges additional attributes into a resource array.
func Merge(resource map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(resource)+len(extra))
	for key, value := range resource {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

// When includes a key only when condition is true.
func When(condition bool, key string, value any) map[string]any {
	if !condition {
		return map[string]any{}
	}
	return map[string]any{key: value}
}
