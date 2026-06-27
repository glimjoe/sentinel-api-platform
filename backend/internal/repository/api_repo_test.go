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

func TestAPIRepo_FindByID_Found(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAPIRepo(gdb)

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `apis` WHERE id = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "name", "method", "path", "created_at", "updated_at"}).
			AddRow("01HA", "01HP", "findByStatus", "GET", "/pet/findByStatus", now, now))

	a, err := r.FindByID(context.Background(), "01HA")
	require.NoError(t, err)
	assert.Equal(t, "GET", a.Method)
	assert.Equal(t, "/pet/findByStatus", a.Path)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIRepo_FindByID_NotFound_MapsToErrNotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAPIRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `apis` WHERE id = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := r.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIRepo_Create_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAPIRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `apis`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := r.Create(context.Background(), &model.API{
		ID: "01HA", ProjectID: "01HP", Name: "x", Method: "GET", Path: "/x",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIRepo_ListByProject_Empty(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAPIRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `apis` WHERE project_id = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "name", "method", "path", "created_at", "updated_at"}))

	as, err := r.ListByProject(context.Background(), "01HP")
	require.NoError(t, err)
	assert.Empty(t, as)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIRepo_Delete_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAPIRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `apis` WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := r.Delete(context.Background(), "01HA")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
