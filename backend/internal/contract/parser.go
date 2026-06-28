package contract

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// httpMethods lists the HTTP methods we care about in priority order.
var httpMethods = []struct {
	name string
	get  func(*openapi3.PathItem) *openapi3.Operation
}{
	{"GET", func(p *openapi3.PathItem) *openapi3.Operation { return p.Get }},
	{"POST", func(p *openapi3.PathItem) *openapi3.Operation { return p.Post }},
	{"PUT", func(p *openapi3.PathItem) *openapi3.Operation { return p.Put }},
	{"PATCH", func(p *openapi3.PathItem) *openapi3.Operation { return p.Patch }},
	{"DELETE", func(p *openapi3.PathItem) *openapi3.Operation { return p.Delete }},
	{"HEAD", func(p *openapi3.PathItem) *openapi3.Operation { return p.Head }},
	{"OPTIONS", func(p *openapi3.PathItem) *openapi3.Operation { return p.Options }},
}

// ExtractAPIs walks an OpenAPI 3.x document and returns one *model.API per
// (path, method) pair. Each returned API carries its operation_id, tags,
// request/response schemas serialised as JSON, and the full path-level
// openapi3.PathItem stored in spec_json.
func ExtractAPIs(doc *openapi3.T, projectID string) ([]*model.API, error) {
	result := make([]*model.API, 0)

	if doc.Paths == nil {
		return result, nil
	}

	paths := doc.Paths.Map()
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, path := range keys {
		item := paths[path]
		if item == nil {
			continue
		}
		for _, m := range httpMethods {
			op := m.get(item)
			if op == nil {
				continue
			}
			a := buildAPI(projectID, path, m.name, op, item)
			result = append(result, a)
		}
	}
	return result, nil
}

func buildAPI(projectID, path, method string, op *openapi3.Operation, item *openapi3.PathItem) *model.API {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = fmt.Sprintf("%s %s", strings.ToUpper(method), path)
	}

	a := &model.API{
		ID:          id.New(),
		ProjectID:   projectID,
		Name:        name,
		Method:      method,
		Path:        path,
		OperationID: op.OperationID,
		Source:      "openapi",
	}

	if len(op.Tags) > 0 {
		tagsJSON, _ := json.Marshal(op.Tags)
		a.TagsJSON = tagsJSON
	}

	if op.RequestBody != nil && op.RequestBody.Value != nil {
		b, _ := json.Marshal(op.RequestBody.Value)
		a.RequestSchemaJSON = b
	}

	if op.Responses != nil {
		// Favour 200/201/default as the canonical response schema.
		var pick *openapi3.Response
		for _, status := range []string{"200", "201", "default"} {
			if r := op.Responses.Value(status); r != nil && r.Value != nil {
				pick = r.Value
				break
			}
		}
		if pick != nil {
			b, _ := json.Marshal(pick)
			a.ResponseSchemaJSON = b
		}
	}

	specJSON, _ := json.Marshal(item)
	a.SpecJSON = specJSON

	return a
}
