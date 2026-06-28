package contract

import (
	"testing"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestDiff_Added(t *testing.T) {
	old := []*model.API{}
	new := []*model.API{{Method: "GET", Path: "/pets"}}
	changes := Diff(old, new)
	if len(changes) != 1 || changes[0].Type != ChangeAdded {
		t.Errorf("expected 1 added change, got %v", changes)
	}
}

func TestDiff_Removed(t *testing.T) {
	old := []*model.API{{Method: "GET", Path: "/pets"}}
	new := []*model.API{}
	changes := Diff(old, new)
	if len(changes) != 1 || changes[0].Type != ChangeRemoved {
		t.Errorf("expected 1 removed change, got %v", changes)
	}
	if !changes[0].Breaking {
		t.Error("removal should be breaking")
	}
}

func TestDiff_Modified(t *testing.T) {
	old := []*model.API{{Method: "GET", Path: "/pets", Name: "old"}}
	new := []*model.API{{Method: "GET", Path: "/pets", Name: "new"}}
	changes := Diff(old, new)
	if len(changes) != 1 || changes[0].Type != ChangeModified {
		t.Errorf("expected 1 modified change, got %v", changes)
	}
}

func TestIsBreaking(t *testing.T) {
	changes := []Change{{Breaking: false}, {Breaking: true}}
	if !IsBreaking(changes) {
		t.Error("expected IsBreaking=true when any change is breaking")
	}
	if IsBreaking([]Change{{Breaking: false}}) {
		t.Error("expected IsBreaking=false when no breaking changes")
	}
}

func TestBreaking_Filter(t *testing.T) {
	changes := []Change{{Breaking: false}, {Breaking: true, Path: "/pets"}}
	b := Breaking(changes)
	if len(b) != 1 || b[0].Path != "/pets" {
		t.Errorf("expected filtered breaking changes, got %v", b)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		slice  []string
		s      string
		expect bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"notFound", []string{"a", "b"}, "c", false},
		{"empty", []string{}, "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.slice, tt.s); got != tt.expect {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.expect)
			}
		})
	}
}

func TestCollectRequired(t *testing.T) {
	doc, _ := Load([]byte(`{
		"openapi": "3.0.3",
		"info": { "title": "T", "version": "1.0.0" },
		"paths": {
			"/pets": {
				"get": {
					"operationId": "listPets",
					"parameters": [
						{ "name": "limit", "in": "query", "required": true, "schema": { "type": "integer" } },
						{ "name": "offset", "in": "query", "required": false, "schema": { "type": "integer" } }
					],
					"requestBody": {
						"required": true,
						"content": { "application/json": { "schema": { "type": "object" } } }
					},
					"responses": { "200": { "description": "OK" } }
				}
			}
		}
	}`))
	pi := doc.Paths.Find("/pets")
	if pi == nil {
		t.Fatal("paths not found")
	}
	op := pi.GetOperation("GET")
	if op == nil {
		t.Fatal("operation not found")
	}
	req := collectRequired(op)
	foundLimit := false
	foundBody := false
	for _, r := range req {
		if r == "limit" {
			foundLimit = true
		}
		if r == "requestBody" {
			foundBody = true
		}
	}
	if !foundLimit {
		t.Error("expected 'limit' in required list")
	}
	if !foundBody {
		t.Error("expected 'requestBody' in required list")
	}
}

func TestDiffSpecs_HappyPath(t *testing.T) {
	spec1 := `{
		"openapi": "3.0.3",
		"info": { "title": "V1", "version": "1.0.0" },
		"paths": {
			"/pets": {
				"get": { "operationId": "listPets", "responses": { "200": { "description": "OK" } } }
			}
		}
	}`
	spec2 := `{
		"openapi": "3.0.3",
		"info": { "title": "V2", "version": "2.0.0" },
		"paths": {
			"/pets": {
				"get": { "operationId": "listPets", "responses": { "200": { "description": "OK" } } }
			},
			"/pets/{id}": {
				"parameters": [
					{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
				],
				"get": { "operationId": "getPet", "responses": { "200": { "description": "OK" } } }
			}
		}
	}`
	changes, err := DiffSpecs([]byte(spec1), []byte(spec2))
	if err != nil {
		t.Fatalf("DiffSpecs: %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("expected 2 changes (modified + added), got %v", changes)
	}
}

func TestDiffSpecs_LoadError(t *testing.T) {
	_, err := DiffSpecs([]byte(`not json`), []byte(`{}`))
	if err == nil {
		t.Error("expected error for invalid old spec")
	}
	_, err = DiffSpecs([]byte(`{"openapi":"3.0.3","info":{"title":"T","version":"1.0.0"},"paths":{}}`), []byte(`bad`))
	if err == nil {
		t.Error("expected error for invalid new spec")
	}
}
