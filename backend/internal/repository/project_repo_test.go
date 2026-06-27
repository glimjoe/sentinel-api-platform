package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

func TestProjectRepo_FindBySlug_Found(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectRepo(gdb)

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `projects` WHERE slug = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "owner_id", "created_at", "updated_at"}).
			AddRow("01HXXXXXXXXXXXXXXXXXXXXXXXX", "Petstore", "petstore",
				"01HYYYYYYYYYYYYYYYYYYYYYYYY", now, now))

	p, err := r.FindBySlug(context.Background(), "petstore")
	require.NoError(t, err)
	assert.Equal(t, "petstore", p.Slug)
	assert.Equal(t, "Petstore", p.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectRepo_FindBySlug_NotFound_MapsToErrNotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `projects` WHERE slug = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // empty

	_, err := r.FindBySlug(context.Background(), "ghost")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound),
		"engine looks for this exact match to return 404: got %v", err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectRepo_FindByID_NotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `projects` WHERE id = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := r.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectRepo_Create_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `projects`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := r.Create(context.Background(), &model.Project{
		ID: "01HXXXXXXXXXXXXXXXXXXXXXXXX", Name: "Petstore", Slug: "petstore",
		OwnerID: "01HYYYYYYYYYYYYYYYYYYYYYYYY",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectRepo_ListByOwner_ReturnsRows(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectRepo(gdb)

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `projects` WHERE owner_id = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "owner_id", "created_at", "updated_at"}).
			AddRow("01HA", "A", "a", "owner1", now, now).
			AddRow("01HB", "B", "b", "owner1", now, now))

	ps, err := r.ListByOwner(context.Background(), "owner1")
	require.NoError(t, err)
	assert.Len(t, ps, 2)
	assert.Equal(t, "a", ps[0].Slug)
	assert.Equal(t, "b", ps[1].Slug)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProjectRepo_AddMember_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewProjectRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `project_members`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := r.AddMember(context.Background(), &model.ProjectMember{
		ProjectID: "01HP", UserID: "01HU", Role: model.ProjectRoleAdmin,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
