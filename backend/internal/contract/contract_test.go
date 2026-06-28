package contract

import (
	"encoding/json"
	"testing"
)

const minimalSpec = `{
  "openapi": "3.0.3",
  "info": { "title": "Petstore", "version": "1.0.0" },
  "paths": {
    "/pet": {
      "get": {
        "operationId": "listPets",
        "summary": "List all pets",
        "tags": ["pets"],
        "responses": {
          "200": { "description": "OK" }
        }
      },
      "post": {
        "operationId": "createPet",
        "tags": ["pets"],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": { "type": "object" }
            }
          }
        },
        "responses": {
          "201": { "description": "Created" }
        }
      }
    },
    "/pet/{id}": {
      "parameters": [
        { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
      ],
      "get": {
        "operationId": "getPet",
        "responses": {
          "200": { "description": "OK" }
        }
      },
      "delete": {
        "operationId": "deletePet",
        "responses": {
          "204": { "description": "No Content" }
        }
      }
    }
  }
}`

func TestLoad_ValidSpec(t *testing.T) {
	doc, err := Load([]byte(minimalSpec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if doc.Paths == nil {
		t.Fatal("Paths should not be nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	_, err := Load([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExtractAPIs_HappyPath(t *testing.T) {
	doc, err := Load([]byte(minimalSpec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	apis, err := ExtractAPIs(doc, "proj-1")
	if err != nil {
		t.Fatalf("ExtractAPIs: %v", err)
	}
	if len(apis) != 4 {
		t.Fatalf("got %d APIs, want 4", len(apis))
	}

	first := apis[0]
	if first.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q", first.ProjectID)
	}
	if first.Source != "openapi" {
		t.Errorf("Source = %q", first.Source)
	}
	if first.ID == "" {
		t.Error("ID should be assigned (ULID)")
	}
	if first.SpecJSON == nil {
		t.Error("SpecJSON should not be nil")
	}

	if first.Method == "GET" && first.Path == "/pet" {
		if first.OperationID != "listPets" {
			t.Errorf("OperationID = %q", first.OperationID)
		}
		if first.Name != "List all pets" {
			t.Errorf("Name = %q", first.Name)
		}
		var tags []string
		json.Unmarshal(first.TagsJSON, &tags)
		if len(tags) != 1 || tags[0] != "pets" {
			t.Errorf("tags = %v", tags)
		}
	}
}

func TestExtractAPIs_EmptyPaths(t *testing.T) {
	emptyDoc := `{
    "openapi": "3.0.3",
    "info": { "title": "Empty", "version": "1.0.0" },
    "paths": {}
  }`
	doc, err := Load([]byte(emptyDoc))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	apis, err := ExtractAPIs(doc, "proj-1")
	if err != nil {
		t.Fatalf("ExtractAPIs: %v", err)
	}
	if apis == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(apis) != 0 {
		t.Errorf("len = %d, want 0", len(apis))
	}
}
