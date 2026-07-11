// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package mwcatalog

import (
	"errors"
	"fmt"
)

// ErrInvalidRule is returned when a rule fails validation against its
// descriptor. Handlers map it to 422 MIDDLEWARE_INVALID_RULE.
var ErrInvalidRule = errors.New("invalid middleware rule")

// Validate checks a rule against a catalogued type. It returns nil for an
// uncatalogued type (the advanced escape hatch — passed through to Goma as-is).
// A catalogued type is checked for required keys, value types, and unknown keys,
// so a malformed rule is rejected before it can render into the workspace file
// and take Goma — and every route in the workspace — offline.
func Validate(mwType string, rule map[string]any) error {
	d, ok := Get(mwType)
	if !ok {
		return nil // uncatalogued: advanced passthrough
	}
	allowed := make(map[string]Field, len(d.Fields))
	for _, f := range d.Fields {
		allowed[f.Key] = f
	}
	for k := range rule {
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("%w: unknown field %q for type %q", ErrInvalidRule, k, mwType)
		}
	}
	for _, f := range d.Fields {
		v, present := rule[f.Key]
		if !present || v == nil {
			if f.Required {
				return fmt.Errorf("%w: %q is required", ErrInvalidRule, f.Key)
			}
			continue
		}
		if err := validateField(f, v); err != nil {
			return err
		}
	}
	return nil
}

func validateField(f Field, v any) error {
	switch f.Type {
	case FieldString, FieldDuration:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("%w: %q must be a string", ErrInvalidRule, f.Key)
		}
	case FieldBool:
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%w: %q must be true or false", ErrInvalidRule, f.Key)
		}
	case FieldInt:
		if !isInt(v) {
			return fmt.Errorf("%w: %q must be a whole number", ErrInvalidRule, f.Key)
		}
	case FieldEnum:
		s, ok := v.(string)
		if !ok || !contains(f.Options, s) {
			return fmt.Errorf("%w: %q must be one of %v", ErrInvalidRule, f.Key, f.Options)
		}
	case FieldStrings:
		if !isStringSlice(v) {
			return fmt.Errorf("%w: %q must be a list of strings", ErrInvalidRule, f.Key)
		}
	case FieldUsers:
		if err := validateUsers(f.Key, v); err != nil {
			return err
		}
	case FieldObject:
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("%w: %q must be an object", ErrInvalidRule, f.Key)
		}
	}
	return nil
}

// validateUsers checks the basicAuth users list: a non-empty array of objects,
// each with a username and password.
func validateUsers(key string, v any) error {
	list, ok := v.([]any)
	if !ok || len(list) == 0 {
		return fmt.Errorf("%w: %q must be a non-empty list of users", ErrInvalidRule, key)
	}
	for i, e := range list {
		u, ok := e.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: user %d must be an object with username and password", ErrInvalidRule, i+1)
		}
		if s, _ := u["username"].(string); s == "" {
			return fmt.Errorf("%w: user %d is missing a username", ErrInvalidRule, i+1)
		}
		if s, _ := u["password"].(string); s == "" {
			return fmt.Errorf("%w: user %d is missing a password", ErrInvalidRule, i+1)
		}
	}
	return nil
}

// isInt accepts JSON numbers (float64) with no fractional part, as well as the
// native int kinds, since rules round-trip through JSON.
func isInt(v any) bool {
	switch n := v.(type) {
	case float64:
		return n == float64(int64(n))
	case int, int32, int64:
		return true
	default:
		return false
	}
}

func isStringSlice(v any) bool {
	list, ok := v.([]any)
	if !ok {
		return false
	}
	for _, e := range list {
		if _, ok := e.(string); !ok {
			return false
		}
	}
	return true
}

func contains(opts []string, s string) bool {
	for _, o := range opts {
		if o == s {
			return true
		}
	}
	return false
}
