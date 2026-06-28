package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestTestResultRepo_Create(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestResultRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `test_results`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := r.Create(context.Background(), &model.TestResult{ID: "01HXXX", RunID: "01HYYY"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTestResultRepo_ListByRun(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewTestResultRepo(gdb)
	mock.ExpectQuery("SELECT \\* FROM `test_results`").WillReturnRows(sqlmock.NewRows([]string{"id", "run_id"}).AddRow("a", "r1").AddRow("b", "r1"))
	list, err := r.ListByRun(context.Background(), "r1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}
