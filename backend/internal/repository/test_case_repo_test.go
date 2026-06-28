package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestTestCaseRepo_Create(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestCaseRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `test_cases`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := r.Create(context.Background(), &model.TestCase{ID: "01HXXX", ProjectID: "01HYYY"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestCaseRepo_FindByID_Found(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestCaseRepo(gdb)
	mock.ExpectQuery("SELECT \\* FROM `test_cases`").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("01HXXX", "tc1"))
	tc, err := r.FindByID(context.Background(), "01HXXX")
	require.NoError(t, err)
	assert.Equal(t, "tc1", tc.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestCaseRepo_FindByID_NotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestCaseRepo(gdb)
	mock.ExpectQuery("SELECT \\* FROM `test_cases`").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, err := r.FindByID(context.Background(), "missing")
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestCaseRepo_ListByProject(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestCaseRepo(gdb)
	mock.ExpectQuery("SELECT \\* FROM `test_cases`").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("a", "tc1").AddRow("b", "tc2"))
	list, err := r.ListByProject(context.Background(), "01HYYY")
	require.NoError(t, err)
	assert.Len(t, list, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestCaseRepo_Update(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestCaseRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `test_cases`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	err := r.Update(context.Background(), "01HXXX", map[string]any{"name": "updated"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestCaseRepo_Delete(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestCaseRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `test_cases`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	err := r.Delete(context.Background(), "01HXXX")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
