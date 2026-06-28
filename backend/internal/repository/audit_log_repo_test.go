package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestAuditLogRepo_Insert(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewAuditLogRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `audit_logs`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := r.Insert(context.Background(), &model.AuditLog{UserID: "u1", Action: "create"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
