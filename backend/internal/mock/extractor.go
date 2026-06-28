// Package mock — variable extractor (Phase 2 M1).
//
// The extractor reads `mock_rules.extractor_json` and writes named values into
// the per-session VarBag so subsequent requests can `{{var}}`-substitute them
// (ADR-0007). Per ADR-0007 §"Conflict rule", last-write-wins on the same `as`
// field within one request — we iterate the rules in declared order and the
// last writer to a given name wins.
//
// The path syntax is gjson dot/bracket notation (e.g. `data.token`,
// `users.0.id`, `errors.#.message`). Regex is supported by setting
// `type` to "regex" and providing a regex in `path`.
//
// Example extractor_json:
//
//	[
//	  {"path": "data.token",    "as": "login_token", "from": "response.body"},
//	  {"path": "refresh_token", "as": "refresh",     "from": "response.body"},
//	  {"path": "^req-[0-9]+$",  "as": "trace_id",    "from": "response.body", "type": "regex"}
//	]
package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/tidwall/gjson"
)

// extractRule is the per-instruction shape. `Path` is the gjson path (or
// regex pattern when Type is "regex"). `As` is the destination variable name.
// `From` is currently always "response.body" — M2 may add "response.header".
type extractRule struct {
	Path string `json:"path"`
	As   string `json:"as"`
	From string `json:"from"`
	Type string `json:"type,omitempty"` // "regex" if path is a regex; empty for gjson
}

// Extractor applies extractor_json to a response body and writes the named
// values into the per-session VarBag. Stateless.
type Extractor struct{}

// NewExtractor constructs an Extractor.
func NewExtractor() *Extractor { return &Extractor{} }

// Apply reads extractorJSON and writes each extracted value to bag. Returns
// nil on success (including the "no extractor" case). The engine calls this
// AFTER rule matching but BEFORE varbag.Substitute on the response body, so
// the same request's response can both feed and consume the bag.
//
// The signature takes just the bytes the extractor needs so unit tests don't
// have to construct a full *model.MockRule.
func (e *Extractor) Apply(ctx context.Context, slug, session string, extractorJSON, responseBody []byte, bag VarBag) error {
	if len(extractorJSON) == 0 {
		return nil
	}
	var rules []extractRule
	if err := json.Unmarshal(extractorJSON, &rules); err != nil {
		return fmt.Errorf("extractor_json: not an array: %w", err)
	}
	for i, r := range rules {
		if r.As == "" {
			return fmt.Errorf("extractor_json[%d]: `as` is required", i)
		}
		if r.Path == "" {
			return fmt.Errorf("extractor_json[%d]: `path` is required", i)
		}
		if r.From != "" && r.From != "response.body" {
			return fmt.Errorf("extractor_json[%d]: `from` must be \"response.body\" (got %q)", i, r.From)
		}
		var value string
		if r.Type == "regex" {
			re, err := regexp.Compile(r.Path)
			if err != nil {
				return fmt.Errorf("extractor_json[%d]: bad regex: %w", i, err)
			}
			m := re.FindStringSubmatch(string(responseBody))
			switch len(m) {
			case 0:
				continue // no match → no-op (don't write empty strings)
			case 1:
				value = m[0] // whole match
			default:
				value = m[1] // first capture group
			}
		} else {
			res := gjson.GetBytes(responseBody, r.Path)
			if !res.Exists() {
				continue
			}
			value = res.String()
		}
		if err := bag.Set(ctx, slug, session, r.As, value); err != nil {
			return fmt.Errorf("extractor_json[%d]: write %s: %w", i, r.As, err)
		}
	}
	return nil
}
