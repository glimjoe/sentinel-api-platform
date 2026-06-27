package mock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_EmptyJSON_NoOp(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	require.NoError(t, ex.Apply(ctx, "p", "s", nil, []byte(`{"a":1}`), bag))
	m, err := bag.GetAll(ctx, "p", "s")
	require.NoError(t, err)
	assert.Empty(t, m)
}

func TestExtractor_GjsonPath(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	extractorJSON := []byte(`[{"path":"data.token","as":"login_token","from":"response.body"}]`)
	body := []byte(`{"data":{"token":"abc-123"}}`)

	require.NoError(t, ex.Apply(ctx, "p", "s", extractorJSON, body, bag))
	got, ok, err := bag.Get(ctx, "p", "s", "login_token")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "abc-123", got)
}

func TestExtractor_GjsonNestedArray(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	extractorJSON := []byte(`[{"path":"users.0.id","as":"first_id","from":"response.body"}]`)
	body := []byte(`{"users":[{"id":7},{"id":8}]}`)

	require.NoError(t, ex.Apply(ctx, "p", "s", extractorJSON, body, bag))
	got, _, err := bag.Get(ctx, "p", "s", "first_id")
	require.NoError(t, err)
	assert.Equal(t, "7", got)
}

func TestExtractor_RegexWithCaptureGroup(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	extractorJSON := []byte(`[{"path":"req-([0-9]+)","as":"trace_num","from":"response.body","type":"regex"}]`)
	body := []byte(`prefix req-4242 suffix`)

	require.NoError(t, ex.Apply(ctx, "p", "s", extractorJSON, body, bag))
	got, ok, err := bag.Get(ctx, "p", "s", "trace_num")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "4242", got)
}

func TestExtractor_RegexNoMatch_NoWrite(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	extractorJSON := []byte(`[{"path":"req-([0-9]+)","as":"trace_num","from":"response.body","type":"regex"}]`)
	body := []byte(`no trace here`)

	require.NoError(t, ex.Apply(ctx, "p", "s", extractorJSON, body, bag))
	m, err := bag.GetAll(ctx, "p", "s")
	require.NoError(t, err)
	assert.Empty(t, m, "no match must not write an empty value to the bag")
}

func TestExtractor_MissingGjsonPath_NoWrite(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	extractorJSON := []byte(`[{"path":"data.does_not_exist","as":"x","from":"response.body"}]`)
	body := []byte(`{"data":{"other":1}}`)

	require.NoError(t, ex.Apply(ctx, "p", "s", extractorJSON, body, bag))
	m, err := bag.GetAll(ctx, "p", "s")
	require.NoError(t, err)
	assert.Empty(t, m)
}

func TestExtractor_LastWriteWins(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	// Two rules write to the same `as`. The second one wins.
	extractorJSON := []byte(`[
	  {"path":"a","as":"k","from":"response.body"},
	  {"path":"b","as":"k","from":"response.body"}
	]`)
	body := []byte(`{"a":"first","b":"second"}`)

	require.NoError(t, ex.Apply(ctx, "p", "s", extractorJSON, body, bag))
	got, _, err := bag.Get(ctx, "p", "s", "k")
	require.NoError(t, err)
	assert.Equal(t, "second", got, "last extractor wins per ADR-0007")
}

func TestExtractor_MultipleRules(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	extractorJSON := []byte(`[
	  {"path":"token","as":"login_token","from":"response.body"},
	  {"path":"user_id","as":"uid","from":"response.body"}
	]`)
	body := []byte(`{"token":"tk-1","user_id":99}`)

	require.NoError(t, ex.Apply(ctx, "p", "s", extractorJSON, body, bag))
	m, err := bag.GetAll(ctx, "p", "s")
	require.NoError(t, err)
	assert.Equal(t, "tk-1", m["login_token"])
	assert.Equal(t, "99", m["uid"])
}

func TestExtractor_RejectsInvalidJSON(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	// Not an array.
	err := ex.Apply(ctx, "p", "s", []byte(`{"path":"a","as":"k"}`), []byte(`{}`), bag)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an array")
}

func TestExtractor_RejectsMissingAS(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	err := ex.Apply(ctx, "p", "s", []byte(`[{"path":"a"}]`), []byte(`{"a":1}`), bag)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "`as` is required")
}

func TestExtractor_RejectsUnknownFrom(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	err := ex.Apply(ctx, "p", "s",
		[]byte(`[{"path":"a","as":"k","from":"response.cookie"}]`),
		[]byte(`{}`), bag)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response.body")
}

func TestExtractor_RejectsBadRegex(t *testing.T) {
	ex := NewExtractor()
	bag, _ := newTestBag(t)
	ctx := context.Background()

	err := ex.Apply(ctx, "p", "s",
		[]byte(`[{"path":"(","as":"k","from":"response.body","type":"regex"}]`),
		[]byte(`{}`), bag)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad regex")
}
