package repository

import (
	"context"
	"fmt"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

func TestMockRuleRepo_FindByID_Found(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `mock_rules` WHERE id = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id", "api_id", "name", "match_json", "response_status", "priority", "enabled", "created_at", "updated_at"}).
			AddRow("01HMR", "01HA", "r1", []byte(`{"query":{"a":"1"}}`), 200, 100, true, now, now))

	m, err := r.FindByID(context.Background(), "01HMR")
	require.NoError(t, err)
	assert.Equal(t, "r1", m.Name)
	assert.Equal(t, 100, m.Priority)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_FindByID_NotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectQuery("SELECT \\* FROM `mock_rules` WHERE id = \\?").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := r.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_Create_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `mock_rules`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := r.Create(context.Background(), &model.MockRule{
		ID: "01HMR", APIID: "01HA", Name: "r1", MatchJSON: []byte(`{}`),
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_ListByAPI_OrderedByPriorityThenID(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `mock_rules` WHERE api_id = \\? AND enabled = 1 ORDER BY priority ASC, id ASC").
		WillReturnRows(sqlmock.NewRows([]string{"id", "api_id", "name", "match_json", "response_status", "priority", "enabled", "created_at", "updated_at"}).
			AddRow("01HMR-1", "01HA", "lo", []byte(`{}`), 200, 100, true, now, now).
			AddRow("01HMR-2", "01HA", "hi", []byte(`{}`), 200, 200, true, now, now))

	rules, err := r.ListByAPI(context.Background(), "01HA")
	require.NoError(t, err)
	assert.Len(t, rules, 2)
	assert.Equal(t, "lo", rules[0].Name)
	assert.Equal(t, "hi", rules[1].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_ListByProject_JoinsApis(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	now := time.Now()
	// The engine's hot path: joined query across apis + mock_rules, ordered.
	// We just verify the SQL shape is correct — full result-set assertions
	// would be brittle. gorm generates the JOIN with backticks.
	mock.ExpectQuery("SELECT .* FROM mock_rules AS r JOIN apis AS a ON a\\.id = r\\.api_id WHERE a\\.project_id = \\? AND r\\.enabled = 1 ORDER BY r\\.priority ASC, r\\.id ASC").
		WillReturnRows(sqlmock.NewRows([]string{"id", "api_id", "name", "match_json", "response_status", "priority", "enabled", "created_at", "updated_at"}).
			AddRow("01HMR-1", "01HA", "rule-a", []byte(`{}`), 200, 100, true, now, now))

	rules, err := r.ListByProject(context.Background(), "01HP")
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, "rule-a", rules[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_IncrementHitCount_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `mock_rules` SET .*`hit_count`=hit_count \\+ 1.* WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := r.IncrementHitCount(context.Background(), "01HMR")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_IncrementHitCount_NotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `mock_rules` SET .*`hit_count`=hit_count \\+ 1.* WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 0)) // no rows affected
	mock.ExpectCommit()

	err := r.IncrementHitCount(context.Background(), "ghost")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_Update_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `mock_rules`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := r.Update(context.Background(), "01HMR", map[string]any{"name": "updated"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_Update_NotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `mock_rules`").
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected
	mock.ExpectCommit()

	err := r.Update(context.Background(), "ghost", map[string]any{"name": "x"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_Update_Error(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `mock_rules`").
		WillReturnError(fmt.Errorf("db down"))
	mock.ExpectRollback()

	err := r.Update(context.Background(), "01HMR", map[string]any{"name": "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update mock_rule")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_Delete_Success(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `mock_rules` WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := r.Delete(context.Background(), "01HMR")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_Delete_NotFound(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `mock_rules` WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := r.Delete(context.Background(), "ghost")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMockRuleRepo_Delete_Error(t *testing.T) {
	gdb, mock := newMockGorm(t)
	r := NewMockRuleRepo(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `mock_rules` WHERE id = \\?").
		WillReturnError(fmt.Errorf("db down"))
	mock.ExpectRollback()

	err := r.Delete(context.Background(), "01HMR")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete mock_rule")
	require.NoError(t, mock.ExpectationsWereMet())
}
