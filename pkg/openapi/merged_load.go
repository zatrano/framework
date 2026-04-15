package openapi

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// LoadMergedDocument merges the base OpenAPI file with built-in routes, parses YAML, and validates.
// basePath may point to a missing file (minimal doc is used). Returns raw YAML and the validated doc.
func LoadMergedDocument(ctx context.Context, basePath string) (raw []byte, doc *openapi3.T, err error) {
	raw, err = MergedYAMLBytes(basePath)
	if err != nil {
		return nil, nil, fmt.Errorf("merge: %w", err)
	}
	loader := openapi3.NewLoader()
	doc, err = loader.LoadFromData(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("parse merged: %w", err)
	}
	if err := doc.Validate(ctx); err != nil {
		return nil, nil, fmt.Errorf("invalid OpenAPI: %w", err)
	}
	return raw, doc, nil
}

