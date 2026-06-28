// Package contract — OpenAPI change detection.
package contract

import (
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// ChangeType classifies the severity of a detected API change.
type ChangeType string

const (
	ChangeAdded    ChangeType = "added"
	ChangeRemoved  ChangeType = "removed"
	ChangeModified ChangeType = "modified"
)

// Change records one difference between two API specs.
type Change struct {
	Type     ChangeType `json:"type"`
	Path     string     `json:"path"`
	Method   string     `json:"method"`
	Field    string     `json:"field"`
	OldValue string     `json:"old_value,omitempty"`
	NewValue string     `json:"new_value,omitempty"`
	Breaking bool       `json:"breaking"`
}

// Diff compares two sets of extracted APIs and returns changes.
func Diff(old, newer []*model.API) []Change {
	oldMap := indexByPath(old)
	newMap := indexByPath(newer)
	var changes []Change

	for key, oldAPI := range oldMap {
		newAPI, exists := newMap[key]
		if !exists {
			changes = append(changes, Change{
				Type: ChangeRemoved, Path: oldAPI.Path, Method: oldAPI.Method,
				Breaking: true,
			})
			continue
		}
		changes = append(changes, diffOne(oldAPI, newAPI)...)
	}
	for key, newAPI := range newMap {
		if _, exists := oldMap[key]; !exists {
			changes = append(changes, Change{
				Type: ChangeAdded, Path: newAPI.Path, Method: newAPI.Method,
			})
		}
	}
	return changes
}

// IsBreaking returns true if any change is breaking.
func IsBreaking(changes []Change) bool {
	for _, c := range changes {
		if c.Breaking {
			return true
		}
	}
	return false
}

// Breaking returns only breaking changes.
func Breaking(changes []Change) []Change {
	var out []Change
	for _, c := range changes {
		if c.Breaking {
			out = append(out, c)
		}
	}
	return out
}

// DiffSpecs compares two raw OpenAPI documents.
func DiffSpecs(oldSpec, newSpec []byte) ([]Change, error) {
	oldDoc, err := Load(oldSpec)
	if err != nil {
		return nil, err
	}
	newDoc, err := Load(newSpec)
	if err != nil {
		return nil, err
	}
	oldAPIs, err := ExtractAPIs(oldDoc, "")
	if err != nil {
		return nil, err
	}
	newAPIs, err := ExtractAPIs(newDoc, "")
	if err != nil {
		return nil, err
	}
	return Diff(oldAPIs, newAPIs), nil
}

func indexByPath(apis []*model.API) map[string]*model.API {
	m := make(map[string]*model.API, len(apis))
	for _, a := range apis {
		m[a.Method+" "+a.Path] = a
	}
	return m
}

func diffOne(oldAPI, newAPI *model.API) []Change {
	var changes []Change
	if oldAPI.Method != newAPI.Method {
		changes = append(changes, Change{
			Type: ChangeModified, Path: oldAPI.Path, Method: oldAPI.Method,
			Field: "method", OldValue: oldAPI.Method, NewValue: newAPI.Method,
			Breaking: true,
		})
	}
	oldDoc, _ := openapi3.NewLoader().LoadFromData(oldAPI.SpecJSON)
	newDoc, _ := openapi3.NewLoader().LoadFromData(newAPI.SpecJSON)
	if oldDoc != nil && newDoc != nil {
		oldPI := oldDoc.Paths.Find(oldAPI.Path)
		newPI := newDoc.Paths.Find(newAPI.Path)
		if oldPI != nil && newPI != nil {
			oldOp := oldPI.GetOperation(oldAPI.Method)
			newOp := newPI.GetOperation(newAPI.Method)
			if oldOp != nil && newOp != nil {
				oldReq := collectRequired(oldOp)
				newReq := collectRequired(newOp)
				for _, f := range oldReq {
					if !contains(newReq, f) {
						changes = append(changes, Change{
							Type: ChangeModified, Path: oldAPI.Path, Method: oldAPI.Method,
							Field: "required." + f, OldValue: "required", NewValue: "removed",
							Breaking: true,
						})
					}
				}
			}
		}
	}
	if len(changes) == 0 {
		changes = append(changes, Change{Type: ChangeModified, Path: oldAPI.Path, Method: oldAPI.Method})
	}
	return changes
}

func collectRequired(op *openapi3.Operation) []string {
	var req []string
	for _, p := range op.Parameters {
		if p.Value != nil && p.Value.Required {
			req = append(req, p.Value.Name)
		}
	}
	if op.RequestBody != nil && op.RequestBody.Value != nil && op.RequestBody.Value.Required {
		req = append(req, "requestBody")
	}
	return req
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
