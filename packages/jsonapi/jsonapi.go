package jsonapi

import (
	"fmt"

	"github.com/zatrano/framework/packages/http"
)

// Resource is a JSON:API resource object.
type Resource struct {
	Type          string         `json:"type"`
	ID            string         `json:"id"`
	Attributes    map[string]any `json:"attributes,omitempty"`
	Relationships map[string]any `json:"relationships,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
}

// ErrorObject is a JSON:API error.
type ErrorObject struct {
	Status string `json:"status,omitempty"`
	Code   string `json:"code,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Document is a JSON:API top-level document.
type Document struct {
	Data     any            `json:"data,omitempty"`
	Errors   []ErrorObject  `json:"errors,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
	Included []Resource     `json:"included,omitempty"`
	Links    map[string]any `json:"links,omitempty"`
	JSONAPI  map[string]any `json:"jsonapi,omitempty"`
}

// Make builds a single resource.
func Make(resourceType, id string, attributes map[string]any) Resource {
	return Resource{Type: resourceType, ID: id, Attributes: attributes}
}

// Collection builds a resource collection.
func Collection(items []Resource) []Resource {
	if items == nil {
		return []Resource{}
	}
	return items
}

// One wraps a single resource document.
func One(resource Resource) Document {
	return Document{
		Data:    resource,
		JSONAPI: map[string]any{"version": "1.0"},
	}
}

// Many wraps a collection document.
func Many(items []Resource) Document {
	return Document{
		Data:    Collection(items),
		JSONAPI: map[string]any{"version": "1.0"},
	}
}

// ErrorDoc builds an error document.
func ErrorDoc(status int, title, detail string) Document {
	return Document{
		Errors: []ErrorObject{{
			Status: fmt.Sprint(status),
			Title:  title,
			Detail: detail,
		}},
		JSONAPI: map[string]any{"version": "1.0"},
	}
}

// Response returns a JSON:API HTTP response.
func Response(doc Document, status ...int) *http.Response {
	code := 200
	if len(status) > 0 {
		code = status[0]
	}
	if len(doc.Errors) > 0 && len(status) == 0 {
		code = 400
	}
	resp := http.JSON(doc).Status(code)
	resp.SetContent(resp.Content(), "application/vnd.api+json")
	return resp
}
