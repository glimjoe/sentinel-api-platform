//go:build integration

package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/repository"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

// TestIntegration_RBAC: verify that role checks work correctly with real DB.
// Creates a project, adds members with different roles, checks RoleFor.
func TestIntegration_RBAC(t *testing.T) {
	if testDB == nil {
		t.Skip("MySQL not available")
	}
	projID := "01JRBACINTEG00000000001"
	userID1 := "01JRBACUSER00000000001"
	userID2 := "01JRBACUSER00000000002"
	userID3 := "01JRBACUSER00000000003"

	// Cleanup first
	testDB.Exec("DELETE FROM project_members WHERE project_id = ?", projID)
	testDB.Exec("DELETE FROM projects WHERE id = ?", projID)

	ctx := context.Background()

	projRepo := repository.NewProjectRepo(testDB)
	memberRepo := repository.NewProjectMemberRepo(testDB)
	svc := service.NewProjectService(projRepo, memberRepo)

	// Create project
	p, err := svc.Create(ctx, userID1, "RBAC Test", "rbac-test", "")
	require.NoError(t, err)
	require.NotNil(t, p)

	// Owner is admin
	role, err := svc.RoleFor(ctx, p.ID, userID1)
	require.NoError(t, err)
	assert.Equal(t, model.ProjectRoleAdmin, role)

	// Non-member returns empty
	role, err = svc.RoleFor(ctx, p.ID, userID2)
	require.NoError(t, err)
	assert.Equal(t, "", role)

	// Add engineer
	err = svc.AddMember(ctx, userID1, p.ID, userID2, model.ProjectRoleEngineer)
	require.NoError(t, err)
	role, err = svc.RoleFor(ctx, p.ID, userID2)
	require.NoError(t, err)
	assert.Equal(t, model.ProjectRoleEngineer, role)

	// Add viewer
	err = svc.AddMember(ctx, userID1, p.ID, userID3, model.ProjectRoleViewer)
	require.NoError(t, err)
	role, err = svc.RoleFor(ctx, p.ID, userID3)
	require.NoError(t, err)
	assert.Equal(t, model.ProjectRoleViewer, role)

	// Viewer cannot add members
	err = svc.AddMember(ctx, userID3, p.ID, "someone-else", model.ProjectRoleEngineer)
	require.Error(t, err)
	// should be forbidden - check error
	assert.Error(t, err)

	// ListMembers
	members, err := svc.ListMembers(ctx, p.ID)
	require.NoError(t, err)
	assert.Len(t, members, 3) // owner (auto-admin) + engineer + viewer

	// Invalid role rejected
	err = svc.AddMember(ctx, userID1, p.ID, "new-user", "superadmin")
	require.Error(t, err)

	// Cleanup
	testDB.Exec("DELETE FROM project_members WHERE project_id = ?", p.ID)
	testDB.Exec("DELETE FROM projects WHERE id = ?", p.ID)
}
