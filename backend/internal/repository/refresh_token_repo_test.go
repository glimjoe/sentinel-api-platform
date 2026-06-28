package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

)

func TestRefreshTokenRepo_GenerateToken(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewRefreshTokenRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `refresh_tokens`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	raw, err := r.GenerateToken(context.Background(), "u1", "", "")
	require.NoError(t, err)
	assert.Len(t, raw, 64) // hex-encoded 32-byte token
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepo_Consume(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewRefreshTokenRepo(gdb)
	// Generate first to get the hash
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `refresh_tokens`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	raw, err := r.GenerateToken(context.Background(), "u1", "", "")
	require.NoError(t, err)

	// Consume uses UPDATE WHERE revoked_at IS NULL
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `refresh_tokens` SET `revoked_at`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	// After UPDATE, fetch user_id
	mock.ExpectQuery("SELECT \\* FROM `refresh_tokens`").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow("t1", "u1"))
	uid, err := r.Consume(context.Background(), raw)
	require.NoError(t, err)
	assert.Equal(t, "u1", uid)
}

func TestRefreshTokenRepo_LookupUserID(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewRefreshTokenRepo(gdb)
	mock.ExpectQuery("SELECT \\* FROM `refresh_tokens`").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow("t1", "u1"))
	uid, err := r.LookupUserID(context.Background(), "deadbeef")
	require.NoError(t, err)
	assert.Equal(t, "u1", uid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepo_RevokeAllForUser(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewRefreshTokenRepo(gdb)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `refresh_tokens` SET `revoked_at`").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	err := r.RevokeAllForUser(context.Background(), "u1")
	require.NoError(t, err)
}
