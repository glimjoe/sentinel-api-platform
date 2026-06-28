package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// Execute runs one test case against the target base URL and returns a result.
func Execute(ctx context.Context, tc *model.TestCase, baseURL string) *model.TestResult {
	start := time.Now()
	result := &model.TestResult{
		CaseID:  tc.ID,
		Attempt: 1,
	}

	// Build target URL
	target := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(tc.Path, "/")

	// Build request body
	var bodyReader io.Reader
	if tc.BodyJSON != nil {
		bodyReader = strings.NewReader(string(tc.BodyJSON))
	}

	req, err := http.NewRequestWithContext(ctx, tc.Method, target, bodyReader)
	if err != nil {
		result.Status = "error"
		result.ErrorMsg = fmt.Sprintf("build request: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result
	}

	// Set headers
	if tc.HeadersJSON != nil {
		var headers map[string]string
		if err := json.Unmarshal(tc.HeadersJSON, &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Status = "error"
		result.ErrorMsg = fmt.Sprintf("execute: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	result.ActualStatus = &resp.StatusCode
	result.DurationMs = int(time.Since(start).Milliseconds())

	// Marshal headers
	headerMap := make(map[string][]string)
	for k, v := range resp.Header {
		headerMap[k] = v
	}
	headerJSON, _ := json.Marshal(headerMap)
	result.ActualHeadersJSON = headerJSON

	// Marshal body
	if len(body) > 0 {
		compactBody := compactJSON(body)
		result.ActualBodyJSON = compactBody
	}

	// Run assertions
	failures := Assert(tc, resp.StatusCode, headerMap, body)
	if len(failures) > 0 {
		result.Status = "fail"
		failJSON, _ := json.Marshal(failures)
		result.AssertionFailuresJSON = failJSON
	} else {
		result.Status = "pass"
	}

	return result
}

func compactJSON(raw []byte) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	b, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return b
}
