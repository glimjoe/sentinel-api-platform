package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestTestRunRepo_Create(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestRunRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `test_runs`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := r.Create(context.Background(), &model.TestRun{ID: "01HXXX", ProjectID: "01HYYY"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestRunRepo_FindByID(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestRunRepo(gdb)
	mock.ExpectQuery("SELECT \\* FROM `test_runs`").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("01HXXX", "smoke"))
	tr, err := r.FindByID(context.Background(), "01HXXX")
	require.NoError(t, err)
	assert.Equal(t, "smoke", tr.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestRunRepo_ListByProject(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestRunRepo(gdb)
	mock.ExpectQuery("SELECT \\* FROM `test_runs`").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("a", "r1"))
	list, err := r.ListByProject(context.Background(), "01HYYY")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestRunRepo_Update(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestRunRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `test_runs`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	err := r.Update(context.Background(), "01HXXX", map[string]any{"status": "running"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
