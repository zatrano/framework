package openapi

import (
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// MergeFrameworkRoutes adds or replaces operations for built-in ZATRANO endpoints (docs stay accurate).
func MergeFrameworkRoutes(doc *openapi3.T) {
	type route struct {
		method, path, summary string
		tags                  []string
		status                int
		desc                  string
	}
	routes := []route{
		{http.MethodGet, "/health", "Liveness probe", []string{"Health"}, 200, "Process is up"},
		{http.MethodGet, "/ready", "Readiness probe", []string{"Health"}, 200, "Dependencies OK or 503"},
		{http.MethodGet, "/status", "Aggregated checks", []string{"Health"}, 200, "JSON status"},
		{http.MethodGet, "/", "Welcome JSON", []string{"Meta"}, 200, "Links to probes and API"},
		{http.MethodGet, "/api/v1/public/ping", "Public ping", []string{"API"}, 200, "pong"},
		{http.MethodGet, "/api/v1/private/me", "JWT echo (Bearer required)", []string{"API"}, 200, "claims"},
		{http.MethodPost, "/api/v1/auth/token", "Demo token (if enabled)", []string{"API"}, 200, "access_token"},
		{http.MethodGet, "/openapi.yaml", "Raw OpenAPI document", []string{"Meta"}, 200, "YAML"},
		{http.MethodGet, "/docs", "Scalar API browser", []string{"Meta"}, 200, "HTML"},
		{http.MethodGet, "/auth/oauth/{provider}/login", "Start OAuth2 login", []string{"OAuth"}, 302, "Redirect to provider"},
		{http.MethodGet, "/auth/oauth/{provider}/callback", "OAuth2 callback", []string{"OAuth"}, 302, "Redirect to app root"},
	}
	for _, r := range routes {
		op := openapi3.NewOperation()
		op.Summary = r.summary
		op.Tags = r.tags
		if strings.Contains(r.path, "{provider}") {
			op.Parameters = openapi3.Parameters{
				&openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						In:       openapi3.ParameterInPath,
						Name:     "provider",
						Required: true,
						Schema:   openapi3.NewStringSchema().NewRef(),
					},
				},
			}
		}
		switch {
		case r.path == "/" && r.method == http.MethodGet:
			op.Responses = responsesRootIndex()
		case r.path == "/status" && r.method == http.MethodGet:
			op.Responses = responsesStatusJSON()
		default:
			op.Responses = openapi3.NewResponses(
				openapi3.WithStatus(r.status, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().WithDescription(r.desc),
				}),
			)
		}
		doc.AddOperation(r.path, r.method, op)
	}
}

func schemaRef(s *openapi3.Schema) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: s}
}

func responsesRootIndex() *openapi3.Responses {
	endpoints := openapi3.NewObjectSchema().WithAdditionalProperties(openapi3.NewStringSchema())
	s := openapi3.NewObjectSchema()
	httpMeta := openapi3.NewObjectSchema()
	httpMeta.Properties = openapi3.Schemas{
		"cors_enabled":       schemaRef(openapi3.NewBoolSchema()),
		"rate_limit_enabled": schemaRef(openapi3.NewBoolSchema()),
		"request_timeout":    schemaRef(openapi3.NewStringSchema()),
		"body_limit":         schemaRef(&openapi3.Schema{Type: &openapi3.Types{openapi3.TypeString}, Description: `"default" or max request body bytes as decimal string`}),
	}
	i18nOff := openapi3.NewObjectSchema()
	i18nOff.Properties = openapi3.Schemas{
		"enabled": schemaRef(openapi3.NewBoolSchema()),
	}
	supportedArr := openapi3.NewArraySchema()
	supportedArr.Items = &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}
	i18nOn := openapi3.NewObjectSchema()
	i18nOn.Properties = openapi3.Schemas{
		"enabled":           schemaRef(openapi3.NewBoolSchema()),
		"default_locale":    schemaRef(openapi3.NewStringSchema()),
		"supported_locales": schemaRef(supportedArr),
		"active_locale":     schemaRef(openapi3.NewStringSchema()),
	}
	s.Properties = openapi3.Schemas{
		"name":                      schemaRef(openapi3.NewStringSchema()),
		"env":                       schemaRef(openapi3.NewStringSchema()),
		"version":                   schemaRef(openapi3.NewStringSchema()),
		"error_includes_request_id": schemaRef(openapi3.NewBoolSchema()),
		"endpoints":                 schemaRef(endpoints),
		"http":                      schemaRef(httpMeta),
		"i18n":                      schemaRef(openapi3.NewAnyOfSchema(i18nOff, i18nOn)),
	}
	return openapi3.NewResponses(
		openapi3.WithStatus(200, &openapi3.ResponseRef{
			Value: openapi3.NewResponse().
				WithDescription("Application index with probe and API path hints").
				WithJSONSchema(s),
		}),
	)
}

func responsesStatusJSON() *openapi3.Responses {
	checks := openapi3.NewObjectSchema().WithAnyAdditionalProperties()
	s := openapi3.NewObjectSchema()
	s.Properties = openapi3.Schemas{
		"app":       schemaRef(openapi3.NewStringSchema()),
		"env":       schemaRef(openapi3.NewStringSchema()),
		"version":   schemaRef(openapi3.NewStringSchema()),
		"ready":     schemaRef(openapi3.NewBoolSchema()),
		"checks":    schemaRef(checks),
		"timestamp": schemaRef(openapi3.NewStringSchema().WithFormat("date-time")),
	}
	return openapi3.NewResponses(
		openapi3.WithStatus(200, &openapi3.ResponseRef{
			Value: openapi3.NewResponse().
				WithDescription("Aggregated dependency checks and host metadata").
				WithJSONSchema(s),
		}),
	)
}

