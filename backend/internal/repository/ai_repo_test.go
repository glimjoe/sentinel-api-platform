package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestAiRepo_RecordUsage(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAiRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `ai_usage`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := r.RecordUsage(context.Background(), &model.AiUsage{Model: "mock", Function: "attribution"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAiRepo_GetDailyUsage(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAiRepo(gdb)
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(cost_usd\\), 0\\) FROM `ai_usage`").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(0.05))
	total, err := r.GetDailyUsage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0.05, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAiRepo_GetMonthlyUsage(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAiRepo(gdb)
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(cost_usd\\), 0\\) FROM `ai_usage`").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1.5))
	total, err := r.GetMonthlyUsage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1.5, total)
	require.NoError(t, mock.ExpectationsWereMet())
}
