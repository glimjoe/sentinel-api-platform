package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestMockHitRepo_Create_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockHitRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `mock_hits`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := r.Create(context.Background(), &model.MockHit{
		ID: "01HMH", MockRuleID: "01HMR",
		RequestMethod: "GET", RequestPath: "/x",
		ResponseStatus: 200, DurationMs: 12,
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
