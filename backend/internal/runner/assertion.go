package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// AssertionFailure records one assertion that did not pass.
type AssertionFailure struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Assert runs all assertions declared on a test case against the actual
// HTTP response. Returns nil if every check passes.
func Assert(tc *model.TestCase, actualStatus int, actualHeaders map[string][]string, actualBody []byte) []AssertionFailure {
	var failures []AssertionFailure

	// 1. Status code
	if tc.ExpectedStatus > 0 && actualStatus != tc.ExpectedStatus {
		failures = append(failures, AssertionFailure{
			Type: "status", Message: fmt.Sprintf("expected %d, got %d", tc.ExpectedStatus, actualStatus),
		})
	}

	// 2. Body match
	hasBodyAssertion := tc.ExpectedBodyMatch != "" && tc.ExpectedBodyMatch != "none"
	if hasBodyAssertion && (tc.ExpectedBodyJSON != nil || (tc.ExpectedBodyMatch == "regex" && tc.ExpectedBodyPattern != "")) {
		f := assertBody(tc, actualBody)
		failures = append(failures, f...)
	}

	// 3. Advanced assertions (assertions_json)
	if tc.AssertionsJSON != nil {
		f := assertAdvanced(tc.AssertionsJSON, actualStatus, actualHeaders, actualBody)
		failures = append(failures, f...)
	}

	return failures
}

func assertBody(tc *model.TestCase, body []byte) []AssertionFailure {
	switch tc.ExpectedBodyMatch {
	case "exact":
		if !jsonEqual(tc.ExpectedBodyJSON, body) {
			return []AssertionFailure{{Type: "body.exact", Message: "body does not match expected JSON"}}
		}
	case "contains":
		if !strings.Contains(string(body), string(tc.ExpectedBodyJSON)) {
			return []AssertionFailure{{Type: "body.contains", Message: "body does not contain expected substring"}}
		}
	case "regex":
		pattern := tc.ExpectedBodyPattern
		if pattern == "" {
			pattern = string(tc.ExpectedBodyJSON)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return []AssertionFailure{{Type: "body.regex", Message: fmt.Sprintf("invalid regex: %v", err)}}
		}
		if !re.Match(body) {
			return []AssertionFailure{{Type: "body.regex", Message: "body does not match pattern"}}
		}
	case "jsonpath":
		return assertJSONPath(tc.ExpectedBodyJSON, body)
	case "schema":
		return assertJSONSchema(tc.ExpectedBodyJSON, body)
	default:
		return []AssertionFailure{{Type: "body.unknown", Message: fmt.Sprintf("unknown match type: %s", tc.ExpectedBodyMatch)}}
	}
	return nil
}

func assertAdvanced(raw []byte, status int, headers map[string][]string, body []byte) []AssertionFailure {
	var rules []map[string]any
	if err := json.Unmarshal(raw, &rules); err != nil {
		return []AssertionFailure{{Type: "assertions.parse", Message: fmt.Sprintf("invalid assertions_json: %v", err)}}
	}
	var failures []AssertionFailure
	for _, rule := range rules {
		typ, _ := rule["type"].(string)
		switch typ {
		case "status":
			expect := toFloat(rule["expect"])
			if int(expect) != status {
				failures = append(failures, AssertionFailure{Type: "status", Message: fmt.Sprintf("expected %d, got %d", int(expect), status)})
			}
		case "header":
			name, _ := rule["name"].(string)
			expect, _ := rule["expect"].(string)
			vals := headers[strings.ToLower(name)]
			found := false
			for _, v := range vals {
				if v == expect {
					found = true
					break
				}
			}
			if !found {
				failures = append(failures, AssertionFailure{Type: "header", Message: fmt.Sprintf("header %s: expected %q, got %v", name, expect, vals)})
			}
		}
	}
	return failures
}

func jsonEqual(a, b []byte) bool {
	var ja, jb any
	if err := json.Unmarshal(a, &ja); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &jb); err != nil {
		return false
	}
	aa, _ := json.Marshal(ja)
	bb, _ := json.Marshal(jb)
	return string(aa) == string(bb)
}

func assertJSONPath(raw []byte, body []byte) []AssertionFailure {
	var spec struct {
		Path   string `json:"path"`
		Expect any    `json:"expect"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return []AssertionFailure{{Type: "body.jsonpath", Message: fmt.Sprintf("invalid jsonpath spec: %v", err)}}
	}
	val, err := jsonPathLookup(body, spec.Path)
	if err != nil {
		return []AssertionFailure{{Type: "body.jsonpath", Message: err.Error()}}
	}
	expectJSON, _ := json.Marshal(spec.Expect)
	gotJSON, _ := json.Marshal(val)
	if string(expectJSON) != string(gotJSON) {
		return []AssertionFailure{{Type: "body.jsonpath", Message: fmt.Sprintf("path %s: expected %s, got %s", spec.Path, expectJSON, gotJSON)}}
	}
	return nil
}

func jsonPathLookup(raw []byte, path string) (any, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	// Simple $.foo.bar[0].baz implementation
	parts := strings.Split(strings.TrimPrefix(path, "$."), ".")
	cur := root
	for _, part := range parts {
		// Check for array index: key[0]
		key := part
		var idx *int
		if i := strings.IndexByte(part, '['); i >= 0 {
			key = part[:i]
			end := strings.IndexByte(part[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("invalid index syntax in %q", part)
			}
			n := 0
			if _, err := fmt.Sscanf(part[i+1:i+end], "%d", &n); err != nil {
				return nil, fmt.Errorf("invalid array index in %q", part)
			}
			idx = &n
		}
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot navigate into non-object at %q", key)
		}
		v, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("key %q not found", key)
		}
		if idx != nil {
			arr, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("%q is not an array", key)
			}
			if *idx >= len(arr) {
				return nil, fmt.Errorf("index %d out of range for %q", *idx, key)
			}
			v = arr[*idx]
		}
		cur = v
	}
	return cur, nil
}

func assertJSONSchema(schemaRaw, body []byte) []AssertionFailure {
	schemaLoader := gojsonschema.NewBytesLoader(schemaRaw)
	docLoader := gojsonschema.NewBytesLoader(body)
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return []AssertionFailure{{Type: "body.schema", Message: fmt.Sprintf("schema validation error: %v", err)}}
	}
	if !result.Valid() {
		var msgs []string
		for _, e := range result.Errors() {
			msgs = append(msgs, e.String())
		}
		return []AssertionFailure{{Type: "body.schema", Message: strings.Join(msgs, "; ")}}
	}
	return nil
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64: return x
	case int: return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	}
	return 0
}
