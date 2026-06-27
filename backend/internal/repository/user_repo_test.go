package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

func newMockGorm(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	// Regexp matcher: GORM generates fully-qualified column lists we don't want
	// to hand-mirror in every Expect. We just match the SQL "shape".
	conn, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	require.NoError(t, err)
	return gdb, mock
}

func TestUserRepo_FindByEmail_Found(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewUserRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE email = \\? ORDER BY `users`.`id` LIMIT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow("01HXXXXXXXXXXXXXXXXXXXXXXXX", "alice@example.com"))

	u, err := r.FindByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", u.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_FindByEmail_NotFound_MapsToErrNotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewUserRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE email = \\? ORDER BY `users`.`id` LIMIT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"})) // empty result

	_, err := r.FindByEmail(context.Background(), "ghost@example.com")
	require.Error(t, err)
	// Service layer relies on errors.Is(err, errs.ErrNotFound) — verify the wrap.
	assert.True(t, errors.Is(err, errs.ErrNotFound),
		"expected errors.Is(err, errs.ErrNotFound), got %v", err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_FindByID_NotFound_MapsToErrNotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewUserRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE id = \\? ORDER BY `users`.`id` LIMIT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}))

	_, err := r.FindByID(context.Background(), "01HXXXXXXXXXXXXXXXXXXXXXXXX")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound),
		"expected errors.Is(err, errs.ErrNotFound), got %v", err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_Create_DuplicateEmail_WrapsGormError(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewUserRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnError(errors.New("Error 1062: Duplicate entry 'alice@example.com' for key 'uq_users_email'"))
	mock.ExpectRollback()

	err := r.Create(context.Background(), &model.User{ID: "01HXXX", Email: "alice@example.com"})
	require.Error(t, err)
	// The wrapped error must still allow isDuplicateEmail (in auth_service) to
	// errors.As the MySQLError. Wrapping with %w preserves the chain.
	assert.Contains(t, err.Error(), "create user")
	require.NoError(t, mock.ExpectationsWereMet())
}