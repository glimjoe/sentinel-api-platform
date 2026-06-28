package ai

import (
	"context"
	"strings"
)

// MockProvider returns deterministic canned responses based on keyword
// matching in the system prompt. Zero cost, zero network — used for dev.
type MockProvider struct{}

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) Complete(_ context.Context, req *ProviderRequest) (*ProviderResponse, error) {
	prompt := strings.ToLower(req.SystemPrompt)
	var content string

	switch {
	case strings.Contains(prompt, "attribution") || strings.Contains(prompt, "failure attribution"):
		content = `{"analysis":"Mock attribution: the test result shows a mismatch between expected and actual response body.","root_cause":"Response body mismatch — the API returned a different JSON structure than expected.","confidence":0.85,"suggested_fix":"Update the expected body to match the current API response, or fix the API if the regression is unintended."}`

	case strings.Contains(prompt, "completion") || strings.Contains(prompt, "generate test case"):
		content = `{"test_cases":[{"name":"Mock: Happy path — valid request","method":"GET","path":"/api/v1/endpoint","expected_status":200,"expected_body_match":"contains","expected_body_pattern":"success"},{"name":"Mock: Not found — missing resource","method":"GET","path":"/api/v1/endpoint/999","expected_status":404},{"name":"Mock: Invalid input — bad request","method":"POST","path":"/api/v1/endpoint","expected_status":400}]}`

	case strings.Contains(prompt, "priorit") || strings.Contains(prompt, "priority"):
		content = `{"priorities":[{"case_id":"mock-case-1","priority":"p0","reasoning":"Mock: critical-path endpoint — failures here cascade to dependent services."},{"case_id":"mock-case-2","priority":"p1","reasoning":"Mock: core functionality validation."},{"case_id":"mock-case-3","priority":"p2","reasoning":"Mock: edge case — lower blast radius."}]}`

	default:
		content = `{"message":"Mock response — no keyword matched."}`
	}

	return &ProviderResponse{
		Content:      content,
		Model:        "mock",
		InputTokens:  100,
		OutputTokens: 50,
	}, nil
}
