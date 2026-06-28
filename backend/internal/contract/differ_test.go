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
