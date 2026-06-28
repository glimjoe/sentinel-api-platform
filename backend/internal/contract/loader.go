// Package contract — OpenAPI 3.x loader and parser.
//
// Phase 2 M2: enables POST /api/v1/projects/:pid/apis/import-openapi.
// Loader reads raw spec bytes into a validated openapi3.T; parser
// extracts the path+method inventory as []*model.API.
package contract

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// Load parses and validates a raw OpenAPI 3.x JSON or YAML document.
// Returns a bad-request error when the document is invalid so the caller
// can map it to HTTP 400.
func Load(data []byte) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrBadRequest, err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrBadRequest, err)
	}
	return doc, nil
}
