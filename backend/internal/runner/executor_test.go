package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

func TestExecute_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tc := &model.TestCase{
		ID:             "tc1",
		Method:         "GET",
		Path:           "/api/health",
		ExpectedStatus: 200,
	}
	result := Execute(context.Background(), tc, srv.URL)
	assert.Equal(t, "pass", result.Status)
	assert.Equal(t, 200, *result.ActualStatus)
	assert.GreaterOrEqual(t, result.DurationMs, 0)
}

func TestExecute_FailStatusMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	tc := &model.TestCase{
		ID:             "tc2",
		Method:         "GET",
		Path:           "/api/broken",
		ExpectedStatus: 200,
	}
	result := Execute(context.Background(), tc, srv.URL)
	assert.Equal(t, "fail", result.Status)
	assert.Equal(t, 500, *result.ActualStatus)
}

func TestExecute_ErrorConnectionRefused(t *testing.T) {
	tc := &model.TestCase{
		ID:     "tc3",
		Method: "GET",
		Path:   "/api/ghost",
	}
	result := Execute(context.Background(), tc, "http://127.0.0.1:1")
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.ErrorMsg, "execute:")
}

func TestExecute_PostWithBody(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody = make([]byte, r.ContentLength)
		r.Body.Read(receivedBody)
		w.WriteHeader(201)
		w.Write([]byte(`{"created":true}`))
	}))
	defer srv.Close()

	body := []byte(`{"name":"test"}`)
	tc := &model.TestCase{
		ID:             "tc4",
		Method:         "POST",
		Path:           "/api/resource",
		BodyJSON:       body,
		ExpectedStatus: 201,
	}
	result := Execute(context.Background(), tc, srv.URL)
	assert.Equal(t, "pass", result.Status)
	assert.Equal(t, 201, *result.ActualStatus)
	assert.Equal(t, `{"name":"test"}`, string(receivedBody))
}

func TestExecute_Headers(t *testing.T) {
	var receivedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("X-Response", "bar")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	headersJSON, err := json.Marshal(map[string]string{"X-Custom": "foo"})
	require.NoError(t, err)

	tc := &model.TestCase{
		ID:          "tc5",
		Method:      "GET",
		Path:        "/api/headers",
		HeadersJSON: headersJSON,
	}
	result := Execute(context.Background(), tc, srv.URL)
	assert.Equal(t, "pass", result.Status)
	assert.Equal(t, "foo", receivedHeaders.Get("X-Custom"))
	assert.Contains(t, string(result.ActualHeadersJSON), "X-Response")
}

func TestExecute_AssertionsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rate-Limit", "100")
		w.WriteHeader(200)
		w.Write([]byte(`{"data":[1,2,3]}`))
	}))
	defer srv.Close()

	assertionsJSON, err := json.Marshal([]map[string]any{
		{"type": "status", "expect": float64(200)},
	})
	require.NoError(t, err)

	tc := &model.TestCase{
		ID:             "tc6",
		Method:         "GET",
		Path:           "/api/data",
		ExpectedStatus: 200,
		AssertionsJSON: assertionsJSON,
	}
	result := Execute(context.Background(), tc, srv.URL)
	assert.Equal(t, "pass", result.Status)
}

func TestExecute_MethodAndPath(t *testing.T) {
	var receivedMethod, receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(200)
	}))
	defer srv.Close()

	tc := &model.TestCase{
		ID:     "tc7",
		Method: "DELETE",
		Path:   "/api/resource/42",
	}
	result := Execute(context.Background(), tc, srv.URL)
	assert.Equal(t, "pass", result.Status)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/resource/42", receivedPath)
}

func TestCompactJSON(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{"valid json", []byte(`{"a":  1,  "b":2}`)},
		{"invalid json", []byte(`not json`)},
		{"empty array", []byte(`[1, 2 ,3]`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compactJSON(tt.in)
			assert.NotNil(t, got)
		})
	}
}
