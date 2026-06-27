package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestProjectMemberRepo_Add(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectMemberRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `project_members`").
		WithArgs("p1", "u1", "engineer", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := r.Add(context.Background(), &model.ProjectMember{
		ProjectID: "p1", UserID: "u1", Role: "engineer",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectMemberRepo_Find_Found(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectMemberRepo(gdb)

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `project_members` WHERE project_id = \\? AND user_id = \\? ORDER BY").
		WithArgs("p1", "u1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "user_id", "role", "created_at"}).
			AddRow("p1", "u1", "admin", now))

	m, err := r.Find(context.Background(), "p1", "u1")
	require.NoError(t, err)
	assert.Equal(t, "p1", m.ProjectID)
	assert.Equal(t, "u1", m.UserID)
	assert.Equal(t, "admin", m.Role)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectMemberRepo_ListByProject(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectMemberRepo(gdb)

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `project_members` WHERE project_id = \\?").
		WithArgs("p1").
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "user_id", "role", "created_at"}).
			AddRow("p1", "u1", "admin", now).
			AddRow("p1", "u2", "engineer", now))

	list, err := r.ListByProject(context.Background(), "p1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "u1", list[0].UserID)
	assert.Equal(t, "engineer", list[1].Role)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectMemberRepo_Remove(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectMemberRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `project_members` WHERE project_id = \\? AND user_id = \\?").
		WithArgs("p1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := r.Remove(context.Background(), "p1", "u1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
